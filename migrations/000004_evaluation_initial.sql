CREATE TABLE IF NOT EXISTS evaluation_results (
  id text PRIMARY KEY,
  submission_id text NOT NULL REFERENCES submissions(id) ON DELETE RESTRICT,
  experiment_id text NOT NULL REFERENCES experiments(id) ON DELETE RESTRICT,
  rubric_version_id text NOT NULL REFERENCES rubric_template_versions(id) ON DELETE RESTRICT,
  status text NOT NULL CHECK (status IN ('completed', 'needs_review', 'failed')),
  rule_status text NOT NULL CHECK (rule_status IN ('succeeded', 'failed', 'skipped')),
  llm_status text NOT NULL CHECK (llm_status IN ('succeeded', 'failed', 'skipped', 'not_configured')),
  prompt_version text NOT NULL,
  evidence_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  rule_summary jsonb NOT NULL DEFAULT '{}'::jsonb,
  llm_summary text NOT NULL DEFAULT '',
  needs_teacher_review boolean NOT NULL DEFAULT true,
  error text NOT NULL DEFAULT '',
  created_by text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_evaluation_results_submission_created_at ON evaluation_results (submission_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_evaluation_results_status_created_at ON evaluation_results (status, created_at DESC);

CREATE TABLE IF NOT EXISTS rule_check_findings (
  id text PRIMARY KEY,
  evaluation_result_id text NOT NULL REFERENCES evaluation_results(id) ON DELETE CASCADE,
  category text NOT NULL,
  severity text NOT NULL CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
  message text NOT NULL,
  evidence_ref text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_rule_check_findings_eval_severity ON rule_check_findings (evaluation_result_id, severity);
CREATE INDEX IF NOT EXISTS idx_rule_check_findings_category ON rule_check_findings (category);

CREATE TABLE IF NOT EXISTS metric_scores (
  id text PRIMARY KEY,
  evaluation_result_id text NOT NULL REFERENCES evaluation_results(id) ON DELETE CASCADE,
  metric_id text NOT NULL REFERENCES rubric_metrics(id) ON DELETE RESTRICT,
  metric_code text NOT NULL,
  source text NOT NULL CHECK (source IN ('rule', 'llm')),
  suggested_score integer NOT NULL CHECK (suggested_score >= 0),
  max_score integer NOT NULL CHECK (max_score > 0),
  confidence_bps integer NOT NULL CHECK (confidence_bps BETWEEN 0 AND 10000),
  rationale text NOT NULL DEFAULT '',
  evidence_refs jsonb NOT NULL DEFAULT '[]'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (evaluation_result_id, metric_id, source)
);

CREATE INDEX IF NOT EXISTS idx_metric_scores_eval_source ON metric_scores (evaluation_result_id, source);
CREATE INDEX IF NOT EXISTS idx_metric_scores_metric_code ON metric_scores (metric_code);

CREATE TABLE IF NOT EXISTS llm_call_logs (
  id text PRIMARY KEY,
  evaluation_result_id text NOT NULL REFERENCES evaluation_results(id) ON DELETE CASCADE,
  provider text NOT NULL DEFAULT 'openai-compatible',
  model text NOT NULL,
  prompt_version text NOT NULL,
  input_hash text NOT NULL,
  output jsonb NOT NULL DEFAULT '{}'::jsonb,
  status text NOT NULL CHECK (status IN ('succeeded', 'failed', 'skipped')),
  error text NOT NULL DEFAULT '',
  latency_ms integer NOT NULL DEFAULT 0 CHECK (latency_ms >= 0),
  prompt_tokens integer NOT NULL DEFAULT 0 CHECK (prompt_tokens >= 0),
  completion_tokens integer NOT NULL DEFAULT 0 CHECK (completion_tokens >= 0),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_llm_call_logs_eval_created_at ON llm_call_logs (evaluation_result_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_llm_call_logs_status_created_at ON llm_call_logs (status, created_at DESC);