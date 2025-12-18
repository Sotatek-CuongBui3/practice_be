package router

import (
	"net/http"

	"github.com/cuongbtq/practice-be/internal/api/handler"
	"github.com/gin-gonic/gin"
)

// SetupRouter configures and returns the Gin router with all routes
func SetupRouter(deps *handler.Dependencies) *gin.Engine {
	r := gin.New()

	// Middleware
	r.Use(gin.Recovery())
	r.Use(LoggerMiddleware(deps.Logger))
	r.Use(CORSMiddleware())

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "job-api-service",
		})
	})

	// Initialize job handler
	jobHandler := handler.NewJobHandler(deps)

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		jobs := v1.Group("/jobs")
		{
			// POST /api/v1/jobs - Create a new job
			jobs.POST("", jobHandler.CreateJob)

			// GET /api/v1/jobs - List jobs with filtering and pagination
			jobs.GET("", jobHandler.ListJobs)

			// GET /api/v1/jobs/:job_id - Get job details
			jobs.GET("/:job_id", jobHandler.GetJob)

			// POST /api/v1/jobs/:job_id/cancel - Cancel a job
			jobs.POST("/:job_id/cancel", jobHandler.CancelJob)

			// DELETE /api/v1/jobs/:job_id - Delete a job
			jobs.DELETE("/:job_id", jobHandler.DeleteJob)
		}
	}

	return r
}
