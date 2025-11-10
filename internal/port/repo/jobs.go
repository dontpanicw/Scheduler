package repo

import (
	"context"
	"scheduler/internal/entity"
)

type Jobs interface {
	Upsert(ctx context.Context, jobs []*entity.Job) error
	Create(ctx context.Context, job *entity.Job) error
	Read(ctx context.Context, jobID string) (*entity.Job, error)
	List(ctx context.Context) ([]*entity.Job, error)
	Delete(ctx context.Context, jobID string) error
}
