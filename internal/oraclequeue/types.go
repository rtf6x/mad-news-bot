package oraclequeue

const (
	QueueKey      = "advice:queue"
	ProcessingKey = "advice:processing"
	JobKeyPrefix  = "advice:job:"
	EventsChannel = "advice:events"

	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusDone       = "done"
	StatusFailed     = "failed"
	StatusRetrying   = "retrying"
)

type Advice struct {
	Advice     string `json:"advice"`
	HarmLevel  int    `json:"harm_level"`
	Disclaimer string `json:"disclaimer"`
	Mode       string `json:"mode"`
	Lang       string `json:"lang"`
	Topic      string `json:"topic,omitempty"`
	Source     string `json:"source,omitempty"`
}

type Job struct {
	ID      string  `json:"id"`
	Status  string  `json:"status"`
	Prompt  string  `json:"prompt"`
	Lang    string  `json:"lang"`
	Attempt int     `json:"attempt,omitempty"`
	Error   string  `json:"error,omitempty"`
	Result  *Advice `json:"result,omitempty"`
}

type Event struct {
	JobID         string  `json:"job_id"`
	Status        string  `json:"status"`
	QueuePosition int     `json:"queue_position"`
	Attempt       int     `json:"attempt,omitempty"`
	RetryJobID    string  `json:"retry_job_id,omitempty"`
	Error         string  `json:"error,omitempty"`
	Result        *Advice `json:"result,omitempty"`
}
