package handler

import (
	"log/slog"

	"github.com/cuongbtq/practice-be/internal/api/storage"
	"github.com/cuongbtq/practice-be/shared/postgresql"
	"github.com/cuongbtq/practice-be/shared/rabbitmq"
)

// Dependencies holds all dependencies needed by handlers
type Dependencies struct {
	Logger       *slog.Logger
	DBClient     *postgresql.Client
	RabbitClient *rabbitmq.Client
}

// JobHandler handles job-related HTTP requests
type JobHandler struct {
	logger       *slog.Logger
	rabbitClient *rabbitmq.Client
	storage      *storage.Storage
}

// NewJobHandler creates a new JobHandler instance
func NewJobHandler(deps *Dependencies) *JobHandler {
	return &JobHandler{
		logger:       deps.Logger,
		rabbitClient: deps.RabbitClient,
		storage:      storage.NewStorage(deps.DBClient),
	}
}
