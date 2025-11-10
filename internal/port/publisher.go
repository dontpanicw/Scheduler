package port

import (
	"context"
	"scheduler/internal/entity"
)

type JobPublisher interface {
	Publish(ctx context.Context, job *entity.Job) error
}
