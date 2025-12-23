package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/cuongbtq/practice-be/shared/postgresql"
	"github.com/cuongbtq/practice-be/shared/rabbitmq"
)

// Config holds worker configuration
type Config struct {
	Logger       *slog.Logger
	DBClient     *postgresql.Client
	RabbitClient *rabbitmq.Client
	Concurrency  int
	JobTimeout   time.Duration
}

// Worker represents the background job worker
type Worker struct {
	logger       *slog.Logger
	dbClient     *postgresql.Client
	rabbitClient *rabbitmq.Client
	concurrency  int
	jobTimeout   time.Duration
	wg           sync.WaitGroup
	stopChan     chan struct{}
}

// NewWorker creates a new worker instance
func NewWorker(cfg *Config) *Worker {
	return &Worker{
		logger:       cfg.Logger,
		dbClient:     cfg.DBClient,
		rabbitClient: cfg.RabbitClient,
		concurrency:  cfg.Concurrency,
		jobTimeout:   cfg.JobTimeout,
		stopChan:     make(chan struct{}),
	}
}

// Start begins processing jobs
func (w *Worker) Start(ctx context.Context) error {
	w.logger.Info("Starting worker",
		slog.Int("concurrency", w.concurrency),
		slog.Duration("job_timeout", w.jobTimeout),
	)

	// TODO: Phase 2 - Implement job processing logic
	// 1. Subscribe to RabbitMQ queue
	// 2. Spawn worker goroutines
	// 3. Process jobs concurrently
	// 4. Handle job execution, retries, and status updates

	// Placeholder: Keep worker running until context is canceled
	<-ctx.Done()
	w.logger.Info("Worker context canceled, stopping...")

	return nil
}

// Stop gracefully stops the worker
func (w *Worker) Stop() {
	w.logger.Info("Stopping worker...")
	close(w.stopChan)
	w.wg.Wait()
	w.logger.Info("Worker stopped")
}

// processJob processes a single job (placeholder)
func (w *Worker) processJob(ctx context.Context, jobID string) error {
	w.logger.Info("Processing job",
		slog.String("job_id", jobID),
	)

	// TODO: Phase 2 - Implement job execution logic
	// 1. Fetch job from database
	// 2. Update job status to RUNNING
	// 3. Execute job logic based on job_type
	// 4. Update job status to COMPLETED/FAILED
	// 5. Send heartbeat during execution
	// 6. Handle retries on failure

	return nil
}
