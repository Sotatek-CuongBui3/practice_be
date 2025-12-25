package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/cuongbtq/practice-be/internal/worker/domain"
)

// spawnWorkerPool spawns N worker goroutines based on concurrency configuration
func (w *Worker) spawnWorkerPool(ctx context.Context) {
	w.logger.Info("Spawning worker pool",
		slog.Int("concurrency", w.concurrency),
		slog.String("worker_id", w.workerID),
	)

	for i := 0; i < w.concurrency; i++ {
		w.wg.Add(1)
		go w.workerLoop(ctx, i)
	}

	w.logger.Info("Worker pool spawned successfully",
		slog.Int("worker_count", w.concurrency),
	)
}

// workerLoop is the main processing loop for each worker goroutine
func (w *Worker) workerLoop(ctx context.Context, workerNum int) {
	defer w.wg.Done()

	workerName := fmt.Sprintf("%s-%d", w.workerID, workerNum)
	w.logger.Info("Worker goroutine started",
		slog.String("worker_name", workerName),
		slog.Int("worker_num", workerNum),
	)

	for {
		select {
		case <-w.stopChan:
			w.logger.Info("Worker goroutine stopping - stopChan closed",
				slog.String("worker_name", workerName),
			)
			return

		case <-ctx.Done():
			w.logger.Info("Worker goroutine stopping - context canceled",
				slog.String("worker_name", workerName),
			)
			return

		case msg, ok := <-w.jobsChan:
			if !ok {
				w.logger.Info("Worker goroutine stopping - jobsChan closed",
					slog.String("worker_name", workerName),
				)
				return
			}

			w.logger.Info("Worker received job",
				slog.String("worker_name", workerName),
				slog.String("job_id", msg.JobID),
				slog.Uint64("delivery_tag", msg.DeliveryTag),
			)

			// Process the job
			err := w.processJob(ctx, msg)

			// Get RabbitMQ channel for ACK/NACK
			channel := w.rabbitClient.GetChannel()
			if channel == nil {
				w.logger.Error("Failed to get RabbitMQ channel for ACK/NACK",
					slog.String("worker_name", workerName),
					slog.String("job_id", msg.JobID),
				)
				continue
			}

			// ACK or NACK based on processing result
			if err != nil {
				w.logger.Error("Job processing failed",
					slog.String("worker_name", workerName),
					slog.String("job_id", msg.JobID),
					slog.String("error", err.Error()),
				)

				// Smart requeue decision based on error type
				requeue := w.shouldRequeueJob(err)

				if nackErr := channel.Nack(msg.DeliveryTag, false, requeue); nackErr != nil {
					w.logger.Error("Failed to NACK message",
						slog.String("worker_name", workerName),
						slog.String("job_id", msg.JobID),
						slog.String("error", nackErr.Error()),
					)
				} else {
					w.logger.Info("Message NACKed",
						slog.String("worker_name", workerName),
						slog.String("job_id", msg.JobID),
						slog.Bool("requeue", requeue),
					)
				}
			} else {
				// Job completed successfully - ACK the message
				if ackErr := channel.Ack(msg.DeliveryTag, false); ackErr != nil {
					w.logger.Error("Failed to ACK message",
						slog.String("worker_name", workerName),
						slog.String("job_id", msg.JobID),
						slog.String("error", ackErr.Error()),
					)
				} else {
					w.logger.Info("Job completed successfully",
						slog.String("worker_name", workerName),
						slog.String("job_id", msg.JobID),
					)
				}
			}
		}
	}
}

// shouldRequeueJob determines if a job should be requeued based on the error type
func (w *Worker) shouldRequeueJob(err error) bool {
	// Don't requeue if job already claimed by another worker
	if errors.Is(err, domain.ErrJobAlreadyClaimed) {
		return false
	}

	// Don't requeue if max retries exceeded
	if errors.Is(err, domain.ErrMaxRetriesExceeded) {
		return false
	}

	// Don't requeue if invalid payload
	if errors.Is(err, domain.ErrInvalidPayload) {
		return false
	}

	// Requeue for transient/retryable errors
	var retryableErr *domain.RetryableError
	if errors.As(err, &retryableErr) {
		return true
	}

	// Default: don't requeue for unknown errors
	return false
}
