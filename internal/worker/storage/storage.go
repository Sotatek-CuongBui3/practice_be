package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/cuongbtq/practice-be/internal/worker/domain"
	"github.com/jmoiron/sqlx"
)

// Storage handles all database operations for the worker
type Storage struct {
	db     *sqlx.DB
	logger *slog.Logger
}

// NewStorage creates a new Storage instance
func NewStorage(db *sqlx.DB, logger *slog.Logger) *Storage {
	return &Storage{
		db:     db,
		logger: logger,
	}
}

// GetJobByID retrieves a job from the database by its ID
func (s *Storage) GetJobByID(ctx context.Context, jobID string) (*domain.Job, error) {
	query := `
		SELECT job_id, job_type, payload, status, worker_id, retry_count, max_retries, timeout_seconds
		FROM jobs
		WHERE job_id = $1
	`

	var job domain.Job
	var workerID sql.NullString

	err := s.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.JobID,
		&job.JobType,
		&job.Payload,
		&job.Status,
		&workerID,
		&job.RetryCount,
		&job.MaxRetries,
		&job.TimeoutSeconds,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if workerID.Valid {
		job.WorkerID = workerID.String
	}

	return &job, nil
}

// ClaimJob attempts to claim a job using optimistic locking
// Returns full job details on success, error if job is already claimed or doesn't exist
func (s *Storage) ClaimJob(ctx context.Context, jobID, workerID string) (*domain.Job, error) {
	query := `
		UPDATE jobs 
		SET status = $1, 
		    worker_id = $2,
		    started_at = NOW(),
		    last_heartbeat_at = NOW(),
		    updated_at = NOW()
		WHERE job_id = $3 
		  AND status = $4
		RETURNING job_id, job_type, payload, retry_count, max_retries, timeout_seconds
	`

	var job domain.Job
	err := s.db.QueryRowContext(ctx, query, domain.JobStatusRunning, workerID, jobID, domain.JobStatusPending).Scan(
		&job.JobID,
		&job.JobType,
		&job.Payload,
		&job.RetryCount,
		&job.MaxRetries,
		&job.TimeoutSeconds,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			s.logger.Warn("Failed to claim job - already claimed or not found",
				slog.String("job_id", jobID),
				slog.String("worker_id", workerID),
			)
			return nil, domain.ErrJobAlreadyClaimed
		}
		return nil, fmt.Errorf("failed to claim job: %w", err)
	}

	job.Status = domain.JobStatusRunning
	job.WorkerID = workerID

	s.logger.Info("Job claimed successfully",
		slog.String("job_id", jobID),
		slog.String("worker_id", workerID),
		slog.String("job_type", job.JobType),
	)

	return &job, nil
}

// UpdateJobStatus updates the job status and optionally sets result/error
func (s *Storage) UpdateJobStatus(ctx context.Context, jobID, status string, result map[string]interface{}, errorMsg string) error {
	query := `
		UPDATE jobs
		SET status = $1::text,
			result = $2,
			error_message = $3,
			completed_at = CASE 
				WHEN $1::text IN ($4::text, $5::text) THEN NOW() 
				ELSE NULL 
			END,
			updated_at = NOW()
		WHERE job_id = $6
	`

	var resultJSON []byte
	var err error
	if result != nil {
		resultJSON, err = json.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
	}

	_, err = s.db.ExecContext(ctx, query, status, resultJSON, errorMsg, domain.JobStatusCompleted, domain.JobStatusFailed, jobID)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	s.logger.Info("Job status updated",
		slog.String("job_id", jobID),
		slog.String("status", status),
	)

	return nil
}

// UpdateJobHeartbeat updates the last_heartbeat_at timestamp for a running job
func (s *Storage) UpdateJobHeartbeat(ctx context.Context, jobID string) error {
	query := `
		UPDATE jobs
		SET last_heartbeat_at = NOW(),
		    updated_at = NOW()
		WHERE job_id = $1 AND status = $2
	`

	result, err := s.db.ExecContext(ctx, query, jobID, domain.JobStatusRunning)
	if err != nil {
		return fmt.Errorf("failed to update job heartbeat: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		s.logger.Warn("Job heartbeat update - no rows affected (job may not be running)",
			slog.String("job_id", jobID),
		)
	}

	return nil
}
