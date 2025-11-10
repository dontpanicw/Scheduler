package handler

//
import (
	"context"
	"errors"
	"fmt"
	"scheduler/internal/cases"
	"scheduler/internal/entity"
	"scheduler/internal/input/http/gen"
)

// var _ gen.StrictServerInterface = (*Handler)(nil)
type JobsCases interface {
	Create(ctx context.Context, job *entity.Job) (string, error)
	Read(ctx context.Context, jobID string) (*entity.Job, error)
	List(ctx context.Context, status *string) ([]*entity.Job, error)
	Delete(ctx context.Context, jobID string) error
	Start(ctx context.Context) error
}

type Handler struct {
	schedulerCase JobsCases
}

func NewServer(schCase JobsCases) *Handler {
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
	job, err := toEntityJob(request.Body)
	if err != nil {
		return nil, fmt.Errorf("convert to entity job failed") // 500
	}

	jobId, err := r.schedulerCase.Create(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("create job failed")
	}
	return gen.PostJobs201JSONResponse(jobId), nil
}

// List jobs
// (GET /jobs)
func (r *Handler) GetJobs(ctx context.Context, request gen.GetJobsRequestObject) (gen.GetJobsResponseObject, error) {

	var status *string
	if request.Params.Status != nil {
		s := string(*request.Params.Status)
		status = &s
	}

	jobs, err := r.schedulerCase.List(ctx, status)
	if err != nil {
		return nil, err // 500
	}

	response := make([]gen.Job, len(jobs))
	for i, job := range jobs {
		response[i] = toGenJob(job)
	}

	return gen.GetJobs200JSONResponse(response), nil
}

// Delete a job
// (DELETE /jobs/{job_id})
func (r *Handler) DeleteJobsJobId(ctx context.Context, request gen.DeleteJobsJobIdRequestObject) (gen.DeleteJobsJobIdResponseObject, error) {
	err := r.schedulerCase.Delete(ctx, request.JobId)
	if err != nil {
		if errors.Is(err, cases.ErrNotFound) {
			return gen.DeleteJobsJobId404Response{}, nil
		}
		return nil, fmt.Errorf("delete job failed: %w", err) // 500
	}
	return gen.DeleteJobsJobId204Response{}, nil
}

// Get job details
// (GET /jobs/{job_id})
func (r *Handler) GetJobsJobId(ctx context.Context, request gen.GetJobsJobIdRequestObject) (gen.GetJobsJobIdResponseObject, error) {
	job, err := r.schedulerCase.Read(ctx, request.JobId)
	if err != nil {
		if errors.Is(err, cases.ErrNotFound) {
			return gen.GetJobsJobId404Response{}, nil
		}
		return nil, fmt.Errorf("read job failed: %w", err) // 500
	}

	response := toGenJob(job)
	return gen.GetJobsJobId200JSONResponse(response), nil
}

// Get job executions
// (GET /jobs/{job_id}/executions)
func (r *Handler) GetJobsJobIdExecutions(ctx context.Context, request gen.GetJobsJobIdExecutionsRequestObject) (gen.GetJobsJobIdExecutionsResponseObject, error) {
	_, err := r.schedulerCase.Read(ctx, request.JobId)
	if err != nil {
		if errors.Is(err, cases.ErrNotFound) {
			return gen.GetJobsJobIdExecutions404Response{}, nil
		}
		return nil, fmt.Errorf("read job failed: %w", err) // 500
	}

	// Executions не реализованы, возвращаем пустой массив
	response := []gen.Execution{}
	return gen.GetJobsJobIdExecutions200JSONResponse(response), nil
}
