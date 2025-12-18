# Background Job Processing System

[![CI](https://github.com/cuongbtq/practice-be/workflows/CI/badge.svg)](https://github.com/cuongbtq/practice-be/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/cuongbtq/practice-be)](https://goreportcard.com/report/github.com/cuongbtq/practice-be)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A distributed, reliable background job processing system built with Go and PostgreSQL. This system enables clients to submit long-running tasks via HTTP API, with guaranteed at-least-once execution, idempotent processing, and automatic crash recovery.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Technology Stack](#technology-stack)
- [Database Schema](#database-schema)
- [API Specifications](#api-specifications)
- [Job Lifecycle](#job-lifecycle)
- [System Guarantees](#system-guarantees)
- [Local Development Setup](#local-development-setup)
- [Development Phases](#development-phases)

## Overview

This system provides a robust infrastructure for handling background jobs with the following capabilities:

- **HTTP-based job submission** - RESTful API for creating and managing jobs
- **Asynchronous processing** - Jobs are processed by worker services independent of client requests
- **Status tracking** - Real-time job status and result queries
- **Idempotency** - Duplicate submissions are detected and handled via idempotency keys
- **Reliability** - At-least-once execution guarantee with automatic retries
- **Fault tolerance** - System recovers gracefully from worker crashes

### Use Cases

- Long-running data processing tasks
- Batch operations (email sending, report generation)
- External API integrations with retries
- Scheduled/delayed task execution
- Webhook delivery with guaranteed delivery

## Architecture

```
┌──────────┐      HTTP      ┌─────────────┐
│  Client  │───────────────>│ API Service │
└──────────┘                └──────┬──────┘
                                   │
                          ┌────────┼────────┐
                          │        │        │
                          ↓        ↓        ↓
                   ┌──────────┐ ┌──────────┐
                   │PostgreSQL│ │ RabbitMQ │
                   │  (Jobs)  │ │ (Queue)  │
                   └──────────┘ └────┬─────┘
                          ↑          │
                          │          ↓
                          │    ┌─────────────┐
                          └────│   Worker    │
                               │   Service   │
                               └──────┬──────┘
                                      │
                                      ↓
                               ┌─────────────┐
                               │  Callback   │
                               │   Service   │
                               └─────────────┘
```

### Components

1. **API Service** - HTTP REST API for job management (CRUD operations), publishes job requests to RabbitMQ
2. **Worker Service** - Consumes job messages from RabbitMQ and executes jobs (Phase 2)
3. **Callback Service** - Delivers webhook notifications on job completion (Phase 2)
4. **PostgreSQL** - Persistent storage for job metadata, status, and results
5. **RabbitMQ** - Message queue for distributing job requests to workers (Phase 2)

## Technology Stack

### Backend
- **Language:** Go 1.21+
- **HTTP Framework:** Gin Gonic (high-performance HTTP framework with built-in validation)
- **Database:** PostgreSQL 15+
- **ORM/Query Builder:** sqlx (SQL-first approach)
- **Migrations:** golang-migrate
- **Logging:** zerolog (structured JSON logging)

### Infrastructure
- **Containerization:** Docker & Docker Compose
- **Message Queue:** RabbitMQ 3.12+ (Phase 2)
- **Cache:** Redis 7+ (Phase 2 - for rate limiting & idempotency)

### Observability (Future)
- **Metrics:** Prometheus
- **Tracing:** OpenTelemetry
- **Dashboards:** Grafana

## Database Schema

### Jobs Table

```sql
CREATE TABLE jobs (
    id                BIGSERIAL PRIMARY KEY,
    job_id            VARCHAR(36) NOT NULL UNIQUE,      -- UUID for external reference
    idempotency_key   VARCHAR(255) UNIQUE,              -- Client deduplication key
    user_id           VARCHAR(100),                     -- Job owner
    job_type          VARCHAR(50) NOT NULL,             -- Type of job (e.g., 'email', 'report')
    status            VARCHAR(20) NOT NULL,             -- PENDING, RUNNING, COMPLETED, FAILED, CANCELED, RETRYING
    priority          INTEGER DEFAULT 5,                -- 1 (highest) to 10 (lowest)
    payload           JSONB NOT NULL,                   -- Job input data
    result            JSONB,                            -- Job output data
    error_message     TEXT,                             -- Failure reason
    worker_id         VARCHAR(100),                     -- Which worker is processing
    retry_count       INTEGER DEFAULT 0,                -- Current retry attempt
    max_retries       INTEGER DEFAULT 3,                -- Maximum retry attempts
    timeout_seconds   INTEGER DEFAULT 300,              -- Execution timeout
    progress          INTEGER DEFAULT 0,                -- 0-100 completion percentage
    created_at        TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at        TIMESTAMP,                        -- When job execution began
    completed_at      TIMESTAMP,                        -- When job finished
    last_heartbeat_at TIMESTAMP,                        -- For crash detection
    callback_url      VARCHAR(500)                      -- Webhook notification URL
);

-- Indexes for query performance
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_idempotency_key ON jobs(idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);
CREATE INDEX idx_jobs_user_id ON jobs(user_id);
```

### Job History Table (Future - Audit Trail)

```sql
CREATE TABLE job_history (
    id            BIGSERIAL PRIMARY KEY,
    job_id        VARCHAR(36) NOT NULL,
    status_from   VARCHAR(20),
    status_to     VARCHAR(20) NOT NULL,
    worker_id     VARCHAR(100),
    error_message TEXT,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## API Specifications

### Base URL
```
http://localhost:8080/api/v1
```

### 1. Create Job

**Endpoint:** `POST /api/v1/jobs`

**Description:** Submit a new background job for processing. Supports idempotency via optional idempotency key.

**Request Headers:**
```
Content-Type: application/json
X-Idempotency-Key: <optional-unique-key>
```

**Request Body:**
```json
{
  "job_type": "send_email",
  "payload": {
    "to": "user@example.com",
    "subject": "Welcome!",
    "body": "Thanks for signing up."
  },
  "priority": 5,
  "max_retries": 3,
  "timeout_seconds": 300,
  "callback_url": "https://example.com/webhooks/job-completed"
}
```

**Response (201 Created):**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "PENDING",
  "created_at": "2025-12-17T10:30:00Z",
  "message": "Job created successfully"
}
```

**Response (200 OK - Idempotent duplicate):**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "COMPLETED",
  "created_at": "2025-12-17T10:25:00Z",
  "message": "Job already exists (idempotency key matched)"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid request body or parameters
- `422 Unprocessable Entity` - Validation failed
- `500 Internal Server Error` - Server error

---

### 2. Get Job Status

**Endpoint:** `GET /api/v1/jobs/{job_id}`

**Description:** Retrieve detailed information about a specific job.

**Response (200 OK):**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "job_type": "send_email",
  "status": "COMPLETED",
  "priority": 5,
  "payload": {
    "to": "user@example.com",
    "subject": "Welcome!",
    "body": "Thanks for signing up."
  },
  "result": {
    "message_id": "msg_12345",
    "sent_at": "2025-12-17T10:31:45Z"
  },
  "error_message": null,
  "retry_count": 0,
  "max_retries": 3,
  "progress": 100,
  "created_at": "2025-12-17T10:30:00Z",
  "updated_at": "2025-12-17T10:31:45Z",
  "started_at": "2025-12-17T10:30:05Z",
  "completed_at": "2025-12-17T10:31:45Z"
}
```

**Error Responses:**
- `404 Not Found` - Job does not exist
- `500 Internal Server Error` - Server error

---

### 3. List Jobs

**Endpoint:** `GET /api/v1/jobs`

**Description:** List jobs with optional filtering and pagination.

**Query Parameters:**
- `status` - Filter by status (PENDING, RUNNING, COMPLETED, FAILED, CANCELED, RETRYING)
- `job_type` - Filter by job type
- `user_id` - Filter by user ID
- `limit` - Number of results per page (default: 50, max: 100)
- `offset` - Pagination offset (default: 0)
- `sort` - Sort order: `created_at_asc`, `created_at_desc` (default)

**Example Request:**
```
GET /api/v1/jobs?status=COMPLETED&limit=20&offset=0
```

**Response (200 OK):**
```json
{
  "jobs": [
    {
      "job_id": "550e8400-e29b-41d4-a716-446655440000",
      "job_type": "send_email",
      "status": "COMPLETED",
      "created_at": "2025-12-17T10:30:00Z",
      "completed_at": "2025-12-17T10:31:45Z"
    },
    {
      "job_id": "660e8400-e29b-41d4-a716-446655440001",
      "job_type": "generate_report",
      "status": "COMPLETED",
      "created_at": "2025-12-17T09:15:00Z",
      "completed_at": "2025-12-17T09:25:30Z"
    }
  ],
  "total": 150,
  "limit": 20,
  "offset": 0
}
```

**Error Responses:**
- `400 Bad Request` - Invalid query parameters
- `500 Internal Server Error` - Server error

---

### 4. Cancel Job

**Endpoint:** `POST /api/v1/jobs/{job_id}/cancel`

**Description:** Cancel a pending or running job. Signals the worker to stop execution and updates the job status to CANCELED. Completed jobs cannot be canceled.

**Response (200 OK):**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "CANCELED",
  "message": "Job canceled successfully"
}
```

**Response (409 Conflict - Already completed):**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "COMPLETED",
  "message": "Job already completed, cannot cancel"
}
```

**Error Responses:**
- `404 Not Found` - Job does not exist
- `409 Conflict` - Job already in terminal state (COMPLETED/FAILED)
- `500 Internal Server Error` - Server error

---

### 5. Delete Job

**Endpoint:** `DELETE /api/v1/jobs/{job_id}`

**Description:** Permanently delete a job record from the database. Only jobs in terminal states (COMPLETED, FAILED, CANCELED) can be deleted.

**Response (204 No Content):**
```
(Empty response body)
```

**Error Responses:**
- `404 Not Found` - Job does not exist
- `409 Conflict` - Job is still active (PENDING, RUNNING, RETRYING)
- `500 Internal Server Error` - Server error

**Example Error Response (409):**
```json
{
  "error": "Cannot delete active job",
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "RUNNING",
  "message": "Job must be in terminal state (COMPLETED, FAILED, or CANCELED) before deletion"
}
```

---

## Job Lifecycle

```
                    ┌──────────┐
                    │ PENDING  │ ◄── Initial state after creation
                    └────┬─────┘
                         │
                         ↓
                    ┌──────────┐
              ┌────►│ RUNNING  │ ◄── Worker picks up job
              │     └────┬─────┘
              │          │
              │          ├─────────► Success
              │          │              ↓
              │          │         ┌──────────┐
              │          │         │COMPLETED │
              │          │         └──────────┘
              │          │
              │          ├─────────► Failure
              │          │              ↓
              │          │         ┌──────────┐
              │          │         │ FAILED   │
              │          │         └────┬─────┘
              │          │              │
              │          │              ├─► Retry limit not reached
              │          │              │      ↓
              │          │              │  ┌──────────┐
              │          │              │  │RETRYING  │
              │          │              │  └────┬─────┘
              │          │              │       │
              └──────────┴──────────────┴───────┘
                         │
                         └─► Retry limit reached
                                   ↓
                              ┌──────────┐
                              │ FAILED   │ (final)
                              └──────────┘

Note: CANCELED can be triggered from any state via POST /api/v1/jobs/{id}/cancel API
```

### State Transitions

| From State | To State  | Trigger                                   |
|-----------|-----------|-------------------------------------------|
| PENDING   | RUNNING   | Worker picks up job                       |
| PENDING   | CANCELED  | Client POST /cancel request               |
| RUNNING   | COMPLETED | Job execution successful                  |
| RUNNING   | FAILED    | Job execution failed                      |
| RUNNING   | CANCELED  | Client POST /cancel request (graceful stop)|
| FAILED    | RETRYING  | Retry count < max_retries                 |
| RETRYING  | RUNNING   | Automatic retry after backoff             |
| FAILED    | FAILED    | Retry count >= max_retries (final state)  |

## System Guarantees

### 1. Reliability
- **At-least-once execution** - Jobs are guaranteed to execute at least once, even after system failures
- **No job loss** - All jobs are persisted to PostgreSQL before acknowledgment
- **Crash recovery** - Workers can detect and resume abandoned jobs via heartbeat mechanism

### 2. Idempotency
- **Duplicate detection** - Idempotency keys prevent duplicate job creation
- **Safe retries** - Jobs can be retried multiple times without side effects
- **Client control** - Clients provide idempotency keys to ensure exactly-once semantics

### 3. Fault Tolerance
- **Worker failure detection** - Heartbeat monitoring detects crashed workers
- **Automatic retry** - Failed jobs are automatically retried with exponential backoff
- **Graceful degradation** - System continues operating with reduced capacity during partial failures

### 4. Consistency
- **Transactional updates** - Job state changes are atomic and transactional
- **Optimistic locking** - Prevents concurrent worker race conditions (Phase 2)

## Local Development Setup

### Prerequisites

- **Go 1.21+** - [Install Go](https://go.dev/doc/install)
- **Docker & Docker Compose** - [Install Docker](https://docs.docker.com/get-docker/)
- **Make** (optional) - Build automation
- **PostgreSQL Client** (optional) - For database inspection

### Quick Start

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd practice-be
   ```

2. **Start PostgreSQL and RabbitMQ with Docker Compose**
   ```bash
   docker-compose up -d postgres rabbitmq
   ```

3. **Install Go dependencies**
   ```bash
   go mod download
   ```

4. **Run database migrations**
   ```bash
   make migrate-up
   # OR manually:
   migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/jobs_db?sslmode=disable" up
   ```

5. **Run the API service**
   ```bash
   make run-api
   # OR manually:
   go run cmd/api-service/main.go
   ```

6. **Test the API**
   ```bash
   curl -X POST http://localhost:8080/api/v1/jobs \
     -H "Content-Type: application/json" \
     -H "X-Idempotency-Key: test-key-001" \
     -d '{
       "job_type": "test_job",
       "payload": {"message": "Hello, World!"},
       "priority": 5
     }'
   ```

### Environment Variables

Create a `.env` file in the project root:

```bash
# Database
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_NAME=jobs_db
DATABASE_SSLMODE=disable

# RabbitMQ
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USER=guest
RABBITMQ_PASSWORD=guest
RABBITMQ_VHOST=/
RABBITMQ_QUEUE_NAME=jobs_queue
RABBITMQ_EXCHANGE_NAME=jobs_exchange
RABBITMQ_EXCHANGE_TYPE=direct
RABBITMQ_ROUTING_KEY=job.created

# API Service
API_PORT=8080
API_READ_TIMEOUT=10s
API_WRITE_TIMEOUT=10s
API_IDLE_TIMEOUT=120s

# Logging
LOG_LEVEL=debug
LOG_FORMAT=json
```

### Development Commands

```bash
# Build all services
make build

# Run tests
make test

# Run with hot reload (using air)
make dev

# View logs
docker-compose logs -f api-service

# Access PostgreSQL
docker-compose exec postgres psql -U postgres -d jobs_db

# Stop all services
docker-compose down
```

## Development Phases

### Phase 1: HTTP API Foundation ✅ (Current)

**Goal:** Build core HTTP REST API for job management

**Deliverables:**
- ✅ HTTP API endpoints (POST, GET, GET list, POST cancel, DELETE)
- ✅ PostgreSQL database with migrations
- ✅ RabbitMQ integration for job queue
- ✅ Job CRUD operations with idempotency
- ✅ Request validation and error handling
- ✅ Structured logging
- ✅ Docker Compose setup (PostgreSQL + RabbitMQ)
- ✅ API documentation

**Status:** In Development

---

### Phase 2: Background Job Processing

**Goal:** Implement asynchronous job execution

**Deliverables:**
- Background worker service
- Job queue mechanism (RabbitMQ/Redis)
- Job execution engine with timeout handling
- Automatic retry logic with exponential backoff
- Worker heartbeat and crash recovery
- Concurrent job processing (configurable workers)

**Timeline:** After Phase 1 completion

---

### Phase 3: Callback & Monitoring

**Goal:** Add webhook notifications and observability

**Deliverables:**
- Callback service for webhook delivery
- Webhook retry mechanism with dead letter queue
- Prometheus metrics (job throughput, latency, error rates)
- Health check endpoints
- Crash detection and alerting
- Grafana dashboards

**Timeline:** After Phase 2 completion

---

### Phase 4: Advanced Features

**Goal:** Production-ready enhancements

**Features:**
- Job scheduling (run at specific time)
- Job dependencies (DAG workflows)
- Job prioritization and SLA guarantees
- Rate limiting per job type
- Multi-tenancy support
- Job result caching
- OpenTelemetry distributed tracing
- Horizontal scaling with leader election

**Timeline:** After Phase 3 completion

---

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

[Specify license here]

---

**Project Status:** Phase 1 - Initial Development  
**Last Updated:** December 17, 2025
