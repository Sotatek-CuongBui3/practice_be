-- Drop indexes
DROP INDEX IF EXISTS idx_job_history_created_at;
DROP INDEX IF EXISTS idx_job_history_job_id;

-- Drop job_history table
DROP TABLE IF EXISTS job_history;
