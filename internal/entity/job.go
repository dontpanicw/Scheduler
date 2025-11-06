package entity

const (
	Completed Status = "completed"
	Failed    Status = "failed"
	Queued    Status = "queued"
	Running   Status = "running"
)

type Status string

type Job struct {
	CreatedAt      int64                  `json:"createdAt"`
	ID             string                 `json:"id"`
	Interval       string                 `json:"interval,omitempty"`
	LastFinishedAt int64                  `json:"lastFinishedAt"`
	Once           string                 `json:"once,omitempty"`
	Payload        map[string]interface{} `json:"payload"`
	Status         Status                 `json:"status"`
}
