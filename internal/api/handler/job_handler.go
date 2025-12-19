package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/cuongbtq/practice-be/internal/api/dto"
	"github.com/cuongbtq/practice-be/internal/api/model"
	"github.com/cuongbtq/practice-be/internal/api/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateJob handles POST /api/v1/jobs
// Creates a new background job for processing
func (h *JobHandler) CreateJob(c *gin.Context) {
	h.logger.Info("CreateJob called",
		slog.String("method", c.Request.Method),
		slog.String("path", c.Request.URL.Path),
	)

	// TODO: Implement job creation logic
	// 1. Validate request body
	var req dto.CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	job := model.Job{
		JobID:          uuid.New().String(),
		IdempotencyKey: req.IdempotencyKey,
		UserID:         req.UserID,
		JobType:        req.JobType,
		Payload:        req.Payload,
		Status:         "PENDING",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// 2. Check idempotency key
	// 3. Create job record in database
	err := h.storage.CreateJob(c.Request.Context(), &job)
	if err != nil {
		h.logger.Error("Failed to create job", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create job",
		})
		return
	}

	// 4. Publish message to RabbitMQ
	// 5. Return job response
	c.JSON(http.StatusOK, gin.H{
		"job_id":          job.JobID,
		"idempotency_key": job.IdempotencyKey,
		"user_id":         job.UserID,
		"job_type":        job.JobType,
		"payload":         job.Payload,
		"status":          job.Status,
		"created_at":      job.CreatedAt,
		"updated_at":      job.UpdatedAt,
	})
}

// GetJob handles GET /api/v1/jobs/:job_id
// Retrieves detailed information about a specific job
func (h *JobHandler) GetJob(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "job_id is required",
		})
		return
	}

	h.logger.Info("GetJob called",
		slog.String("method", c.Request.Method),
		slog.String("path", c.Request.URL.Path),
		slog.String("job_id", jobID),
	)

	// 1. Validate job_id format (UUID)
	if _, err := uuid.Parse(jobID); err != nil {
		h.logger.Error("Invalid job_id format", slog.String("job_id", jobID), slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "job_id must be a valid UUID",
		})
		return
	}

	// 2. Query job from database
	job, err := h.storage.GetJobByID(c.Request.Context(), jobID)
	if err != nil {
		h.logger.Error("Failed to get job", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get job",
		})
		return
	}

	// 3. Return job details
	c.JSON(http.StatusOK, gin.H{
		"job_id":          job.JobID,
		"idempotency_key": job.IdempotencyKey,
		"user_id":         job.UserID,
		"job_type":        job.JobType,
		"payload":         job.Payload,
		"status":          job.Status,
		"created_at":      job.CreatedAt,
		"updated_at":      job.UpdatedAt,
	})
}

// ListJobs handles GET /api/v1/jobs
// Lists jobs with optional filtering and pagination
func (h *JobHandler) ListJobs(c *gin.Context) {
	h.logger.Info("ListJobs called",
		slog.String("method", c.Request.Method),
		slog.String("path", c.Request.URL.Path),
		slog.String("query", c.Request.URL.RawQuery),
	)

	// TODO: Implement list jobs logic
	// 1. Parse query parameters (status, job_type, user_id, limit, offset, sort)
	var req dto.ListJobsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("Invalid query parameters", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid query parameters",
		})
		return
	}

	// 2. Validate parameters
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	if req.PageSize > 100 {
		req.PageSize = 100
	}

	// 3. Decode cursor for pagination
	cursor, err := DecodeJobCursor(req.Cursor)
	if err != nil {
		h.logger.Error("Invalid cursor", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid cursor",
		})
		return
	}

	h.logger.Debug("Decoded cursor", slog.Any("cursor", cursor))

	// 4. Build filter and query jobs from database
	filter := storage.JobFilter{
		UserID:   req.UserID,
		JobType:  req.JobType,
		Status:   req.Status,
		PageSize: req.PageSize,
		Cursor:   cursor,
	}

	jobs, err := h.storage.ListJobs(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list jobs", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list jobs",
		})
		return
	}

	// 5. Prepare response with next cursor if more results exist
	hasMore := len(jobs) > req.PageSize
	if hasMore {
		jobs = jobs[:req.PageSize]
	}

	jobResponse := make([]dto.JobDTO, len(jobs))
	for i, job := range jobs {
		jobResponse[i] = dto.JobDTO{
			JobID:          job.JobID,
			IdempotencyKey: job.IdempotencyKey,
			UserID:         job.UserID,
			JobType:        job.JobType,
			Payload:        job.Payload,
			Status:         job.Status,
			CreatedAt:      job.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      job.UpdatedAt.Format(time.RFC3339),
		}
	}

	var nextCursor string
	if hasMore {
		lastJob := jobs[len(jobs)-1]
		cursorObj := storage.JobCursor{
			CreatedAt: lastJob.CreatedAt,
			JobID:     lastJob.JobID,
		}
		nextCursor, err = EncodeJobCursor(&cursorObj)
		if err != nil {
			h.logger.Error("Failed to encode next cursor", slog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to encode next cursor",
			})
			return
		}
	}

	c.JSON(http.StatusOK, dto.ListJobsResponse{
		Jobs:       jobResponse,
		NextCursor: nextCursor,
	})
}

// CancelJob handles POST /api/v1/jobs/:job_id/cancel
// Cancels a pending or running job
func (h *JobHandler) CancelJob(c *gin.Context) {
	jobID := c.Param("job_id")

	h.logger.Info("CancelJob called",
		slog.String("method", c.Request.Method),
		slog.String("path", c.Request.URL.Path),
		slog.String("job_id", jobID),
	)

	// 1. Validate job_id format (UUID)
	if _, err := uuid.Parse(jobID); err != nil {
		h.logger.Error("Invalid job_id format", slog.String("job_id", jobID), slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "job_id must be a valid UUID",
		})
		return
	}

	// TODO: Implement cancel job logic
	// 2. Check if job can be canceled (not in terminal state)
	// 3. Update job status to CANCELED
	// 4. Signal worker to stop execution (via RabbitMQ or database flag)
	// 5. Return cancellation response

	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "CancelJob endpoint - Not implemented yet",
		"job_id":  jobID,
		"status":  "todo",
	})
}

// DeleteJob handles DELETE /api/v1/jobs/:job_id
// Permanently deletes a job record from the database
func (h *JobHandler) DeleteJob(c *gin.Context) {
	jobID := c.Param("job_id")

	h.logger.Info("DeleteJob called",
		slog.String("method", c.Request.Method),
		slog.String("path", c.Request.URL.Path),
		slog.String("job_id", jobID),
	)

	// 1. Validate job_id format (UUID)
	if _, err := uuid.Parse(jobID); err != nil {
		h.logger.Error("Invalid job_id format", slog.String("job_id", jobID), slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "job_id must be a valid UUID",
		})
		return
	}

	// TODO: Implement delete job logic
	// 2. Check if job is in terminal state (COMPLETED, FAILED, CANCELED)
	// 3. Delete job record from database
	// 4. Return 204 No Content on success

	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "DeleteJob endpoint - Not implemented yet",
		"job_id":  jobID,
		"status":  "todo",
	})
}
