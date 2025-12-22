package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/cuongbtq/practice-be/internal/api/model"
	"github.com/cuongbtq/practice-be/shared/postgresql"
	"github.com/jmoiron/sqlx"
)

type Storage struct {
	db *sqlx.DB
}

func NewStorage(pg *postgresql.Client) *Storage {
	return &Storage{
		db: pg.GetDB(),
	}
}

// CreateJob inserts a new job record into the database
func (s *Storage) CreateJob(ctx context.Context, job *model.Job) error {
	query := `
		INSERT INTO jobs (
			job_id, idempotency_key, user_id, job_type,
			payload, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8
		)
	`

	_, err := s.db.ExecContext(
		ctx,
		query,
		job.JobID,
		job.IdempotencyKey,
		job.UserID,
		job.JobType,
		job.Payload,
		job.Status,
		job.CreatedAt,
		job.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

// GetJobByID retrieves a job by its JobID
func (s *Storage) GetJobByID(ctx context.Context, jobID string) (*model.Job, error) {
	var job model.Job
	query := `
		SELECT 
			job_id, idempotency_key, user_id, job_type,
			payload, status, created_at, updated_at
		FROM jobs
		WHERE job_id = $1
	`

	err := s.db.GetContext(ctx, &job, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &job, nil
}

type JobFilter struct {
	UserID   string
	JobType  string
	Status   string
	PageSize int
	Cursor   *JobCursor
}

type JobCursor struct {
	CreatedAt time.Time
	JobID     string
}

// ListJobs retrieves jobs based on the provided filter and pagination cursor
func (s *Storage) ListJobs(ctx context.Context, filter JobFilter) ([]model.Job, error) {
	var conditions []string
	var args []interface{}

	// Build WHERE conditions
	if filter.UserID != "" {
		conditions = append(conditions, "user_id = ?")
		args = append(args, filter.UserID)
	}

	if filter.JobType != "" {
		conditions = append(conditions, "job_type = ?")
		args = append(args, filter.JobType)
	}

	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filter.Status)
	}

	if filter.Cursor != nil {
		conditions = append(conditions, "(created_at, job_id) < (?, ?)")
		args = append(args, filter.Cursor.CreatedAt, filter.Cursor.JobID)
	}

	// Build the query
	query := `
		SELECT 
			job_id, idempotency_key, user_id, job_type,
			payload, status, created_at, updated_at
		FROM jobs`

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	// Order by created_at DESC, job_id DESC for consistent pagination
	query += " ORDER BY created_at DESC, job_id DESC"

	// Fetch one extra to determine if there are more results
	query += " LIMIT ?"
	args = append(args, filter.PageSize+1)

	// Rebind query for PostgreSQL ($1, $2, etc.)
	query = s.db.Rebind(query)

	var jobs []model.Job
	err := s.db.SelectContext(ctx, &jobs, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	return jobs, nil
}
