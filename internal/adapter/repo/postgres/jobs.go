package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"scheduler/internal/port/repo"

	"github.com/jackc/pgx/v5/pgxpool"
)

//var _ repo.Jobs = (*JobsRepo)(nil)

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

	query := `
		INSERT INTO jobs (id, once, interval, status, created_at, last_finished_at, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = r.db.Exec(ctx, query,
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
	query := `
		SELECT id, once, interval, status, created_at, last_finished_at, payload
		FROM jobs
		WHERE id = $1
	`
	var job repo.JobDTO
	var payloadBytes []byte

	err := r.db.QueryRow(ctx, query, jobID).Scan(
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
	res, err := r.db.Exec(ctx, `DELETE FROM jobs WHERE id = $1`, jobID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *JobsRepo) List(ctx context.Context, status *string) ([]repo.JobDTO, error) {
	query := `
		SELECT id, once, interval, status, created_at, last_finished_at, payload
		FROM jobs
	`
	var rows pgx.Rows
	var err error

	if status != nil {
		query += ` WHERE status = $1`
		rows, err = r.db.Query(ctx, query, *status)
	} else {
		rows, err = r.db.Query(ctx, query)
	}

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
		if err := json.Unmarshal(payloadBytes, &j.Payload); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}

	return jobs, nil
}

func (r *JobsRepo) ListExecutions(ctx context.Context, jobID string, workerID *string) ([]repo.ExecutionDTO, error) {
	query := `
		SELECT id, job_id, worker_id, status, started_at, finished_at
		FROM executions
		WHERE job_id = $1
	`
	var rows pgx.Rows
	var err error

	if workerID != nil {
		query += ` AND worker_id = $2`
		rows, err = r.db.Query(ctx, query, jobID, *workerID)
	} else {
		rows, err = r.db.Query(ctx, query, jobID)
	}

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
	return execs, nil
}
