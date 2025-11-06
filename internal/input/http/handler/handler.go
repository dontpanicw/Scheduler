package handler

import (
	"context"
	"scheduler/internal/cases"
	"scheduler/internal/entity"
	"scheduler/internal/input/http/gen"
)

var _ gen.StrictServerInterface = (*Handler)(nil)

type JobsCases interface {
	Create(ctx context.Context, job *entity.Job) (string, error)
	GetOneByID(ctx context.Context, jobID string) (entity.Job, error)
	Delete(ctx context.Context, jobID string) error
	List(ctx context.Context, status *string) ([]entity.Job, error)
	ListExecutions(ctx context.Context, jobID string, workerID *string) ([]entity.Execution, error)
}

type Handler struct {
	schedulerCase JobsCases
}

func NewHandler(schCase JobsCases) *Handler {
	return &Handler{
		schedulerCase: schCase,
	}
}

// Create a new job
// (POST /jobs)
func (r *Handler) PostJobs(ctx context.Context, request gen.PostJobsRequestObject) (gen.PostJobsResponseObject, error) {
	if request.Body == nil {
		return gen.PostJobs400Response{}, nil
	}
	jobID, err := r.schedulerCase.Create(ctx, toEntityJob(request.Body))
	if err != nil {
		return nil, err // 500
	}
	return gen.PostJobs201JSONResponse(jobID), nil
}

// List jobs
// (GET /jobs)
func (r *Handler) GetJobs(ctx context.Context, request gen.GetJobsRequestObject) (gen.GetJobsResponseObject, error) {
	// Извлекаем параметр status, если он есть
	var status *string
	if request.Params.Status != nil {
		s := string(*request.Params.Status)
		status = &s
	}

	// Получаем список заданий с фильтрацией по статусу
	jobs, err := r.schedulerCase.List(ctx, status)
	if err != nil {
		return nil, err // 500
	}

	// Преобразуем сущности в сгенерированные структуры
	response := make([]gen.Job, len(jobs))
	for i, job := range jobs {
		response[i] = gen.Job{
			Id:             job.ID,
			Once:           &job.Once,
			Interval:       &job.Interval,
			Status:         gen.Status(job.Status),
			CreatedAt:      job.CreatedAt,
			LastFinishedAt: job.LastFinishedAt,
			Payload:        job.Payload,
		}
	}

	return gen.GetJobs200JSONResponse(response), nil
}

// Delete a job
// (DELETE /jobs/{job_id})
func (r *Handler) DeleteJobsJobId(ctx context.Context, request gen.DeleteJobsJobIdRequestObject) (gen.DeleteJobsJobIdResponseObject, error) {
	err := r.schedulerCase.Delete(ctx, request.JobId)
	if err != nil {
		if err == cases.ErrNotFound {
			return gen.DeleteJobsJobId404Response{}, nil
		}
		return nil, err // 500
	}
	return gen.DeleteJobsJobId204Response{}, nil
}

// Get job details
// (GET /jobs/{job_id})
func (r *Handler) GetJobsJobId(ctx context.Context, request gen.GetJobsJobIdRequestObject) (gen.GetJobsJobIdResponseObject, error) {
	job, err := r.schedulerCase.GetOneByID(ctx, request.JobId)
	if err != nil {
		if err == cases.ErrNotFound {
			return gen.GetJobsJobId404Response{}, nil
		}
		return nil, err // 500
	}

	response := gen.Job{
		Id:             job.ID,
		Once:           &job.Once,
		Interval:       &job.Interval,
		Status:         gen.Status(job.Status),
		CreatedAt:      job.CreatedAt,
		LastFinishedAt: job.LastFinishedAt,
		Payload:        job.Payload,
	}

	return gen.GetJobsJobId200JSONResponse(response), nil
}

// Get job executions
// (GET /jobs/{job_id}/executions)
func (r *Handler) GetJobsJobIdExecutions(ctx context.Context, request gen.GetJobsJobIdExecutionsRequestObject) (gen.GetJobsJobIdExecutionsResponseObject, error) {
	// Извлекаем параметр worker_id, если он есть
	var workerID *string
	if request.Params.WorkerId != nil {
		workerID = request.Params.WorkerId
	}

	// Получаем выполнения задания
	executions, err := r.schedulerCase.ListExecutions(ctx, request.JobId, workerID)
	if err != nil {
		if err == cases.ErrNotFound {
			return gen.GetJobsJobIdExecutions404Response{}, nil
		}
		return nil, err // 500
	}

	// Преобразуем сущности в сгенерированные структуры
	response := make([]gen.Execution, len(executions))
	for i, exec := range executions {
		response[i] = gen.Execution{
			Id:         &exec.Id,
			JobId:      &exec.JobId,
			WorkerId:   &exec.WorkerId,
			Status:     &exec.Status,
			StartedAt:  &exec.StartedAt,
			FinishedAt: &exec.FinishedAt,
		}
	}

	return gen.GetJobsJobIdExecutions200JSONResponse(response), nil
}
