package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/cuongbtq/practice-be/internal/worker/domain"
	"github.com/cuongbtq/practice-be/internal/worker/storage"
	"github.com/cuongbtq/practice-be/shared/postgresql"
	"github.com/cuongbtq/practice-be/shared/rabbitmq"
)

// Config holds worker configuration
type Config struct {
	Logger            *slog.Logger
	DBClient          *postgresql.Client
	RabbitClient      *rabbitmq.Client
	Concurrency       int
	JobTimeout        time.Duration
	ShutdownTimeout   time.Duration
	PrefetchCount     int
	RabbitMQQueueName string
}

// Worker represents the background job worker
type Worker struct {
	workerID          string
	logger            *slog.Logger
	storage           *storage.Storage
	rabbitClient      *rabbitmq.Client
	concurrency       int
	jobTimeout        time.Duration
	shutdownTimeout   time.Duration
	prefetchCount     int
	rabbitMQQueueName string
	wg                sync.WaitGroup
	stopChan          chan struct{}
	jobsChan          chan *domain.JobMessage
}

// NewWorker creates a new worker instance
func NewWorker(cfg *Config) *Worker {
	// Generate worker ID using PID
	workerID := fmt.Sprintf("worker-%d", os.Getpid())

	// Create storage layer
	storage := storage.NewStorage(cfg.DBClient.GetDB(), cfg.Logger)

	return &Worker{
		workerID:          workerID,
		logger:            cfg.Logger,
		storage:           storage,
		rabbitClient:      cfg.RabbitClient,
		concurrency:       cfg.Concurrency,
		jobTimeout:        cfg.JobTimeout,
		shutdownTimeout:   cfg.ShutdownTimeout,
		prefetchCount:     cfg.PrefetchCount,
		rabbitMQQueueName: cfg.RabbitMQQueueName,
		stopChan:          make(chan struct{}),
		jobsChan:          make(chan *domain.JobMessage, cfg.Concurrency*2), // Buffered channel
	}
}

// Start begins processing jobs
func (w *Worker) Start(ctx context.Context) error {
	w.logger.Info("Starting worker",
		slog.String("worker_id", w.workerID),
		slog.Int("concurrency", w.concurrency),
		slog.Duration("job_timeout", w.jobTimeout),
	)

	// Step 1: Setup RabbitMQ consumer
	deliveries, err := w.setupConsumer(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup consumer: %w", err)
	}

	// Step 2: Start message dispatcher goroutine
	go w.startMessageDispatcher(ctx, deliveries)

	// Step 3: Spawn worker pool goroutines
	w.spawnWorkerPool(ctx)

	w.logger.Info("Worker started successfully",
		slog.String("worker_id", w.workerID),
		slog.Int("worker_count", w.concurrency),
	)

	// Step 4: Wait for context cancellation
	<-ctx.Done()
	w.logger.Info("Worker context canceled, stopping...")

	return nil
}

// Stop gracefully stops the worker with timeout
func (w *Worker) Stop() {
	w.logger.Info("Initiating graceful shutdown...",
		slog.Duration("timeout", w.shutdownTimeout),
	)

	// Signal all workers to stop
	close(w.stopChan)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		w.logger.Info("All workers stopped gracefully")
	case <-time.After(w.shutdownTimeout):
		w.logger.Warn("Shutdown timeout exceeded, forcing stop",
			slog.Duration("timeout", w.shutdownTimeout),
		)
	}

	// Close jobs channel to signal message dispatcher
	close(w.jobsChan)

	w.logger.Info("Worker shutdown complete")
}
