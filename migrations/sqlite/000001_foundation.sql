CREATE TABLE IF NOT EXISTS app_metadata (
  key text PRIMARY KEY,
  value text NOT NULL,
  updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS jobs (
  id text PRIMARY KEY,
  job_type text NOT NULL,
  status text NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled')),
  payload text NOT NULL DEFAULT '{}',
  error text,
  attempts integer NOT NULL DEFAULT 0,
  run_after timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  started_at timestamp,
  finished_at timestamp,
  created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_jobs_status_run_after ON jobs (status, run_after);
CREATE INDEX IF NOT EXISTS idx_jobs_job_type ON jobs (job_type);

CREATE TABLE IF NOT EXISTS audit_logs (
  id integer PRIMARY KEY AUTOINCREMENT,
  actor_id text,
  action text NOT NULL,
  target_type text NOT NULL,
  target_id text,
  detail text NOT NULL DEFAULT '{}',
  request_id text,
  created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_action_created_at ON audit_logs (action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_target ON audit_logs (target_type, target_id);
