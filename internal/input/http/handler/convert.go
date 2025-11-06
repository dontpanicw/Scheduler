package handler

import (
	"scheduler/internal/entity"
	"scheduler/internal/input/http/gen"
)

// toEntityJob преобразует сгенерированную структуру в сущность для бизнес-логики
func toEntityJob(j *gen.JobCreate) *entity.Job {
	job := &entity.Job{
		Payload: *j.Payload,
	}
	if j.Once != nil {
		job.Once = *j.Once
	}
	if j.Interval != nil {
		job.Interval = *j.Interval
	}
	return job
}
