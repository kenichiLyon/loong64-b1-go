CREATE TABLE IF NOT EXISTS teacher_reviews (
  id text PRIMARY KEY,
  submission_id text NOT NULL UNIQUE REFERENCES submissions(id) ON DELETE RESTRICT,
  evaluation_result_id text REFERENCES evaluation_results(id) ON DELETE SET NULL,
  experiment_id text NOT NULL REFERENCES experiments(id) ON DELETE RESTRICT,
  rubric_version_id text NOT NULL REFERENCES rubric_template_versions(id) ON DELETE RESTRICT,
  status text NOT NULL CHECK (status IN ('draft', 'published')),
  total_score_bps integer NOT NULL CHECK (total_score_bps BETWEEN 0 AND 10000),
  teacher_comment text NOT NULL DEFAULT '',
  created_by text NOT NULL,
  updated_by text NOT NULL,
  published_by text NOT NULL DEFAULT '',
  published_at timestamp,
  created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_teacher_reviews_submission_status ON teacher_reviews (submission_id, status);
CREATE INDEX IF NOT EXISTS idx_teacher_reviews_published_at ON teacher_reviews (published_at DESC) WHERE status = 'published';

CREATE TABLE IF NOT EXISTS teacher_metric_scores (
  id text PRIMARY KEY,
  teacher_review_id text NOT NULL REFERENCES teacher_reviews(id) ON DELETE CASCADE,
  metric_id text NOT NULL REFERENCES rubric_metrics(id) ON DELETE RESTRICT,
  metric_code text NOT NULL,
  final_score integer NOT NULL CHECK (final_score >= 0),
  max_score integer NOT NULL CHECK (max_score > 0),
  weight_bps integer NOT NULL CHECK (weight_bps >= 0),
  source text NOT NULL CHECK (source IN ('manual', 'rule', 'llm')),
  source_metric_score_id text REFERENCES metric_scores(id) ON DELETE SET NULL,
  comment text NOT NULL DEFAULT '',
  adjustment_reason text NOT NULL DEFAULT '',
  created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (teacher_review_id, metric_id),
  CHECK (final_score <= max_score)
);

CREATE INDEX IF NOT EXISTS idx_teacher_metric_scores_review ON teacher_metric_scores (teacher_review_id);
CREATE INDEX IF NOT EXISTS idx_teacher_metric_scores_metric_code ON teacher_metric_scores (metric_code);

DROP TRIGGER IF EXISTS trg_prevent_published_teacher_review_update;
CREATE TRIGGER trg_prevent_published_teacher_review_update
BEFORE UPDATE ON teacher_reviews
FOR EACH ROW
WHEN OLD.status = 'published'
BEGIN
  SELECT RAISE(ABORT, 'published teacher reviews are immutable');
END;

DROP TRIGGER IF EXISTS trg_prevent_published_teacher_review_delete;
CREATE TRIGGER trg_prevent_published_teacher_review_delete
BEFORE DELETE ON teacher_reviews
FOR EACH ROW
WHEN OLD.status = 'published'
BEGIN
  SELECT RAISE(ABORT, 'published teacher reviews are immutable');
END;

DROP TRIGGER IF EXISTS trg_prevent_published_teacher_metric_score_update;
CREATE TRIGGER trg_prevent_published_teacher_metric_score_update
BEFORE UPDATE ON teacher_metric_scores
FOR EACH ROW
WHEN EXISTS (SELECT 1 FROM teacher_reviews WHERE id = OLD.teacher_review_id AND status = 'published')
BEGIN
  SELECT RAISE(ABORT, 'published teacher metric scores are immutable');
END;

DROP TRIGGER IF EXISTS trg_prevent_published_teacher_metric_score_delete;
CREATE TRIGGER trg_prevent_published_teacher_metric_score_delete
BEFORE DELETE ON teacher_metric_scores
FOR EACH ROW
WHEN EXISTS (SELECT 1 FROM teacher_reviews WHERE id = OLD.teacher_review_id AND status = 'published')
BEGIN
  SELECT RAISE(ABORT, 'published teacher metric scores are immutable');
END;
