package handler

import (
	"fmt"
	"scheduler/internal/entity"
	"scheduler/internal/input/http/gen"
	"strconv"
	"time"
)

// toEntityJob преобразует сгенерированную структуру в сущность для бизнес-логики
func toEntityJob(job *gen.JobCreate) (*entity.Job, error) {
	entityJob := new(entity.Job)

	if job.Interval != nil {
		interval, err := time.ParseDuration(*job.Interval)
		if err != nil {
			return nil, fmt.Errorf("parse interval duraton: %w", err)
		}
		entityJob.Interval = &interval
		entityJob.Kind = entity.JobKindInterval
	}

	if job.Once != nil {
		onceTimestamp, err := strconv.ParseInt(*job.Once, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse once timestamp: %w", err)
		}
		entityJob.Once = &onceTimestamp
		entityJob.Kind = entity.JobKindOnce
	}

	if job.Payload != nil {
		entityJob.Payload = *job.Payload
	}

	return entityJob, nil
}

// toGenJob преобразует сущность в сгенерированную структуру для HTTP ответа
func toGenJob(job *entity.Job) gen.Job {
	genJob := gen.Job{
		Id:             job.ID,
		Status:         gen.Status(job.Status),
		LastFinishedAt: job.LastFinishedAt,
		CreatedAt:      0, // TODO: добавить CreatedAt в entity.Job
	}

	// Конвертируем Interval в строку
	if job.Interval != nil {
		intervalStr := job.Interval.String()
		genJob.Interval = &intervalStr
	}

	// Конвертируем Once в строку
	if job.Once != nil {
		onceStr := strconv.FormatInt(*job.Once, 10)
		genJob.Once = &onceStr
	}

	// Конвертируем Payload
	if job.Payload != nil {
		if payloadMap, ok := job.Payload.(map[string]interface{}); ok {
			genJob.Payload = payloadMap
		} else {
			// Если payload не map, создаем пустой map
			genJob.Payload = make(map[string]interface{})
		}
	} else {
		genJob.Payload = make(map[string]interface{})
	}

	return genJob
}
