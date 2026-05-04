CREATE TABLE IF NOT EXISTS report_exports (
  id text PRIMARY KEY,
  report_type text NOT NULL CHECK (report_type IN ('submission_report', 'experiment_summary')),
  scope_type text NOT NULL CHECK (scope_type IN ('submission', 'experiment')),
  scope_id text NOT NULL,
  format text NOT NULL CHECK (format IN ('html', 'csv', 'pdf')),
  status text NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
  storage_key text NOT NULL DEFAULT '',
  sha256_hex text NOT NULL DEFAULT '',
  byte_size bigint NOT NULL DEFAULT 0 CHECK (byte_size >= 0),
  filter_json jsonb NOT NULL DEFAULT '{}'::jsonb,
  error text NOT NULL DEFAULT '',
  requested_by text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_report_exports_requested_created ON report_exports (requested_by, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_report_exports_status_created ON report_exports (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_report_exports_scope ON report_exports (scope_type, scope_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_report_exports_report_type ON report_exports (report_type, created_at DESC);