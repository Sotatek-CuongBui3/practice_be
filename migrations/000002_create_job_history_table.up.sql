-- Create job_history table for audit trail
CREATE TABLE IF NOT EXISTS job_history (
    id            BIGSERIAL PRIMARY KEY,
    job_id        VARCHAR(36) NOT NULL,
    status_from   VARCHAR(20),
    status_to     VARCHAR(20) NOT NULL,
    worker_id     VARCHAR(100),
    error_message TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index for efficient job history queries
CREATE INDEX IF NOT EXISTS idx_job_history_job_id ON job_history(job_id);
CREATE INDEX IF NOT EXISTS idx_job_history_created_at ON job_history(created_at DESC);
