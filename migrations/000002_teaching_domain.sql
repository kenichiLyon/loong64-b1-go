CREATE TABLE IF NOT EXISTS users (
  id text PRIMARY KEY,
  username text NOT NULL,
  display_name text NOT NULL,
  email text,
  student_no text,
  employee_no text,
  status text NOT NULL CHECK (status IN ('active', 'disabled')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_unique ON users (lower(username));
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users (lower(email)) WHERE email IS NOT NULL AND email <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_student_no_unique ON users (student_no) WHERE student_no IS NOT NULL AND student_no <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_employee_no_unique ON users (employee_no) WHERE employee_no IS NOT NULL AND employee_no <> '';
CREATE INDEX IF NOT EXISTS idx_users_status ON users (status);
CREATE INDEX IF NOT EXISTS idx_users_display_name ON users (display_name);

CREATE TABLE IF NOT EXISTS user_roles (
  user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role text NOT NULL CHECK (role IN ('admin', 'teacher', 'student')),
  PRIMARY KEY (user_id, role)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles (role);

CREATE TABLE IF NOT EXISTS classes (
  id text PRIMARY KEY,
  code text NOT NULL,
  name text NOT NULL,
  grade_year integer,
  major text,
  status text NOT NULL CHECK (status IN ('active', 'archived')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_classes_code_unique ON classes (lower(code));
CREATE INDEX IF NOT EXISTS idx_classes_grade_year ON classes (grade_year);
CREATE INDEX IF NOT EXISTS idx_classes_status ON classes (status);

CREATE TABLE IF NOT EXISTS courses (
  id text PRIMARY KEY,
  code text NOT NULL,
  name text NOT NULL,
  term text NOT NULL,
  status text NOT NULL CHECK (status IN ('draft', 'active', 'archived')),
  created_by text REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_courses_code_term_unique ON courses (lower(code), lower(term));
CREATE INDEX IF NOT EXISTS idx_courses_status_term ON courses (status, term);
CREATE INDEX IF NOT EXISTS idx_courses_created_by ON courses (created_by);

CREATE TABLE IF NOT EXISTS course_classes (
  course_id text NOT NULL REFERENCES courses(id) ON DELETE RESTRICT,
  class_id text NOT NULL REFERENCES classes(id) ON DELETE RESTRICT,
  PRIMARY KEY (course_id, class_id)
);

CREATE INDEX IF NOT EXISTS idx_course_classes_class_id ON course_classes (class_id);

CREATE TABLE IF NOT EXISTS course_teachers (
  course_id text NOT NULL REFERENCES courses(id) ON DELETE RESTRICT,
  teacher_id text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  permission text NOT NULL CHECK (permission IN ('owner', 'editor', 'viewer')),
  assigned_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (course_id, teacher_id)
);

CREATE INDEX IF NOT EXISTS idx_course_teachers_teacher_id ON course_teachers (teacher_id);

CREATE TABLE IF NOT EXISTS enrollments (
  course_id text NOT NULL REFERENCES courses(id) ON DELETE RESTRICT,
  class_id text,
  student_id text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  status text NOT NULL CHECK (status IN ('active', 'dropped')),
  enrolled_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (course_id, student_id),
  FOREIGN KEY (course_id, class_id) REFERENCES course_classes(course_id, class_id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_enrollments_student_id ON enrollments (student_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_course_status ON enrollments (course_id, status);

CREATE TABLE IF NOT EXISTS rubric_templates (
  id text PRIMARY KEY,
  name text NOT NULL,
  description text,
  owner_id text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  scope text NOT NULL CHECK (scope IN ('private', 'course', 'global')),
  status text NOT NULL CHECK (status IN ('draft', 'active', 'archived')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_rubric_templates_owner_status ON rubric_templates (owner_id, status);
CREATE INDEX IF NOT EXISTS idx_rubric_templates_scope_status ON rubric_templates (scope, status);

CREATE TABLE IF NOT EXISTS rubric_template_versions (
  id text PRIMARY KEY,
  template_id text NOT NULL REFERENCES rubric_templates(id) ON DELETE RESTRICT,
  version_no integer NOT NULL CHECK (version_no > 0),
  status text NOT NULL CHECK (status IN ('draft', 'published', 'archived')),
  weight_mode text NOT NULL CHECK (weight_mode IN ('strict_100', 'normalized')),
  total_weight_bps integer NOT NULL CHECK (total_weight_bps >= 0),
  published_at timestamptz,
  created_by text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (template_id, version_no)
);

CREATE INDEX IF NOT EXISTS idx_rubric_template_versions_template_status ON rubric_template_versions (template_id, status);

CREATE TABLE IF NOT EXISTS rubric_metrics (
  id text PRIMARY KEY,
  version_id text NOT NULL REFERENCES rubric_template_versions(id) ON DELETE RESTRICT,
  code text NOT NULL,
  name text NOT NULL,
  description text,
  weight_bps integer NOT NULL CHECK (weight_bps >= 0),
  max_score integer NOT NULL CHECK (max_score > 0),
  sort_order integer NOT NULL CHECK (sort_order > 0),
  required_evidence jsonb NOT NULL DEFAULT '{}'::jsonb,
  UNIQUE (version_id, code),
  UNIQUE (version_id, sort_order)
);

CREATE INDEX IF NOT EXISTS idx_rubric_metrics_version_order ON rubric_metrics (version_id, sort_order);

CREATE TABLE IF NOT EXISTS experiments (
  id text PRIMARY KEY,
  course_id text NOT NULL REFERENCES courses(id) ON DELETE RESTRICT,
  title text NOT NULL,
  description text,
  submission_spec jsonb NOT NULL DEFAULT '{}'::jsonb,
  rubric_version_id text NOT NULL REFERENCES rubric_template_versions(id) ON DELETE RESTRICT,
  status text NOT NULL CHECK (status IN ('draft', 'published', 'closed', 'archived')),
  start_at timestamptz,
  due_at timestamptz,
  published_at timestamptz,
  created_by text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (due_at IS NULL OR start_at IS NULL OR due_at > start_at)
);

CREATE INDEX IF NOT EXISTS idx_experiments_course_status ON experiments (course_id, status);
CREATE INDEX IF NOT EXISTS idx_experiments_rubric_version_id ON experiments (rubric_version_id);
CREATE INDEX IF NOT EXISTS idx_experiments_due_at ON experiments (due_at);

CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_created_at ON audit_logs (actor_id, created_at DESC);

CREATE OR REPLACE FUNCTION prevent_published_rubric_version_update()
RETURNS trigger AS $$
BEGIN
  IF OLD.status = 'published' THEN
    RAISE EXCEPTION 'published rubric versions are immutable';
  END IF;
  IF TG_OP = 'DELETE' THEN
    RETURN OLD;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_published_rubric_version_update ON rubric_template_versions;
CREATE TRIGGER trg_prevent_published_rubric_version_update
BEFORE UPDATE OR DELETE ON rubric_template_versions
FOR EACH ROW EXECUTE FUNCTION prevent_published_rubric_version_update();

CREATE OR REPLACE FUNCTION prevent_published_rubric_metric_update()
RETURNS trigger AS $$
BEGIN
  IF EXISTS (SELECT 1 FROM rubric_template_versions WHERE id = OLD.version_id AND status = 'published') THEN
    RAISE EXCEPTION 'published rubric metrics are immutable';
  END IF;
  IF TG_OP = 'DELETE' THEN
    RETURN OLD;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_published_rubric_metric_update ON rubric_metrics;
CREATE TRIGGER trg_prevent_published_rubric_metric_update
BEFORE UPDATE OR DELETE ON rubric_metrics
FOR EACH ROW EXECUTE FUNCTION prevent_published_rubric_metric_update();
