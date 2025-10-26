package entity

type Execution struct {
	FinishedAt int64  `json:"finishedAt,omitempty"`
	Id         string `json:"id,omitempty"`
	JobId      string `json:"jobId,omitempty"`
	StartedAt  int64  `json:"startedAt,omitempty"`
	Status     string `json:"status,omitempty"`
	WorkerId   string `json:"workerId,omitempty"`
}
