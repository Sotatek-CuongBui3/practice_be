package domain

import "errors"

var (
	// ErrJobNotFound is returned when a job cannot be found in the database
	ErrJobNotFound = errors.New("job not found")

	// ErrJobAlreadyClaimed is returned when attempting to claim a job that's already claimed
	ErrJobAlreadyClaimed = errors.New("job already claimed or not in PENDING status")

	// ErrInvalidPayload is returned when job payload JSON is malformed
	ErrInvalidPayload = errors.New("invalid job payload")

	// ErrMaxRetriesExceeded is returned when a job has exceeded its retry limit
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
)

// RetryableError wraps transient errors that should trigger a requeue
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return "retryable error: " + e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NewRetryableError creates a new retryable error
func NewRetryableError(err error) error {
	return &RetryableError{Err: err}
}
