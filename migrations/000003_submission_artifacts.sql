CREATE TABLE IF NOT EXISTS submissions (
  id text PRIMARY KEY,
  experiment_id text NOT NULL REFERENCES experiments(id) ON DELETE RESTRICT,
  student_id text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  status text NOT NULL CHECK (status IN ('draft', 'submitted', 'parsing', 'parsed', 'failed', 'archived')),
  attempt_no integer NOT NULL DEFAULT 1 CHECK (attempt_no > 0),
  submitted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (experiment_id, student_id, attempt_no)
);

CREATE INDEX IF NOT EXISTS idx_submissions_experiment_status ON submissions (experiment_id, status);
CREATE INDEX IF NOT EXISTS idx_submissions_student_created_at ON submissions (student_id, created_at DESC);

CREATE TABLE IF NOT EXISTS artifacts (
  id text PRIMARY KEY,
  submission_id text NOT NULL REFERENCES submissions(id) ON DELETE RESTRICT,
  artifact_kind text NOT NULL CHECK (artifact_kind IN ('document', 'report', 'screenshot', 'code_archive', 'git_link', 'other')),
  original_name text NOT NULL,
  content_type text,
  byte_size bigint NOT NULL DEFAULT 0 CHECK (byte_size >= 0),
  sha256_hex text,
  storage_key text,
  source_url text,
  status text NOT NULL CHECK (status IN ('stored', 'queued', 'parsing', 'parsed', 'failed', 'rejected')),
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_by text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (storage_key IS NOT NULL OR source_url IS NOT NULL),
  CHECK (sha256_hex IS NULL OR sha256_hex ~ '^[0-9a-f]{64}$')
);

CREATE INDEX IF NOT EXISTS idx_artifacts_submission_kind ON artifacts (submission_id, artifact_kind);
CREATE INDEX IF NOT EXISTS idx_artifacts_sha256 ON artifacts (sha256_hex) WHERE sha256_hex IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_artifacts_storage_key_unique ON artifacts (storage_key) WHERE storage_key IS NOT NULL;

CREATE TABLE IF NOT EXISTS extracted_contents (
  id text PRIMARY KEY,
  artifact_id text NOT NULL UNIQUE REFERENCES artifacts(id) ON DELETE RESTRICT,
  status text NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
  text_excerpt text NOT NULL DEFAULT '',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  error text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_extracted_contents_status ON extracted_contents (status, updated_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_submission_targets ON audit_logs (target_type, target_id, created_at DESC);
