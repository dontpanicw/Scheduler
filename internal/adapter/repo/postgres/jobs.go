package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"scheduler/internal/port/repo"

	"github.com/jackc/pgx/v5/pgxpool"
)

//var _ repo.Jobs = (*JobsRepo)(nil)

const (
	createQuery = `
		INSERT INTO jobs (id, once, interval, status, created_at, last_finished_at, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	readQuery = `
		SELECT id, once, interval, status, created_at, last_finished_at, payload
		FROM jobs
		WHERE id = $1
	`
	deleteQuery = `DELETE FROM jobs WHERE id = $1`
)

type JobsRepo struct {
	db *pgxpool.Pool
}

func NewJobsRepo(connString string) (*JobsRepo, error) {
	db, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		return nil, err
	}
	return &JobsRepo{db: db}, nil
}

func (r *JobsRepo) Close() {
	r.db.Close()
}

func (r *JobsRepo) Create(ctx context.Context, job *repo.JobDTO) error {
	payloadBytes, err := json.Marshal(job.Payload)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, createQuery,
		job.ID,
		job.Once,
		job.Interval,
		job.Status,
		job.CreatedAt,
		job.LastFinishedAt,
		payloadBytes,
	)
	return err
}

func (r *JobsRepo) Read(ctx context.Context, jobID string) (*repo.JobDTO, error) {
	var job repo.JobDTO
	var payloadBytes []byte

	err := r.db.QueryRow(ctx, readQuery, jobID).Scan(
		&job.ID,
		&job.Once,
		&job.Interval,
		&job.Status,
		&job.CreatedAt,
		&job.LastFinishedAt,
		&payloadBytes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repo.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Unmarshal payload from JSONB
	if err := json.Unmarshal(payloadBytes, &job.Payload); err != nil {
		return nil, err
	}

	return &job, nil
}

func (r *JobsRepo) Delete(ctx context.Context, jobID string) error {
	res, err := r.db.Exec(ctx, deleteQuery, jobID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *JobsRepo) List(ctx context.Context, status *string) ([]repo.JobDTO, error) {
	var rows pgx.Rows
	var err error
	qb := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).
		Select("id", "once", "interval", "status", "created_at", "last_finished_at", "payload").
		From("jobs")

	if status != nil {
		qb = qb.Where(squirrel.Eq{"status": *status})
	}

	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err = r.db.Query(ctx, sql, args...)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []repo.JobDTO

	for rows.Next() {
		var j repo.JobDTO
		var payloadBytes []byte
		if err := rows.Scan(
			&j.ID,
			&j.Once,
			&j.Interval,
			&j.Status,
			&j.CreatedAt,
			&j.LastFinishedAt,
			&payloadBytes,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (r *JobsRepo) ListExecutions(ctx context.Context, jobID string, workerID *string) ([]repo.ExecutionDTO, error) {
	qb := squirrel.Select(
		"id", "job_id", "worker_id", "status", "started_at", "finished_at",
	).From("executions").
		Where(squirrel.Eq{"job_id": jobID}).
		PlaceholderFormat(squirrel.Dollar) // $1, $2 для PostgreSQL

	// Добавляем фильтр по worker_id, только если передан
	if workerID != nil {
		qb = qb.Where(squirrel.Eq{"worker_id": *workerID})
	}

	// Генерируем SQL и аргументы
	sql, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	// Выполняем запрос
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var execs []repo.ExecutionDTO
	for rows.Next() {
		var e repo.ExecutionDTO
		if err := rows.Scan(
			&e.ID,
			&e.JobID,
			&e.WorkerID,
			&e.Status,
			&e.StartedAt,
			&e.FinishedAt,
		); err != nil {
			return nil, err
		}
		execs = append(execs, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return execs, nil
}
