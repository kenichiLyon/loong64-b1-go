package teaching

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kenichiLyon/loong64-b1-go/internal/database"
)

type PostgresRepository struct {
	db *database.Pool
}

func NewPostgresRepository(db *database.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) pool() (pgxPool, error) {
	if r == nil || r.db == nil || r.db.Raw() == nil {
		return nil, unavailableError("postgres teaching repository is not configured", nil)
	}
	return r.db.Raw(), nil
}

type pgxPool interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (r *PostgresRepository) CreateUser(ctx context.Context, user User, roles []Role, audit AuditEntry) (User, error) {
	pool, err := r.pool()
	if err != nil {
		return User{}, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return User{}, err
	}
	defer rollback(ctx, tx)
	if err := tx.QueryRow(ctx, `
INSERT INTO users (id, username, display_name, email, student_no, employee_no, status)
VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7)
RETURNING id, username, display_name, COALESCE(email, ''), COALESCE(student_no, ''), COALESCE(employee_no, ''), status, created_at, updated_at`,
		user.ID, user.Username, user.DisplayName, user.Email, user.StudentNo, user.EmployeeNo, user.Status,
	).Scan(&user.ID, &user.Username, &user.DisplayName, &user.Email, &user.StudentNo, &user.EmployeeNo, &user.Status, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return User{}, mapDBError(err)
	}
	for _, role := range roles {
		if _, err := tx.Exec(ctx, `INSERT INTO user_roles (user_id, role) VALUES ($1, $2)`, user.ID, role); err != nil {
			return User{}, mapDBError(err)
		}
	}
	user.Roles = roles
	if err := insertAudit(ctx, tx, audit); err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return user, nil
}

func (r *PostgresRepository) ListUsers(ctx context.Context, limit int) ([]User, error) {
	pool, err := r.pool()
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
WITH selected AS (
  SELECT id, username, display_name, COALESCE(email, '') AS email, COALESCE(student_no, '') AS student_no,
         COALESCE(employee_no, '') AS employee_no, status, created_at, updated_at
  FROM users
  ORDER BY created_at DESC, id
  LIMIT $1
)
SELECT selected.id, selected.username, selected.display_name, selected.email, selected.student_no, selected.employee_no,
       selected.status, selected.created_at, selected.updated_at, user_roles.role
FROM selected
LEFT JOIN user_roles ON user_roles.user_id = selected.id
ORDER BY selected.created_at DESC, selected.id, user_roles.role`, limit)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	byID := make(map[string]*User)
	order := make([]string, 0)
	for rows.Next() {
		var user User
		var role pgtype.Text
		if err := rows.Scan(&user.ID, &user.Username, &user.DisplayName, &user.Email, &user.StudentNo, &user.EmployeeNo, &user.Status, &user.CreatedAt, &user.UpdatedAt, &role); err != nil {
			return nil, mapDBError(err)
		}
		current := byID[user.ID]
		if current == nil {
			user.Roles = []Role{}
			byID[user.ID] = &user
			order = append(order, user.ID)
			current = &user
		}
		if role.Valid {
			current.Roles = append(current.Roles, Role(role.String))
		}
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBError(err)
	}
	users := make([]User, 0, len(order))
	for _, id := range order {
		users = append(users, *byID[id])
	}
	return users, nil
}

func (r *PostgresRepository) SetUserRoles(ctx context.Context, userID string, roles []Role, audit AuditEntry) error {
	pool, err := r.pool()
	if err != nil {
		return err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer rollback(ctx, tx)
	var exists int
	if err := tx.QueryRow(ctx, `SELECT 1 FROM users WHERE id = $1`, userID).Scan(&exists); err != nil {
		return mapDBError(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID); err != nil {
		return mapDBError(err)
	}
	for _, role := range roles {
		if _, err := tx.Exec(ctx, `INSERT INTO user_roles (user_id, role) VALUES ($1, $2)`, userID, role); err != nil {
			return mapDBError(err)
		}
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) UserHasRole(ctx context.Context, userID string, role Role) (bool, error) {
	pool, err := r.pool()
	if err != nil {
		return false, err
	}
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1 AND role = $2)`, userID, role).Scan(&exists); err != nil {
		return false, mapDBError(err)
	}
	return exists, nil
}

