package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"scheduler/internal/entity"
	"scheduler/internal/port/repo"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgxpool"
)

var _ repo.Jobs = (*JobsRepo)(nil)

const (
	createQuery = `
		INSERT INTO jobs (id, kind, status, interval_seconds, once_timestamp, last_finished_at, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	readQuery = `
		SELECT id, kind, status, interval_seconds, once_timestamp, last_finished_at, payload
		FROM jobs
		WHERE id = $1
	`
	deleteQuery = `DELETE FROM jobs WHERE id = $1`
	listQuery   = `
		SELECT id, kind, status, interval_seconds, once_timestamp, last_finished_at, payload
		FROM jobs
		ORDER BY id
	`
	upsertQuery = `
		INSERT INTO jobs (id, kind, status, interval_seconds, once_timestamp, last_finished_at, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			interval_seconds = EXCLUDED.interval_seconds,
			last_finished_at = EXCLUDED.last_finished_at
	`
)

type JobsRepo struct {
	pool *pgxpool.Pool
}

func NewJobsRepo(pool *pgxpool.Pool) *JobsRepo {
	return &JobsRepo{
		pool: pool,
	}
}

func (r *JobsRepo) Create(ctx context.Context, job *entity.Job) error {
	payloadBytes, err := json.Marshal(job.Payload)
	if err != nil {
		return err
	}

	var intervalSeconds *int64
	if job.Interval != nil {
		seconds := int64(job.Interval.Seconds())
		intervalSeconds = &seconds
	}

	_, err = r.pool.Exec(ctx, createQuery,
		job.ID,
		int(job.Kind),
		string(job.Status),
		intervalSeconds,
		job.Once,
		job.LastFinishedAt,
		payloadBytes,
	)
	return err
}

func (r *JobsRepo) Read(ctx context.Context, jobID string) (*entity.Job, error) {

	var (
		id              string
		kind            int
		status          string
		intervalSeconds sql.NullInt64
		onceTimestamp   sql.NullInt64
		lastFinishedAt  int64
		payloadJSON     []byte
	)

	err := r.pool.QueryRow(ctx, readQuery, jobID).Scan(
		&id,
		&kind,
		&status,
		&intervalSeconds,
		&onceTimestamp,
		&lastFinishedAt,
		&payloadJSON,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repo.ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read job: %w", err)
	}

	var payload any
	if len(payloadJSON) > 0 {
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			return nil, fmt.Errorf("unmarshal payload: %w", err)
		}
	}

	var interval *time.Duration
	if intervalSeconds.Valid {
		dur := time.Duration(intervalSeconds.Int64) * time.Second
		interval = &dur
	}

	var once *int64
	if onceTimestamp.Valid {
		once = &onceTimestamp.Int64
	}

	return &entity.Job{
		ID:             id,
		Kind:           entity.JobKind(kind),
		Status:         entity.JobStatus(status),
		Interval:       interval,
		Once:           once,
		LastFinishedAt: lastFinishedAt,
		Payload:        payload,
	}, nil
}

func (r *JobsRepo) List(ctx context.Context) ([]*entity.Job, error) {

	rows, err := r.pool.Query(ctx, listQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*entity.Job
	for rows.Next() {
		var (
			id              string
			kind            int
			status          string
			intervalSeconds sql.NullInt64
			onceTimestamp   sql.NullInt64
			lastFinishedAt  int64
			payloadJSON     []byte
		)

		if err := rows.Scan(
			&id,
			&kind,
			&status,
			&intervalSeconds,
			&onceTimestamp,
			&lastFinishedAt,
			&payloadJSON,
		); err != nil {
			return nil, err
		}

		var payload any
		if len(payloadJSON) > 0 {
			if err := json.Unmarshal(payloadJSON, &payload); err != nil {
				return nil, fmt.Errorf("unmarshal payload: %w", err)
			}
		}

		var interval *time.Duration
		if intervalSeconds.Valid {
			dur := time.Duration(intervalSeconds.Int64) * time.Second
			interval = &dur
		}

		var once *int64
		if onceTimestamp.Valid {
			once = &onceTimestamp.Int64
		}

		jobs = append(jobs, &entity.Job{
			ID:             id,
			Kind:           entity.JobKind(kind),
			Status:         entity.JobStatus(status),
			Interval:       interval,
			Once:           once,
			LastFinishedAt: lastFinishedAt,
			Payload:        payload,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return jobs, nil
}

func (r *JobsRepo) Upsert(ctx context.Context, jobs []*entity.Job) error {
	if len(jobs) == 0 {
		return nil
	}

	for _, job := range jobs {
		payloadJSON, err := json.Marshal(job.Payload)
		if err != nil {
			return err
		}

		var intervalSeconds *int64
		if job.Interval != nil {
			seconds := int64(job.Interval.Seconds())
			intervalSeconds = &seconds
		}

		_, err = r.pool.Exec(ctx, upsertQuery,
			job.ID,
			int(job.Kind),
			string(job.Status),
			intervalSeconds,
			job.Once,
			job.LastFinishedAt,
			payloadJSON,
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (r *JobsRepo) Delete(ctx context.Context, jobID string) error {
	result, err := r.pool.Exec(ctx, deleteQuery, jobID)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repo.ErrJobNotFound
	}

	return nil
}
