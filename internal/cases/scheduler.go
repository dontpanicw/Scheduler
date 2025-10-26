package cases

import (
	"context"
	"errors"
	"scheduler/internal/entity"
	"scheduler/internal/port/repo"
	"time"

	"github.com/google/uuid"
)

type JobsRepo interface {
	Create(ctx context.Context, job *repo.JobDTO) error
	Read(ctx context.Context, jobID string) (*repo.JobDTO, error)
	Delete(ctx context.Context, jobID string) error
	List(ctx context.Context, status *string) ([]repo.JobDTO, error)
	ListExecutions(ctx context.Context, jobID string, workerID *string) ([]repo.ExecutionDTO, error)
}

var (
	ErrNotFound   = errors.New("not found")
	ErrInvalidJob = errors.New("invalid job data")
)

type SchedulerCase struct {
	jobsRepo JobsRepo
}

func NewSchedulerCase(jobsRepo JobsRepo) *SchedulerCase {
	return &SchedulerCase{
		jobsRepo: jobsRepo,
	}
}

// Create creates a new job and returns its ID.
func (r *SchedulerCase) Create(ctx context.Context, job *entity.Job) (string, error) {
	job.ID = uuid.NewString()
	job.CreatedAt = time.Now().Unix()
	job.Status = "queued" // Default status for new jobs

	// Convert entity.Job to repo.JobDTO
	jobDTO := &repo.JobDTO{
		ID:             job.ID,
		Once:           &job.Once,
		Interval:       &job.Interval,
		Status:         repo.Status(job.Status),
		CreatedAt:      job.CreatedAt,
		LastFinishedAt: job.LastFinishedAt,
		Payload:        job.Payload,
	}

	// Save to repository
	if err := r.jobsRepo.Create(ctx, jobDTO); err != nil {
		return "", err
	}

	return job.ID, nil
}

// GetOneByID retrieves a job by its ID.
func (r *SchedulerCase) GetOneByID(ctx context.Context, jobID string) (entity.Job, error) {
	if jobID == "" {
		return entity.Job{}, ErrInvalidJob
	}

	// Get job from repository
	jobDTO, err := r.jobsRepo.Read(ctx, jobID)
	if err != nil {
		if err == repo.ErrNotFound {
			return entity.Job{}, ErrNotFound
		}
		return entity.Job{}, err
	}

	// Convert repo.JobDTO to entity.Job
	return entity.Job{
		ID:             jobDTO.ID,
		Once:           *jobDTO.Once,
		Interval:       *jobDTO.Interval,
		Status:         entity.Status(jobDTO.Status),
		CreatedAt:      jobDTO.CreatedAt,
		LastFinishedAt: jobDTO.LastFinishedAt,
		Payload:        jobDTO.Payload,
	}, nil
}

func (r *SchedulerCase) Delete(ctx context.Context, jobID string) error {
	if jobID == "" {
		return ErrInvalidJob
	}
	if err := r.jobsRepo.Delete(ctx, jobID); err != nil {
		if err == repo.ErrNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (r *SchedulerCase) List(ctx context.Context, status *string) ([]entity.Job, error) {
	jobsDTO, err := r.jobsRepo.List(ctx, status)
	if err != nil {
		return nil, err
	}

	jobs := make([]entity.Job, 0, len(jobsDTO))
	for _, j := range jobsDTO {
		jobs = append(jobs, dtoToEntity(&j))
	}
	return jobs, nil
}

func (r *SchedulerCase) ListExecutions(ctx context.Context, jobID string, workerID *string) ([]entity.Execution, error) {
	if jobID == "" {
		return nil, ErrInvalidJob
	}

	execDTOs, err := r.jobsRepo.ListExecutions(ctx, jobID, workerID)
	if err != nil {
		return nil, err
	}

	execs := make([]entity.Execution, 0, len(execDTOs))
	for _, e := range execDTOs {
		execs = append(execs, entity.Execution{
			Id:         e.ID,
			JobId:      e.JobID,
			WorkerId:   e.WorkerID,
			Status:     e.Status,
			StartedAt:  e.StartedAt,
			FinishedAt: e.FinishedAt,
		})
	}
	return execs, nil
}

func dtoToEntity(j *repo.JobDTO) entity.Job {
	return entity.Job{
		ID:             j.ID,
		Once:           deref(j.Once),
		Interval:       deref(j.Interval),
		Status:         entity.Status(j.Status),
		CreatedAt:      j.CreatedAt,
		LastFinishedAt: j.LastFinishedAt,
		Payload:        j.Payload,
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
