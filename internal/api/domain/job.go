package domain

import (
	"errors"
)

const (
	JobStatusPending   = "PENDING"
	JobStatusRunning   = "RUNNING"
	JobStatusCompleted = "COMPLETED"
	JobStatusFailed    = "FAILED"
	JobStatusCanceled  = "CANCELED"
)

var (
	ErrJobNotFound = errors.New("job not found")
)
