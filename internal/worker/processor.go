package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/cuongbtq/practice-be/internal/worker/domain"
)

// processJob processes a single job with timeout, heartbeat, and status updates
func (w *Worker) processJob(ctx context.Context, msg *domain.JobMessage) error {
	w.logger.Info("Processing job",
		slog.String("job_id", msg.JobID),
		slog.String("worker_id", w.workerID),
	)

	// Step 1: Claim job from database (PENDING → RUNNING)
	job, err := w.storage.ClaimJob(ctx, msg.JobID, w.workerID)
	if err != nil {
		if errors.Is(err, domain.ErrJobAlreadyClaimed) {
			// Job already claimed by another worker - don't requeue
			w.logger.Warn("Job already claimed, skipping",
				slog.String("job_id", msg.JobID),
			)
			return fmt.Errorf("job already claimed: %w", err)
		}
		// Database error - could be transient
		w.logger.Error("Failed to claim job",
			slog.String("job_id", msg.JobID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to claim job: %w", err)
	}

	// Step 2: Parse job payload (JSON unmarshal)
	var payload map[string]interface{}
	if job.Payload != "" {
		if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
			w.logger.Error("Failed to parse job payload",
				slog.String("job_id", msg.JobID),
				slog.String("error", err.Error()),
			)
			// Mark job as failed - invalid payload
			_ = w.storage.UpdateJobStatus(ctx, job.JobID, domain.JobStatusFailed, nil, fmt.Sprintf("Invalid payload JSON: %s", err.Error()))
			return fmt.Errorf("%w: %v", domain.ErrInvalidPayload, err)
		}
	}

	// Step 3: Create timeout context from job.timeout_seconds
	jobTimeout := w.jobTimeout // Default timeout
	if job.TimeoutSeconds > 0 {
		jobTimeout = time.Duration(job.TimeoutSeconds) * time.Second
	}

	jobCtx, cancel := context.WithTimeout(ctx, jobTimeout)
	defer cancel()

	// Step 4: Start heartbeat goroutine
	heartbeatDone := make(chan struct{})
	go w.sendJobHeartbeat(jobCtx, job.JobID, heartbeatDone)
	defer close(heartbeatDone) // Signal heartbeat goroutine to stop

	// Step 5: Execute job logic based on job_type
	result, err := w.executeJob(jobCtx, job, payload)

	// Step 6: Update job status (COMPLETED/FAILED)
	if err != nil {
		w.logger.Error("Job execution failed",
			slog.String("job_id", job.JobID),
			slog.String("job_type", job.JobType),
			slog.String("error", err.Error()),
		)

		// Update job status to FAILED
		if updateErr := w.storage.UpdateJobStatus(ctx, job.JobID, domain.JobStatusFailed, nil, err.Error()); updateErr != nil {
			w.logger.Error("Failed to update job status to FAILED",
				slog.String("job_id", job.JobID),
				slog.String("error", updateErr.Error()),
			)
		}

		// Step 8: Return error for NACK decision
		// Check if we should requeue based on retry count
		if job.RetryCount < job.MaxRetries {
			w.logger.Info("Job will be retried",
				slog.String("job_id", job.JobID),
				slog.Int("retry_count", job.RetryCount),
				slog.Int("max_retries", job.MaxRetries),
			)
			// Return retryable error to trigger NACK with requeue
			return domain.NewRetryableError(fmt.Errorf("job execution failed: %w", err))
		}

		w.logger.Warn("Job exceeded max retries",
			slog.String("job_id", job.JobID),
			slog.Int("retry_count", job.RetryCount),
			slog.Int("max_retries", job.MaxRetries),
		)
		// Don't requeue - exceeded max retries
		return fmt.Errorf("%w: %v", domain.ErrMaxRetriesExceeded, err)
	}

	// Job completed successfully
	w.logger.Info("Job completed successfully",
		slog.String("job_id", job.JobID),
		slog.String("job_type", job.JobType),
	)

	// Update job status to COMPLETED
	if updateErr := w.storage.UpdateJobStatus(ctx, job.JobID, domain.JobStatusCompleted, result, ""); updateErr != nil {
		w.logger.Error("Failed to update job status to COMPLETED",
			slog.String("job_id", job.JobID),
			slog.String("error", updateErr.Error()),
		)
		// Job completed but status update failed - still return success for ACK
	}

	// Step 7: Heartbeat goroutine will stop when heartbeatDone is closed (deferred above)

	return nil // Success - ACK the message
}

// sendJobHeartbeat periodically updates the job's heartbeat timestamp
func (w *Worker) sendJobHeartbeat(ctx context.Context, jobID string, done <-chan struct{}) {
	// Use heartbeat interval from config (default 30s)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	w.logger.Debug("Job heartbeat started",
		slog.String("job_id", jobID),
	)

	for {
		select {
		case <-done:
			w.logger.Debug("Job heartbeat stopped",
				slog.String("job_id", jobID),
			)
			return

		case <-ctx.Done():
			w.logger.Debug("Job heartbeat stopped - context canceled",
				slog.String("job_id", jobID),
			)
			return

		case <-ticker.C:
			if err := w.storage.UpdateJobHeartbeat(ctx, jobID); err != nil {
				w.logger.Warn("Failed to update job heartbeat",
					slog.String("job_id", jobID),
					slog.String("error", err.Error()),
				)
			} else {
				w.logger.Debug("Job heartbeat updated",
					slog.String("job_id", jobID),
				)
			}
		}
	}
}

// executeJob executes the job based on its type (placeholder for now)
func (w *Worker) executeJob(ctx context.Context, job *domain.Job, payload map[string]interface{}) (map[string]interface{}, error) {
	w.logger.Info("Executing job",
		slog.String("job_id", job.JobID),
		slog.String("job_type", job.JobType),
	)

	// TODO: Step 12 - Implement job executor pattern
	// For now, simulate job execution with a delay
	select {
	case <-time.After(2 * time.Second):
		// Job completed
		result := map[string]interface{}{
			"status":  "success",
			"message": fmt.Sprintf("Job %s of type %s completed", job.JobID, job.JobType),
		}
		return result, nil

	case <-ctx.Done():
		// Job timed out or context canceled
		return nil, fmt.Errorf("job execution canceled: %w", ctx.Err())
	}
}
