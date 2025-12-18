package handler

import (
	"log/slog"

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
	dbClient     *postgresql.Client
	rabbitClient *rabbitmq.Client
}

// NewJobHandler creates a new JobHandler instance
func NewJobHandler(deps *Dependencies) *JobHandler {
	return &JobHandler{
		logger:       deps.Logger,
		dbClient:     deps.DBClient,
		rabbitClient: deps.RabbitClient,
	}
}