func (r *PostgresRepository) CreateClass(ctx context.Context, class Class, audit AuditEntry) (Class, error) {
	pool, err := r.pool()
	if err != nil {
		return Class{}, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Class{}, err
	}
	defer rollback(ctx, tx)
	if err := tx.QueryRow(ctx, `
INSERT INTO classes (id, code, name, grade_year, major, status)
VALUES ($1, $2, $3, NULLIF($4, 0), NULLIF($5, ''), $6)
RETURNING id, code, name, COALESCE(grade_year, 0), COALESCE(major, ''), status, created_at, updated_at`,
		class.ID, class.Code, class.Name, class.GradeYear, class.Major, class.Status,
	).Scan(&class.ID, &class.Code, &class.Name, &class.GradeYear, &class.Major, &class.Status, &class.CreatedAt, &class.UpdatedAt); err != nil {
		return Class{}, mapDBError(err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return Class{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Class{}, err
	}
	return class, nil
}

func (r *PostgresRepository) CreateCourse(ctx context.Context, course Course, audit AuditEntry) (Course, error) {
	pool, err := r.pool()
	if err != nil {
		return Course{}, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Course{}, err
	}
	defer rollback(ctx, tx)
	if err := tx.QueryRow(ctx, `
INSERT INTO courses (id, code, name, term, status, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, code, name, term, status, created_by, created_at, updated_at`,
		course.ID, course.Code, course.Name, course.Term, course.Status, course.CreatedBy,
	).Scan(&course.ID, &course.Code, &course.Name, &course.Term, &course.Status, &course.CreatedBy, &course.CreatedAt, &course.UpdatedAt); err != nil {
		return Course{}, mapDBError(err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return Course{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Course{}, err
	}
	return course, nil
}

func (r *PostgresRepository) AddCourseClass(ctx context.Context, courseID, classID string, audit AuditEntry) error {
	pool, err := r.pool()
	if err != nil {
		return err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer rollback(ctx, tx)
	if _, err := tx.Exec(ctx, `INSERT INTO course_classes (course_id, class_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, courseID, classID); err != nil {
		return mapDBError(err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) TeacherCanEditCourse(ctx context.Context, courseID, teacherID string) (bool, error) {
	pool, err := r.pool()
	if err != nil {
		return false, err
	}
	var allowed bool
	if err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1 FROM course_teachers
  WHERE course_id = $1 AND teacher_id = $2 AND permission IN ('owner', 'editor')
)`, courseID, teacherID).Scan(&allowed); err != nil {
		return false, mapDBError(err)
	}
	return allowed, nil
}

func (r *PostgresRepository) AssignTeacher(ctx context.Context, assignment CourseTeacher, audit AuditEntry) error {
	pool, err := r.pool()
	if err != nil {
		return err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer rollback(ctx, tx)
	if _, err := tx.Exec(ctx, `
INSERT INTO course_teachers (course_id, teacher_id, permission)
VALUES ($1, $2, $3)
ON CONFLICT (course_id, teacher_id) DO UPDATE SET permission = EXCLUDED.permission`, assignment.CourseID, assignment.TeacherID, assignment.Permission); err != nil {
		return mapDBError(err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) EnrollStudent(ctx context.Context, enrollment Enrollment, audit AuditEntry) error {
	pool, err := r.pool()
	if err != nil {
		return err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer rollback(ctx, tx)
	var classID any
	if enrollment.ClassID != "" {
		classID = enrollment.ClassID
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO enrollments (course_id, class_id, student_id, status)
VALUES ($1, $2, $3, $4)
ON CONFLICT (course_id, student_id) DO UPDATE SET class_id = EXCLUDED.class_id, status = EXCLUDED.status`,
		enrollment.CourseID, classID, enrollment.StudentID, enrollment.Status); err != nil {
		return mapDBError(err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) CreateRubricTemplate(ctx context.Context, template RubricTemplate, audit AuditEntry) (RubricTemplate, error) {
	pool, err := r.pool()
	if err != nil {
		return RubricTemplate{}, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return RubricTemplate{}, err
	}
	defer rollback(ctx, tx)
	if err := tx.QueryRow(ctx, `
INSERT INTO rubric_templates (id, name, description, owner_id, scope, status)
VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6)
RETURNING id, name, COALESCE(description, ''), owner_id, scope, status, created_at, updated_at`,
		template.ID, template.Name, template.Description, template.OwnerID, template.Scope, template.Status,
	).Scan(&template.ID, &template.Name, &template.Description, &template.OwnerID, &template.Scope, &template.Status, &template.CreatedAt, &template.UpdatedAt); err != nil {
		return RubricTemplate{}, mapDBError(err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return RubricTemplate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RubricTemplate{}, err
	}
	return template, nil
}

func (r *PostgresRepository) RubricTemplateOwner(ctx context.Context, templateID string) (string, error) {
	pool, err := r.pool()
	if err != nil {
		return "", err
	}
	var ownerID string
	if err := pool.QueryRow(ctx, `SELECT owner_id FROM rubric_templates WHERE id = $1`, templateID).Scan(&ownerID); err != nil {
		return "", mapDBError(err)
	}
	return ownerID, nil
}

func (r *PostgresRepository) CreateRubricVersion(ctx context.Context, version RubricTemplateVersion, metrics []Metric, audit AuditEntry) (RubricTemplateVersion, []Metric, error) {
	pool, err := r.pool()
	if err != nil {
		return RubricTemplateVersion{}, nil, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return RubricTemplateVersion{}, nil, err
	}
	defer rollback(ctx, tx)
	var templateID string
	if err := tx.QueryRow(ctx, `SELECT id FROM rubric_templates WHERE id = $1 FOR UPDATE`, version.TemplateID).Scan(&templateID); err != nil {
		return RubricTemplateVersion{}, nil, mapDBError(err)
	}
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(version_no), 0) + 1 FROM rubric_template_versions WHERE template_id = $1`, version.TemplateID).Scan(&version.VersionNo); err != nil {
		return RubricTemplateVersion{}, nil, mapDBError(err)
	}
	if err := scanVersion(tx.QueryRow(ctx, `
INSERT INTO rubric_template_versions (id, template_id, version_no, status, weight_mode, total_weight_bps, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, template_id, version_no, status, weight_mode, total_weight_bps, published_at, created_by, created_at`,
		version.ID, version.TemplateID, version.VersionNo, version.Status, version.WeightMode, version.TotalWeightBPS, version.CreatedBy), &version); err != nil {
		return RubricTemplateVersion{}, nil, mapDBError(err)
	}
	createdMetrics := make([]Metric, 0, len(metrics))
	for _, metric := range metrics {
		metric.VersionID = version.ID
		if err := tx.QueryRow(ctx, `
INSERT INTO rubric_metrics (id, version_id, code, name, description, weight_bps, max_score, sort_order, required_evidence)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, $9)
RETURNING id, version_id, code, name, COALESCE(description, ''), weight_bps, max_score, sort_order, required_evidence`,
			metric.ID, metric.VersionID, metric.Code, metric.Name, metric.Description, metric.WeightBPS, metric.MaxScore, metric.SortOrder, metric.RequiredEvidence,
		).Scan(&metric.ID, &metric.VersionID, &metric.Code, &metric.Name, &metric.Description, &metric.WeightBPS, &metric.MaxScore, &metric.SortOrder, &metric.RequiredEvidence); err != nil {
			return RubricTemplateVersion{}, nil, mapDBError(err)
		}
		createdMetrics = append(createdMetrics, metric)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return RubricTemplateVersion{}, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RubricTemplateVersion{}, nil, err
	}
	return version, createdMetrics, nil
}

func (r *PostgresRepository) RubricVersionOwner(ctx context.Context, versionID string) (string, error) {
	pool, err := r.pool()
	if err != nil {
		return "", err
	}
	var ownerID string
	if err := pool.QueryRow(ctx, `
SELECT rubric_templates.owner_id
FROM rubric_template_versions
JOIN rubric_templates ON rubric_templates.id = rubric_template_versions.template_id
WHERE rubric_template_versions.id = $1`, versionID).Scan(&ownerID); err != nil {
		return "", mapDBError(err)
	}
	return ownerID, nil
}

func (r *PostgresRepository) PublishRubricVersion(ctx context.Context, versionID string, audit AuditEntry) (RubricTemplateVersion, error) {
	pool, err := r.pool()
	if err != nil {
		return RubricTemplateVersion{}, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return RubricTemplateVersion{}, err
	}
	defer rollback(ctx, tx)
	var version RubricTemplateVersion
	if err := scanVersion(tx.QueryRow(ctx, `
SELECT id, template_id, version_no, status, weight_mode, total_weight_bps, published_at, created_by, created_at
FROM rubric_template_versions WHERE id = $1 FOR UPDATE`, versionID), &version); err != nil {
		return RubricTemplateVersion{}, mapDBError(err)
	}
	if version.Status != "draft" {
		return RubricTemplateVersion{}, conflictError("only draft rubric versions can be published")
	}
	metricInputs, err := loadMetricInputs(ctx, tx, versionID)
	if err != nil {
		return RubricTemplateVersion{}, err
	}
	if err := ValidateMetrics(version.WeightMode, metricInputs); err != nil {
		return RubricTemplateVersion{}, err
	}
	if err := scanVersion(tx.QueryRow(ctx, `
UPDATE rubric_template_versions
SET status = 'published', published_at = now()
WHERE id = $1 AND status = 'draft'
RETURNING id, template_id, version_no, status, weight_mode, total_weight_bps, published_at, created_by, created_at`, versionID), &version); err != nil {
		return RubricTemplateVersion{}, mapDBError(err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return RubricTemplateVersion{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RubricTemplateVersion{}, err
	}
	return version, nil
}

func (r *PostgresRepository) RubricVersionStatus(ctx context.Context, versionID string) (string, error) {
	pool, err := r.pool()
	if err != nil {
		return "", err
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM rubric_template_versions WHERE id = $1`, versionID).Scan(&status); err != nil {
		return "", mapDBError(err)
	}
	return status, nil
}

func (r *PostgresRepository) CreateExperiment(ctx context.Context, experiment Experiment, audit AuditEntry) (Experiment, error) {
	pool, err := r.pool()
	if err != nil {
		return Experiment{}, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Experiment{}, err
	}
	defer rollback(ctx, tx)
	if err := scanExperiment(tx.QueryRow(ctx, `
INSERT INTO experiments (id, course_id, title, description, submission_spec, rubric_version_id, status, start_at, due_at, created_by)
VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9, $10)
RETURNING id, course_id, title, COALESCE(description, ''), submission_spec, rubric_version_id, status, start_at, due_at, published_at, created_by, created_at, updated_at`,
		experiment.ID, experiment.CourseID, experiment.Title, experiment.Description, experiment.SubmissionSpec, experiment.RubricVersionID, experiment.Status, experiment.StartAt, experiment.DueAt, experiment.CreatedBy), &experiment); err != nil {
		return Experiment{}, mapDBError(err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return Experiment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Experiment{}, err
	}
	return experiment, nil
}

func (r *PostgresRepository) ExperimentCourseID(ctx context.Context, experimentID string) (string, error) {
	pool, err := r.pool()
	if err != nil {
		return "", err
	}
	var courseID string
	if err := pool.QueryRow(ctx, `SELECT course_id FROM experiments WHERE id = $1`, experimentID).Scan(&courseID); err != nil {
		return "", mapDBError(err)
	}
	return courseID, nil
}

func (r *PostgresRepository) PublishExperiment(ctx context.Context, experimentID string, audit AuditEntry) (Experiment, error) {
	pool, err := r.pool()
	if err != nil {
		return Experiment{}, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Experiment{}, err
	}
	defer rollback(ctx, tx)
	var experiment Experiment
	if err := scanExperiment(tx.QueryRow(ctx, `
UPDATE experiments
SET status = 'published', published_at = now(), updated_at = now()
WHERE id = $1 AND status = 'draft'
RETURNING id, course_id, title, COALESCE(description, ''), submission_spec, rubric_version_id, status, start_at, due_at, published_at, created_by, created_at, updated_at`, experimentID), &experiment); err != nil {
		return Experiment{}, mapDBError(err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return Experiment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Experiment{}, err
	}
	return experiment, nil
}

func (r *PostgresRepository) ExperimentSubmissionAccess(ctx context.Context, experimentID, studentID string) (ExperimentSubmissionAccess, error) {
	pool, err := r.pool()
	if err != nil {
		return ExperimentSubmissionAccess{}, err
	}
	var access ExperimentSubmissionAccess
	var dueAt pgtype.Timestamptz
	if err := pool.QueryRow(ctx, `
SELECT experiments.course_id, experiments.status, experiments.due_at,
       EXISTS (
         SELECT 1 FROM enrollments
         WHERE enrollments.course_id = experiments.course_id
           AND enrollments.student_id = $2
           AND enrollments.status = 'active'
       ) AS enrolled
FROM experiments
WHERE experiments.id = $1`, experimentID, studentID).Scan(&access.CourseID, &access.Status, &dueAt, &access.Enrolled); err != nil {
		return ExperimentSubmissionAccess{}, mapDBError(err)
	}
	access.DueAt = nullableTime(dueAt)
	return access, nil
}

func (r *PostgresRepository) CreateSubmission(ctx context.Context, submission Submission, audit AuditEntry) (Submission, error) {
	pool, err := r.pool()
	if err != nil {
		return Submission{}, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Submission{}, err
	}
	defer rollback(ctx, tx)
	if err := scanSubmission(tx.QueryRow(ctx, `
INSERT INTO submissions (id, experiment_id, student_id, status, attempt_no)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, experiment_id, student_id, status, attempt_no, submitted_at, created_at, updated_at`,
		submission.ID, submission.ExperimentID, submission.StudentID, submission.Status, submission.AttemptNo), &submission); err != nil {
		return Submission{}, mapDBError(err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return Submission{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Submission{}, err
	}
	return submission, nil
}

func (r *PostgresRepository) StudentOwnsSubmission(ctx context.Context, submissionID, studentID string) (bool, error) {
	pool, err := r.pool()
	if err != nil {
		return false, err
	}
	var owns bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM submissions WHERE id = $1 AND student_id = $2)`, submissionID, studentID).Scan(&owns); err != nil {
		return false, mapDBError(err)
	}
	return owns, nil
}

func (r *PostgresRepository) SubmissionCourseID(ctx context.Context, submissionID string) (string, error) {
	pool, err := r.pool()
	if err != nil {
		return "", err
	}
	var courseID string
	if err := pool.QueryRow(ctx, `
SELECT experiments.course_id
FROM submissions
JOIN experiments ON experiments.id = submissions.experiment_id
WHERE submissions.id = $1`, submissionID).Scan(&courseID); err != nil {
		return "", mapDBError(err)
	}
	return courseID, nil
}

func (r *PostgresRepository) SubmissionArtifactCount(ctx context.Context, submissionID string) (int, error) {
	pool, err := r.pool()
	if err != nil {
		return 0, err
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM artifacts WHERE submission_id = $1`, submissionID).Scan(&count); err != nil {
		return 0, mapDBError(err)
	}
	return count, nil
}

func (r *PostgresRepository) CreateArtifact(ctx context.Context, artifact Artifact, extraction ExtractedContent, job *QueuedJob, audit AuditEntry) (ArtifactWithExtraction, error) {
	pool, err := r.pool()
	if err != nil {
		return ArtifactWithExtraction{}, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ArtifactWithExtraction{}, err
	}
	defer rollback(ctx, tx)
	if err := scanArtifact(tx.QueryRow(ctx, `
INSERT INTO artifacts (id, submission_id, artifact_kind, original_name, content_type, byte_size, sha256_hex, storage_key, source_url, status, metadata, created_by)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), $10, $11, $12)
RETURNING id, submission_id, artifact_kind, original_name, COALESCE(content_type, ''), byte_size, COALESCE(sha256_hex, ''), COALESCE(storage_key, ''), COALESCE(source_url, ''), status, metadata, created_by, created_at`,
		artifact.ID, artifact.SubmissionID, artifact.Kind, artifact.OriginalName, artifact.ContentType, artifact.ByteSize, artifact.SHA256Hex, artifact.StorageKey, artifact.SourceURL, artifact.Status, defaultJSON(artifact.Metadata), artifact.CreatedBy), &artifact); err != nil {
		return ArtifactWithExtraction{}, mapDBError(err)
	}
	if err := scanExtraction(tx.QueryRow(ctx, `
INSERT INTO extracted_contents (id, artifact_id, status, text_excerpt, metadata, error)
VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))
RETURNING id, artifact_id, status, text_excerpt, metadata, COALESCE(error, ''), created_at, updated_at`,
		extraction.ID, extraction.ArtifactID, extraction.Status, extraction.TextExcerpt, defaultJSON(extraction.Metadata), extraction.Error), &extraction); err != nil {
		return ArtifactWithExtraction{}, mapDBError(err)
	}
	jobID := ""
	if job != nil {
		payload := defaultJSON(job.Payload)
		if _, err := tx.Exec(ctx, `
INSERT INTO jobs (id, job_type, status, payload)
VALUES ($1, $2, 'queued', $3)`, job.ID, job.Type, payload); err != nil {
			return ArtifactWithExtraction{}, mapDBError(err)
		}
		jobID = job.ID
	}
	nextStatus := "parsed"
	if extraction.Status == "queued" {
		nextStatus = "parsing"
	}
	if _, err := tx.Exec(ctx, `UPDATE submissions SET status = $2, updated_at = now() WHERE id = $1`, artifact.SubmissionID, nextStatus); err != nil {
		return ArtifactWithExtraction{}, mapDBError(err)
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return ArtifactWithExtraction{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ArtifactWithExtraction{}, err
	}
	return ArtifactWithExtraction{Artifact: artifact, Extraction: extraction, JobID: jobID}, nil
}

func (r *PostgresRepository) ListSubmissionsForExperiment(ctx context.Context, experimentID string, limit int) ([]Submission, error) {
	pool, err := r.pool()
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
SELECT id, experiment_id, student_id, status, attempt_no, submitted_at, created_at, updated_at
FROM submissions
WHERE experiment_id = $1
ORDER BY created_at DESC, id
LIMIT $2`, experimentID, limit)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	submissions := make([]Submission, 0)
	for rows.Next() {
		var submission Submission
		if err := scanSubmission(rows, &submission); err != nil {
			return nil, mapDBError(err)
		}
		submissions = append(submissions, submission)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBError(err)
	}
	return submissions, nil
}

func (r *PostgresRepository) GetSubmissionDetail(ctx context.Context, submissionID string) (SubmissionDetail, error) {
	pool, err := r.pool()
	if err != nil {
		return SubmissionDetail{}, err
	}
	var detail SubmissionDetail
	if err := scanSubmission(pool.QueryRow(ctx, `
SELECT id, experiment_id, student_id, status, attempt_no, submitted_at, created_at, updated_at
FROM submissions
WHERE id = $1`, submissionID), &detail.Submission); err != nil {
		return SubmissionDetail{}, mapDBError(err)
	}
	rows, err := pool.Query(ctx, `
SELECT artifacts.id, artifacts.submission_id, artifacts.artifact_kind, artifacts.original_name,
       COALESCE(artifacts.content_type, ''), artifacts.byte_size, COALESCE(artifacts.sha256_hex, ''),
       COALESCE(artifacts.storage_key, ''), COALESCE(artifacts.source_url, ''), artifacts.status,
       artifacts.metadata, artifacts.created_by, artifacts.created_at,
       extracted_contents.id, extracted_contents.artifact_id, extracted_contents.status,
       extracted_contents.text_excerpt, extracted_contents.metadata, COALESCE(extracted_contents.error, ''),
       extracted_contents.created_at, extracted_contents.updated_at
FROM artifacts
JOIN extracted_contents ON extracted_contents.artifact_id = artifacts.id
WHERE artifacts.submission_id = $1
ORDER BY artifacts.created_at, artifacts.id`, submissionID)
	if err != nil {
		return SubmissionDetail{}, mapDBError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var item ArtifactWithExtraction
		if err := scanArtifactAndExtraction(rows, &item.Artifact, &item.Extraction); err != nil {
			return SubmissionDetail{}, mapDBError(err)
		}
		detail.Artifacts = append(detail.Artifacts, item)
	}
	if err := rows.Err(); err != nil {
		return SubmissionDetail{}, mapDBError(err)
	}
	return detail, nil
}

func (r *PostgresRepository) GetEvaluationContext(ctx context.Context, submissionID string) (EvaluationContext, error) {
	pool, err := r.pool()
	if err != nil {
		return EvaluationContext{}, err
	}
	detail, err := r.GetSubmissionDetail(ctx, submissionID)
	if err != nil {
		return EvaluationContext{}, err
	}
	var evalCtx EvaluationContext
	evalCtx.Submission = detail.Submission
	evalCtx.Artifacts = detail.Artifacts
	if err := scanExperiment(pool.QueryRow(ctx, `
SELECT experiments.id, experiments.course_id, experiments.title, COALESCE(experiments.description, ''),
       experiments.submission_spec, experiments.rubric_version_id, experiments.status,
       experiments.start_at, experiments.due_at, experiments.published_at, experiments.created_by,
       experiments.created_at, experiments.updated_at
FROM experiments
WHERE experiments.id = $1`, detail.Submission.ExperimentID), &evalCtx.Experiment); err != nil {
		return EvaluationContext{}, mapDBError(err)
	}
	rows, err := pool.Query(ctx, `
SELECT id, version_id, code, name, COALESCE(description, ''), weight_bps, max_score, sort_order, required_evidence
FROM rubric_metrics
WHERE version_id = $1
ORDER BY sort_order`, evalCtx.Experiment.RubricVersionID)
	if err != nil {
		return EvaluationContext{}, mapDBError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var metric Metric
		if err := scanMetric(rows, &metric); err != nil {
			return EvaluationContext{}, mapDBError(err)
		}
		evalCtx.Metrics = append(evalCtx.Metrics, metric)
	}
	if err := rows.Err(); err != nil {
		return EvaluationContext{}, mapDBError(err)
	}
	return evalCtx, nil
}

func (r *PostgresRepository) CreateInitialEvaluation(ctx context.Context, result EvaluationResult, findings []RuleCheckFinding, scores []MetricScore, llmLog *LLMCallLog, audit AuditEntry) (EvaluationResultDetail, error) {
	pool, err := r.pool()
	if err != nil {
		return EvaluationResultDetail{}, err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return EvaluationResultDetail{}, err
	}
	defer rollback(ctx, tx)
	if err := scanEvaluationResult(tx.QueryRow(ctx, `
INSERT INTO evaluation_results (id, submission_id, experiment_id, rubric_version_id, status, rule_status, llm_status,
                                prompt_version, evidence_snapshot, rule_summary, llm_summary, needs_teacher_review, error, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING id, submission_id, experiment_id, rubric_version_id, status, rule_status, llm_status, prompt_version,
          evidence_snapshot, rule_summary, llm_summary, needs_teacher_review, error, created_by, created_at, updated_at`,
		result.ID, result.SubmissionID, result.ExperimentID, result.RubricVersionID, result.Status, result.RuleStatus, result.LLMStatus,
		result.PromptVersion, defaultJSON(result.EvidenceSnapshot), defaultJSON(result.RuleSummary), result.LLMSummary, result.NeedsTeacherReview,
		result.Error, result.CreatedBy), &result); err != nil {
		return EvaluationResultDetail{}, mapDBError(err)
	}
	createdFindings := make([]RuleCheckFinding, 0, len(findings))
	for _, finding := range findings {
		finding.EvaluationResultID = result.ID
		if err := scanRuleFinding(tx.QueryRow(ctx, `
INSERT INTO rule_check_findings (id, evaluation_result_id, category, severity, message, evidence_ref)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, evaluation_result_id, category, severity, message, evidence_ref, created_at`,
			finding.ID, finding.EvaluationResultID, finding.Category, finding.Severity, finding.Message, finding.EvidenceRef), &finding); err != nil {
			return EvaluationResultDetail{}, mapDBError(err)
		}
		createdFindings = append(createdFindings, finding)
	}
	createdScores := make([]MetricScore, 0, len(scores))
	for _, score := range scores {
		score.EvaluationResultID = result.ID
		if err := scanMetricScore(tx.QueryRow(ctx, `
INSERT INTO metric_scores (id, evaluation_result_id, metric_id, metric_code, source, suggested_score, max_score,
                           confidence_bps, rationale, evidence_refs)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, evaluation_result_id, metric_id, metric_code, source, suggested_score, max_score, confidence_bps,
          rationale, evidence_refs, created_at`,
			score.ID, score.EvaluationResultID, score.MetricID, score.MetricCode, score.Source, score.SuggestedScore, score.MaxScore,
			score.ConfidenceBPS, score.Rationale, defaultJSON(score.EvidenceRefs)), &score); err != nil {
			return EvaluationResultDetail{}, mapDBError(err)
		}
		createdScores = append(createdScores, score)
	}
	if llmLog != nil {
		llmLog.EvaluationResultID = result.ID
		if _, err := tx.Exec(ctx, `
INSERT INTO llm_call_logs (id, evaluation_result_id, provider, model, prompt_version, input_hash, output, status, error,
                           latency_ms, prompt_tokens, completion_tokens)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			llmLog.ID, llmLog.EvaluationResultID, llmLog.Provider, llmLog.Model, llmLog.PromptVersion, llmLog.InputHash,
			defaultJSON(llmLog.Output), llmLog.Status, llmLog.Error, llmLog.LatencyMS, llmLog.PromptTokens, llmLog.CompletionTokens); err != nil {
			return EvaluationResultDetail{}, mapDBError(err)
		}
	}
	if err := insertAudit(ctx, tx, audit); err != nil {
		return EvaluationResultDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return EvaluationResultDetail{}, err
	}
	return EvaluationResultDetail{Result: result, Findings: createdFindings, Scores: createdScores}, nil
}

func (r *PostgresRepository) GetLatestEvaluation(ctx context.Context, submissionID string) (EvaluationResultDetail, error) {
	pool, err := r.pool()
	if err != nil {
		return EvaluationResultDetail{}, err
	}
	var detail EvaluationResultDetail
	if err := scanEvaluationResult(pool.QueryRow(ctx, `
SELECT id, submission_id, experiment_id, rubric_version_id, status, rule_status, llm_status, prompt_version,
       evidence_snapshot, rule_summary, llm_summary, needs_teacher_review, error, created_by, created_at, updated_at
FROM evaluation_results
WHERE submission_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1`, submissionID), &detail.Result); err != nil {
		return EvaluationResultDetail{}, mapDBError(err)
	}
	findingRows, err := pool.Query(ctx, `
SELECT id, evaluation_result_id, category, severity, message, evidence_ref, created_at
FROM rule_check_findings
WHERE evaluation_result_id = $1
ORDER BY created_at, id`, detail.Result.ID)
	if err != nil {
		return EvaluationResultDetail{}, mapDBError(err)
	}
	defer findingRows.Close()
	for findingRows.Next() {
		var finding RuleCheckFinding
		if err := scanRuleFinding(findingRows, &finding); err != nil {
			return EvaluationResultDetail{}, mapDBError(err)
		}
		detail.Findings = append(detail.Findings, finding)
	}
	if err := findingRows.Err(); err != nil {
		return EvaluationResultDetail{}, mapDBError(err)
	}
	scoreRows, err := pool.Query(ctx, `
SELECT id, evaluation_result_id, metric_id, metric_code, source, suggested_score, max_score, confidence_bps,
       rationale, evidence_refs, created_at
FROM metric_scores
WHERE evaluation_result_id = $1
ORDER BY source, metric_code`, detail.Result.ID)
	if err != nil {
		return EvaluationResultDetail{}, mapDBError(err)
	}
	defer scoreRows.Close()
	for scoreRows.Next() {
		var score MetricScore
		if err := scanMetricScore(scoreRows, &score); err != nil {
			return EvaluationResultDetail{}, mapDBError(err)
		}
		detail.Scores = append(detail.Scores, score)
	}
	if err := scoreRows.Err(); err != nil {
		return EvaluationResultDetail{}, mapDBError(err)
	}
	return detail, nil
}

func rollback(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func insertAudit(ctx context.Context, tx pgx.Tx, audit AuditEntry) error {
	if audit.Action == "" || audit.TargetType == "" {
		return nil
	}
	detail := audit.Detail
	if len(detail) == 0 {
		detail = json.RawMessage(`{}`)
	}
	_, err := tx.Exec(ctx, `
INSERT INTO audit_logs (actor_id, action, target_type, target_id, detail, request_id)
VALUES (NULLIF($1, ''), $2, $3, NULLIF($4, ''), $5, NULLIF($6, ''))`,
		audit.ActorID, audit.Action, audit.TargetType, audit.TargetID, detail, audit.RequestID)
	return mapDBError(err)
}

func loadMetricInputs(ctx context.Context, tx pgx.Tx, versionID string) ([]MetricInput, error) {
	rows, err := tx.Query(ctx, `
SELECT code, name, COALESCE(description, ''), weight_bps, max_score, sort_order, required_evidence
FROM rubric_metrics
WHERE version_id = $1
ORDER BY sort_order`, versionID)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	metrics := make([]MetricInput, 0)
	for rows.Next() {
		var metric MetricInput
		if err := rows.Scan(&metric.Code, &metric.Name, &metric.Description, &metric.WeightBPS, &metric.MaxScore, &metric.SortOrder, &metric.RequiredEvidence); err != nil {
			return nil, mapDBError(err)
		}
		metrics = append(metrics, metric)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDBError(err)
	}
	return metrics, nil
}

func scanVersion(row pgx.Row, version *RubricTemplateVersion) error {
	var publishedAt pgtype.Timestamptz
	var mode string
	if err := row.Scan(&version.ID, &version.TemplateID, &version.VersionNo, &version.Status, &mode, &version.TotalWeightBPS, &publishedAt, &version.CreatedBy, &version.CreatedAt); err != nil {
		return err
	}
	version.WeightMode = WeightMode(mode)
	version.PublishedAt = nullableTime(publishedAt)
	return nil
}

func scanExperiment(row pgx.Row, experiment *Experiment) error {
	var startAt, dueAt, publishedAt pgtype.Timestamptz
	if err := row.Scan(&experiment.ID, &experiment.CourseID, &experiment.Title, &experiment.Description, &experiment.SubmissionSpec, &experiment.RubricVersionID, &experiment.Status, &startAt, &dueAt, &publishedAt, &experiment.CreatedBy, &experiment.CreatedAt, &experiment.UpdatedAt); err != nil {
		return err
	}
	experiment.StartAt = nullableTime(startAt)
	experiment.DueAt = nullableTime(dueAt)
	experiment.PublishedAt = nullableTime(publishedAt)
	return nil
}

func scanSubmission(row pgx.Row, submission *Submission) error {
	var submittedAt pgtype.Timestamptz
	if err := row.Scan(&submission.ID, &submission.ExperimentID, &submission.StudentID, &submission.Status, &submission.AttemptNo, &submittedAt, &submission.CreatedAt, &submission.UpdatedAt); err != nil {
		return err
	}
	submission.SubmittedAt = nullableTime(submittedAt)
	return nil
}

func scanArtifact(row pgx.Row, artifact *Artifact) error {
	var kind string
	if err := row.Scan(&artifact.ID, &artifact.SubmissionID, &kind, &artifact.OriginalName, &artifact.ContentType, &artifact.ByteSize, &artifact.SHA256Hex, &artifact.StorageKey, &artifact.SourceURL, &artifact.Status, &artifact.Metadata, &artifact.CreatedBy, &artifact.CreatedAt); err != nil {
		return err
	}
	artifact.Kind = ArtifactKind(kind)
	return nil
}

func scanExtraction(row pgx.Row, extraction *ExtractedContent) error {
	if err := row.Scan(&extraction.ID, &extraction.ArtifactID, &extraction.Status, &extraction.TextExcerpt, &extraction.Metadata, &extraction.Error, &extraction.CreatedAt, &extraction.UpdatedAt); err != nil {
		return err
	}
	return nil
}

func scanArtifactAndExtraction(row pgx.Row, artifact *Artifact, extraction *ExtractedContent) error {
	var kind string
	if err := row.Scan(
		&artifact.ID, &artifact.SubmissionID, &kind, &artifact.OriginalName, &artifact.ContentType, &artifact.ByteSize, &artifact.SHA256Hex, &artifact.StorageKey, &artifact.SourceURL, &artifact.Status, &artifact.Metadata, &artifact.CreatedBy, &artifact.CreatedAt,
		&extraction.ID, &extraction.ArtifactID, &extraction.Status, &extraction.TextExcerpt, &extraction.Metadata, &extraction.Error, &extraction.CreatedAt, &extraction.UpdatedAt,
	); err != nil {
		return err
	}
	artifact.Kind = ArtifactKind(kind)
	return nil
}

func scanMetric(row pgx.Row, metric *Metric) error {
	return row.Scan(&metric.ID, &metric.VersionID, &metric.Code, &metric.Name, &metric.Description, &metric.WeightBPS, &metric.MaxScore, &metric.SortOrder, &metric.RequiredEvidence)
}

func scanEvaluationResult(row pgx.Row, result *EvaluationResult) error {
	var status, ruleStatus, llmStatus string
	if err := row.Scan(&result.ID, &result.SubmissionID, &result.ExperimentID, &result.RubricVersionID, &status, &ruleStatus, &llmStatus,
		&result.PromptVersion, &result.EvidenceSnapshot, &result.RuleSummary, &result.LLMSummary, &result.NeedsTeacherReview,
		&result.Error, &result.CreatedBy, &result.CreatedAt, &result.UpdatedAt); err != nil {
		return err
	}
	result.Status = EvaluationStatus(status)
	result.RuleStatus = EvaluationStepStatus(ruleStatus)
	result.LLMStatus = EvaluationStepStatus(llmStatus)
	return nil
}

func scanRuleFinding(row pgx.Row, finding *RuleCheckFinding) error {
	var severity string
	if err := row.Scan(&finding.ID, &finding.EvaluationResultID, &finding.Category, &severity, &finding.Message, &finding.EvidenceRef, &finding.CreatedAt); err != nil {
		return err
	}
	finding.Severity = FindingSeverity(severity)
	return nil
}

func scanMetricScore(row pgx.Row, score *MetricScore) error {
	var source string
	if err := row.Scan(&score.ID, &score.EvaluationResultID, &score.MetricID, &score.MetricCode, &source, &score.SuggestedScore,
		&score.MaxScore, &score.ConfidenceBPS, &score.Rationale, &score.EvidenceRefs, &score.CreatedAt); err != nil {
		return err
	}
	score.Source = MetricScoreSource(source)
	return nil
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	v := value.Time
	return &v
}

func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return notFoundError("resource not found")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return conflictError("resource already exists")
		case "23503":
			return validationError("referenced resource does not exist")
		case "23514":
			msg := "database constraint failed"
			if pgErr.ConstraintName != "" {
				msg = fmt.Sprintf("database constraint %s failed", pgErr.ConstraintName)
			} else if pgErr.Detail != "" {
				msg = pgErr.Detail
			}
			return validationError(msg)
		case "45000", "P0001":
			return conflictError(pgErr.Message)
		}
	}
	return fmt.Errorf("postgres teaching repository: %w", err)
}
