package domain

// Job represents a job from the database for worker processing
type Job struct {
	JobID          string
	JobType        string
	Payload        string // JSON string
	Status         string
	WorkerID       string
	RetryCount     int
	MaxRetries     int
	TimeoutSeconds int
}

// JobMessage represents a job message from RabbitMQ
type JobMessage struct {
	JobID       string `json:"job_id"`
	DeliveryTag uint64 `json:"-"`
}
