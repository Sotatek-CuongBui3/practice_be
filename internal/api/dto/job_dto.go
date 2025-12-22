package dto

type CreateJobRequest struct {
	IdempotencyKey string `json:"idempotency_key" binding:"required"`
	UserID         string `json:"user_id" binding:"required"`
	JobType        string `json:"job_type" binding:"required"`
	Payload        string `json:"payload" binding:"required"`
}

type ListJobsRequest struct {
	UserID   string `form:"user_id"`
	JobType  string `form:"job_type"`
	Status   string `form:"status"`
	PageSize int    `form:"page_size"`
	Cursor   string `form:"cursor"`
}

type ListJobsResponse struct {
	Jobs       []JobDTO `json:"jobs"`
	NextCursor string   `json:"next_cursor,omitempty"`
}

type JobDTO struct {
	JobID          string `json:"job_id"`
	IdempotencyKey string `json:"idempotency_key"`
	UserID         string `json:"user_id"`
	JobType        string `json:"job_type"`
	Payload        string `json:"payload"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}
