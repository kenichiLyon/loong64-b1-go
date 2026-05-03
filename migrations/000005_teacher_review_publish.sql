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
  published_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
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
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (teacher_review_id, metric_id),
  CHECK (final_score <= max_score)
);

CREATE INDEX IF NOT EXISTS idx_teacher_metric_scores_review ON teacher_metric_scores (teacher_review_id);
CREATE INDEX IF NOT EXISTS idx_teacher_metric_scores_metric_code ON teacher_metric_scores (metric_code);

CREATE OR REPLACE FUNCTION prevent_published_teacher_review_update()
RETURNS trigger AS $$
BEGIN
  IF OLD.status = 'published' THEN
    RAISE EXCEPTION 'published teacher reviews are immutable';
  END IF;
  IF TG_OP = 'DELETE' THEN
    RETURN OLD;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_published_teacher_review_update ON teacher_reviews;
CREATE TRIGGER trg_prevent_published_teacher_review_update
BEFORE UPDATE OR DELETE ON teacher_reviews
FOR EACH ROW EXECUTE FUNCTION prevent_published_teacher_review_update();

CREATE OR REPLACE FUNCTION prevent_published_teacher_metric_score_update()
RETURNS trigger AS $$
BEGIN
  IF EXISTS (SELECT 1 FROM teacher_reviews WHERE id = OLD.teacher_review_id AND status = 'published') THEN
    RAISE EXCEPTION 'published teacher metric scores are immutable';
  END IF;
  IF TG_OP = 'DELETE' THEN
    RETURN OLD;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_published_teacher_metric_score_update ON teacher_metric_scores;
CREATE TRIGGER trg_prevent_published_teacher_metric_score_update
BEFORE UPDATE OR DELETE ON teacher_metric_scores
FOR EACH ROW EXECUTE FUNCTION prevent_published_teacher_metric_score_update();