package repo

const (
	Completed Status = "completed"
	Failed    Status = "failed"
	Queued    Status = "queued"
	Running   Status = "running"
)

type Status string

type JobDTO struct {
	ID             string
	Once           *string
	Interval       *string
	Status         Status
	CreatedAt      int64
	LastFinishedAt int64
	Payload        map[string]any
}

type ExecutionDTO struct {
	ID         string
	JobID      string
	WorkerID   string
	Status     string
	StartedAt  int64
	FinishedAt int64
}
