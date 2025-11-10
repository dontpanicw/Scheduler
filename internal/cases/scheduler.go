package cases

import (
	"context"
	"errors"
	"fmt"
	"scheduler/internal/entity"
	"scheduler/internal/port"
	"scheduler/internal/port/repo"
	"sync"

	"go.uber.org/zap"

	//"scheduler/pkg/utils/pointers"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrInvalidJob = errors.New("invalid job data")
)

type SchedulerCase struct {
	jobsRepo  repo.Jobs
	running   map[string]*entity.RunningJob
	publisher port.JobPublisher
	interval  time.Duration
	mx        sync.Mutex
	logger    *zap.Logger
}

func NewSchedulerCase(
	jobsRepo repo.Jobs,
	publisher port.JobPublisher,
	interval time.Duration,
	logger *zap.Logger,
) *SchedulerCase {
	return &SchedulerCase{
		jobsRepo:  jobsRepo,
		running:   make(map[string]*entity.RunningJob),
		publisher: publisher,
		interval:  interval,
		logger:    logger,
	}
}

// Create creates a new job and returns its ID.
func (r *SchedulerCase) Create(ctx context.Context, job *entity.Job) (string, error) {
	job.ID = uuid.NewString()
	job.Status = entity.JobStatusQueued // Default status for new jobs

	if err := r.jobsRepo.Create(ctx, job); err != nil {
		return "", err
	}

	return job.ID, nil
}

// Read retrieves a job by ID.
func (r *SchedulerCase) Read(ctx context.Context, jobID string) (*entity.Job, error) {
	job, err := r.jobsRepo.Read(ctx, jobID)
	if err != nil {
		if err == repo.ErrJobNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read job: %w", err)
	}
	return job, nil
}

// List retrieves all jobs, optionally filtered by status.
func (r *SchedulerCase) List(ctx context.Context, status *string) ([]*entity.Job, error) {
	jobs, err := r.jobsRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}

	// Filter by status if provided
	if status != nil {
		var filtered []*entity.Job
		for _, job := range jobs {
			if string(job.Status) == *status {
				filtered = append(filtered, job)
			}
		}
		return filtered, nil
	}

	return jobs, nil
}

// Delete removes a job by ID.
func (r *SchedulerCase) Delete(ctx context.Context, jobID string) error {
	// Remove from running jobs if it's running
	r.mx.Lock()
	if runningJob, ok := r.running[jobID]; ok {
		runningJob.Cancel()
		delete(r.running, jobID)
	}
	r.mx.Unlock()

	// Delete from repository
	if err := r.jobsRepo.Delete(ctx, jobID); err != nil {
		if err == repo.ErrJobNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("delete job: %w", err)
	}

	return nil
}

func (r *SchedulerCase) Start(ctx context.Context) error {
	for {
		select {
		case <-time.NewTicker(r.interval).C:
			if err := r.tick(ctx); err != nil {

			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (r *SchedulerCase) tick(ctx context.Context) error {
	jobs, err := r.jobsRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list jobs: %w", err)
	}

	repoJobs := make(map[string]*entity.Job, len(jobs))
	for _, j := range jobs {
		repoJobs[j.ID] = j
	}

	r.mx.Lock()
	for jobID, j := range r.running {
		if _, ok := repoJobs[jobID]; !ok {
			r.logger.Debug("stop deleted job", zap.String("job_id", jobID))
			j.Cancel()
			delete(r.running, jobID)
		}
	}
	r.mx.Unlock()

	now := time.Now().UnixMilli()
	var updates []*entity.Job

	for _, j := range jobs {
		r.mx.Lock()
		_, isRunning := r.running[j.ID]
		r.mx.Unlock()

		if isRunning {
			r.logger.Debug("skip already running job", zap.String("job_id", j.ID))
			continue
		}

		shouldRun := false
		if j.Kind == entity.JobKindInterval {
			if j.Interval != nil {
				intervalMs := j.Interval.Milliseconds()
				if j.LastFinishedAt == 0 || now >= j.LastFinishedAt+intervalMs {
					shouldRun = true
				}
			}
		} else if j.Kind == entity.JobKindOnce {
			if j.Once != nil && j.LastFinishedAt == 0 && now >= *j.Once {
				shouldRun = true
			}
		}

		if shouldRun {
			j.Status = entity.JobStatusQueued
			updates = append(updates, j)
			go r.runJob(ctx, j)
		}
	}
	if len(updates) > 0 {
		if err := r.jobsRepo.Upsert(ctx, updates); err != nil {
			return fmt.Errorf("upsert started jobs: %w", err)
		}
	}

	return nil
}

func (r *SchedulerCase) runJob(ctx context.Context, j *entity.Job) {
	ctx, cancel := context.WithCancel(ctx)

	r.mx.Lock()
	r.running[j.ID] = &entity.RunningJob{
		Job:    j,
		Cancel: cancel,
	}
	r.mx.Unlock()

	if r.publisher != nil {
		if err := r.publisher.Publish(ctx, j); err != nil {
			r.logger.Error("publish job", zap.Error(err), zap.String("job_id", j.ID))
		}
	}
}

// HandleJobCompletion handles job completion messages from workers
func (r *SchedulerCase) HandleJobCompletion(ctx context.Context, jobID string, status string, finishedAt int64) error {
	// Read the job from repository
	job, err := r.jobsRepo.Read(ctx, jobID)
	if err != nil {
		return fmt.Errorf("read job: %w", err)
	}

	switch status {
	case entity.JobStatusCompleted:
		job.Status = entity.JobStatusCompleted
	case entity.JobStatusFailed:
		job.Status = entity.JobStatusFailed
	default:
		return fmt.Errorf("unknown status: %s", status)
	}

	job.LastFinishedAt = finishedAt

	r.mx.Lock()
	defer r.mx.Unlock()

	if runningJob, ok := r.running[jobID]; ok {
		runningJob.Cancel()
		delete(r.running, jobID)
	}

	// Update job in repository
	if err := r.jobsRepo.Upsert(ctx, []*entity.Job{job}); err != nil {
		return fmt.Errorf("upsert job: %w", err)
	}

	r.logger.Info("Job completion handled",
		zap.String("job_id", jobID),
		zap.String("status", status),
		zap.Int64("finished_at", finishedAt))

	return nil
}
