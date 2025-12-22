package model

import "time"

type Job struct {
	JobID          string    `db:"job_id"`
	IdempotencyKey string    `db:"idempotency_key"`
	UserID         string    `db:"user_id"`
	JobType        string    `db:"job_type"`
	Payload        string    `db:"payload"`
	Status         string    `db:"status"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}
