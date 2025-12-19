-- Create jobs table for storing background job information
CREATE TABLE IF NOT EXISTS jobs (
    id                BIGSERIAL PRIMARY KEY,
    job_id            VARCHAR(36) NOT NULL UNIQUE,
    idempotency_key   VARCHAR(255) UNIQUE,
    user_id           VARCHAR(100),
    job_type          VARCHAR(50) NOT NULL,
    status            VARCHAR(20) NOT NULL,
    priority          INTEGER DEFAULT 5,
    payload           JSONB NOT NULL,
    result            JSONB,
    error_message     TEXT,
    worker_id         VARCHAR(100),
    retry_count       INTEGER DEFAULT 0,
    max_retries       INTEGER DEFAULT 3,
    timeout_seconds   INTEGER DEFAULT 300,
    progress          INTEGER DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at        TIMESTAMPTZ,
    completed_at      TIMESTAMPTZ,
    last_heartbeat_at TIMESTAMPTZ,
    callback_url      VARCHAR(500)
);

-- Create indexes for query performance
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_idempotency_key ON jobs(idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_user_id ON jobs(user_id);
