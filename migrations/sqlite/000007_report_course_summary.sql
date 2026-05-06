DROP INDEX IF EXISTS idx_report_exports_report_type;

CREATE TABLE IF NOT EXISTS report_exports_new (
  id text PRIMARY KEY,
  report_type text NOT NULL CHECK (report_type IN ('submission_report', 'experiment_summary', 'course_summary')),
  scope_type text NOT NULL CHECK (scope_type IN ('submission', 'experiment', 'course')),
  scope_id text NOT NULL,
  format text NOT NULL CHECK (format IN ('html', 'csv', 'pdf')),
  status text NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
  storage_key text NOT NULL DEFAULT '',
  sha256_hex text NOT NULL DEFAULT '',
  byte_size integer NOT NULL DEFAULT 0 CHECK (byte_size >= 0),
  filter_json text NOT NULL DEFAULT '{}',
  error text NOT NULL DEFAULT '',
  requested_by text NOT NULL,
  created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  completed_at timestamp
);

INSERT INTO report_exports_new (
  id, report_type, scope_type, scope_id, format, status, storage_key, sha256_hex,
  byte_size, filter_json, error, requested_by, created_at, updated_at, completed_at
)
SELECT
  id, report_type, scope_type, scope_id, format, status, storage_key, sha256_hex,
  byte_size, filter_json, error, requested_by, created_at, updated_at, completed_at
FROM report_exports;

DROP TABLE report_exports;
ALTER TABLE report_exports_new RENAME TO report_exports;

CREATE INDEX IF NOT EXISTS idx_report_exports_requested_created ON report_exports (requested_by, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_report_exports_status_created ON report_exports (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_report_exports_scope ON report_exports (scope_type, scope_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_report_exports_report_type ON report_exports (report_type, created_at DESC);
