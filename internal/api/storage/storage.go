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

func (s *Storage) ListJobs(ctx context.Context, filter JobFilter) ([]model.Job, error) {
	query := `
        SELECT 
            job_id, idempotency_key, user_id, job_type,
            payload, status, created_at, updated_at
        FROM jobs
        WHERE 1=1
    `
	args := []interface{}{}
	argIdx := 1

	// Filters
	if filter.UserID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, filter.UserID)
		argIdx++
	}

	if filter.JobType != "" {
		query += fmt.Sprintf(" AND job_type = $%d", argIdx)
		args = append(args, filter.JobType)
		argIdx++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.Cursor != nil {
		query += fmt.Sprintf(" AND (created_at, job_id) < ($%d, $%d)", argIdx, argIdx+1)
		args = append(args, filter.Cursor.CreatedAt, filter.Cursor.JobID)
		argIdx += 2
	}

	// Order by created_at DESC, job_id DESC for consistent pagination
	query += " ORDER BY created_at DESC, job_id DESC"

	// Fetch one extra to determine if there are more results
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, filter.PageSize+1)

	var jobs []model.Job
	err := s.db.SelectContext(ctx, &jobs, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	return jobs, nil
}
