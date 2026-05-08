package teaching

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kenichiLyon/loong64-b1-go/internal/database"
)

type SQLiteRepository struct {
	db *database.Pool
}

func NewSQLiteRepository(db *database.Pool) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) pool() (sqlitePool, error) {
	if r == nil || r.db == nil || r.db.SQLDB() == nil {
		return nil, unavailableError("sqlite teaching repository is not configured", nil)
	}
	return sqliteDBWrapper{db: r.db.SQLDB()}, nil
}

type sqlitePool interface {
	BeginTx(context.Context, sql.TxOptions) (sqliteTx, error)
	Query(context.Context, string, ...any) (sqliteRows, error)
	QueryRow(context.Context, string, ...any) sqliteScanner
	Exec(context.Context, string, ...any) (sql.Result, error)
}

type sqliteTx interface {
	Query(context.Context, string, ...any) (sqliteRows, error)
	QueryRow(context.Context, string, ...any) sqliteScanner
	Exec(context.Context, string, ...any) (sql.Result, error)
	Commit(context.Context) error
	Rollback(context.Context) error
}

type sqliteRows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close() error
}

type sqliteScanner interface {
	Scan(...any) error
}

type sqliteDBWrapper struct {
	db *sql.DB
}

type sqliteTxWrapper struct {
	tx *sql.Tx
}

type sqliteRowsWrapper struct {
	rows *sql.Rows
}

type sqliteErrorRow struct {
	err error
}

var sqlitePlaceholderPattern = regexp.MustCompile(`\$(\d+)`)

func sqliteSQL(query string) string {
	replaced := query
	replaced = strings.ReplaceAll(replaced, " FOR UPDATE", "")
	replaced = strings.ReplaceAll(replaced, " now()", " CURRENT_TIMESTAMP")
	return replaced
}

func sqliteBind(query string, args []any) (string, []any, error) {
	bound := make([]any, 0, len(args))
	var bindErr error
	replaced := sqlitePlaceholderPattern.ReplaceAllStringFunc(query, func(token string) string {
		if bindErr != nil {
			return token
		}
		index, err := strconv.Atoi(token[1:])
		if err != nil || index <= 0 || index > len(args) {
			bindErr = fmt.Errorf("invalid sqlite placeholder %s", token)
			return token
		}
		bound = append(bound, args[index-1])
		return "?"
	})
	return sqliteSQL(replaced), bound, bindErr
}

func (w sqliteDBWrapper) BeginTx(ctx context.Context, _ sql.TxOptions) (sqliteTx, error) {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return sqliteTxWrapper{tx: tx}, nil
}

func (w sqliteDBWrapper) Query(ctx context.Context, query string, args ...any) (sqliteRows, error) {
	rewritten, bound, err := sqliteBind(query, args)
	if err != nil {
		return nil, err
	}
	rows, err := w.db.QueryContext(ctx, rewritten, bound...)
	if err != nil {
		return nil, err
	}
	return sqliteRowsWrapper{rows: rows}, nil
}

func (w sqliteDBWrapper) QueryRow(ctx context.Context, query string, args ...any) sqliteScanner {
	rewritten, bound, err := sqliteBind(query, args)
	if err != nil {
		return sqliteErrorRow{err: err}
	}
	return w.db.QueryRowContext(ctx, rewritten, bound...)
}

func (w sqliteDBWrapper) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	rewritten, bound, err := sqliteBind(query, args)
	if err != nil {
		return nil, err
	}
	return w.db.ExecContext(ctx, rewritten, bound...)
}

func (w sqliteTxWrapper) Query(ctx context.Context, query string, args ...any) (sqliteRows, error) {
	rewritten, bound, err := sqliteBind(query, args)
	if err != nil {
		return nil, err
	}
	rows, err := w.tx.QueryContext(ctx, rewritten, bound...)
	if err != nil {
		return nil, err
	}
	return sqliteRowsWrapper{rows: rows}, nil
}

func (w sqliteTxWrapper) QueryRow(ctx context.Context, query string, args ...any) sqliteScanner {
	rewritten, bound, err := sqliteBind(query, args)
	if err != nil {
		return sqliteErrorRow{err: err}
	}
	return w.tx.QueryRowContext(ctx, rewritten, bound...)
}

func (w sqliteTxWrapper) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	rewritten, bound, err := sqliteBind(query, args)
	if err != nil {
		return nil, err
	}
	return w.tx.ExecContext(ctx, rewritten, bound...)
}

func (w sqliteTxWrapper) Commit(_ context.Context) error {
	return w.tx.Commit()
}

func (w sqliteTxWrapper) Rollback(_ context.Context) error {
	return w.tx.Rollback()
}

func (w sqliteRowsWrapper) Next() bool {
	return w.rows.Next()
}

func (w sqliteRowsWrapper) Scan(dest ...any) error {
	return w.rows.Scan(dest...)
}

func (w sqliteRowsWrapper) Err() error {
	return w.rows.Err()
}

func (w sqliteRowsWrapper) Close() error {
	return w.rows.Close()
}

func (w sqliteErrorRow) Scan(...any) error {
	return w.err
}

func (r *SQLiteRepository) CreateUser(ctx context.Context, user User, roles []Role, audit AuditEntry) (User, error) {
	pool, err := r.pool()
	if err != nil {
		return User{}, err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return User{}, err
	}
	defer sqliteRollback(ctx, tx)
	if err := tx.QueryRow(ctx, `
INSERT INTO users (id, username, display_name, email, student_no, employee_no, password_hash, status)
VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7, $8)
RETURNING id, username, display_name, COALESCE(email, ''), COALESCE(student_no, ''), COALESCE(employee_no, ''), status, created_at, updated_at`,
		user.ID, user.Username, user.DisplayName, user.Email, user.StudentNo, user.EmployeeNo, user.PasswordHash, user.Status,
	).Scan(&user.ID, &user.Username, &user.DisplayName, &user.Email, &user.StudentNo, &user.EmployeeNo, &user.Status, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return User{}, sqliteMapDBError(err)
	}
	for _, role := range roles {
		if _, err := tx.Exec(ctx, `INSERT INTO user_roles (user_id, role) VALUES ($1, $2)`, user.ID, role); err != nil {
			return User{}, sqliteMapDBError(err)
		}
	}
	user.Roles = roles
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return user, nil
}

func (r *SQLiteRepository) CountUsers(ctx context.Context) (int, error) {
	pool, err := r.pool()
	if err != nil {
		return 0, err
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return 0, sqliteMapDBError(err)
	}
	return count, nil
}

func (r *SQLiteRepository) ListUsers(ctx context.Context, limit int) ([]User, error) {
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
		return nil, sqliteMapDBError(err)
	}
	defer func() { _ = rows.Close() }()
	byID := make(map[string]*User)
	order := make([]string, 0)
	for rows.Next() {
		var user User
		var role sql.NullString
		if err := rows.Scan(&user.ID, &user.Username, &user.DisplayName, &user.Email, &user.StudentNo, &user.EmployeeNo, &user.Status, &user.CreatedAt, &user.UpdatedAt, &role); err != nil {
			return nil, sqliteMapDBError(err)
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
		return nil, sqliteMapDBError(err)
	}
	users := make([]User, 0, len(order))
	for _, id := range order {
		users = append(users, *byID[id])
	}
	return users, nil
}

func (r *SQLiteRepository) SetUserRoles(ctx context.Context, userID string, roles []Role, audit AuditEntry) error {
	pool, err := r.pool()
	if err != nil {
		return err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return err
	}
	defer sqliteRollback(ctx, tx)
	var exists int
	if err := tx.QueryRow(ctx, `SELECT 1 FROM users WHERE id = $1`, userID).Scan(&exists); err != nil {
		return sqliteMapDBError(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1`, userID); err != nil {
		return sqliteMapDBError(err)
	}
	for _, role := range roles {
		if _, err := tx.Exec(ctx, `INSERT INTO user_roles (user_id, role) VALUES ($1, $2)`, userID, role); err != nil {
			return sqliteMapDBError(err)
		}
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *SQLiteRepository) SetUserPassword(ctx context.Context, userID string, passwordHash string, audit AuditEntry) error {
	pool, err := r.pool()
	if err != nil {
		return err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return err
	}
	defer sqliteRollback(ctx, tx)
	if _, err := tx.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`, userID, passwordHash); err != nil {
		return sqliteMapDBError(err)
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *SQLiteRepository) UserHasRole(ctx context.Context, userID string, role Role) (bool, error) {
	pool, err := r.pool()
	if err != nil {
		return false, err
	}
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1 AND role = $2)`, userID, role).Scan(&exists); err != nil {
		return false, sqliteMapDBError(err)
	}
	return exists, nil
}

func (r *SQLiteRepository) CreateClass(ctx context.Context, class Class, audit AuditEntry) (Class, error) {
	pool, err := r.pool()
	if err != nil {
		return Class{}, err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return Class{}, err
	}
	defer sqliteRollback(ctx, tx)
	if err := tx.QueryRow(ctx, `
INSERT INTO classes (id, code, name, grade_year, major, status)
VALUES ($1, $2, $3, NULLIF($4, 0), NULLIF($5, ''), $6)
RETURNING id, code, name, COALESCE(grade_year, 0), COALESCE(major, ''), status, created_at, updated_at`,
		class.ID, class.Code, class.Name, class.GradeYear, class.Major, class.Status,
	).Scan(&class.ID, &class.Code, &class.Name, &class.GradeYear, &class.Major, &class.Status, &class.CreatedAt, &class.UpdatedAt); err != nil {
		return Class{}, sqliteMapDBError(err)
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return Class{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Class{}, err
	}
	return class, nil
}

func (r *SQLiteRepository) CreateCourse(ctx context.Context, course Course, audit AuditEntry) (Course, error) {
	pool, err := r.pool()
	if err != nil {
		return Course{}, err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return Course{}, err
	}
	defer sqliteRollback(ctx, tx)
	if err := tx.QueryRow(ctx, `
INSERT INTO courses (id, code, name, term, status, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, code, name, term, status, created_by, created_at, updated_at`,
		course.ID, course.Code, course.Name, course.Term, course.Status, course.CreatedBy,
	).Scan(&course.ID, &course.Code, &course.Name, &course.Term, &course.Status, &course.CreatedBy, &course.CreatedAt, &course.UpdatedAt); err != nil {
		return Course{}, sqliteMapDBError(err)
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return Course{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Course{}, err
	}
	return course, nil
}

func (r *SQLiteRepository) AddCourseClass(ctx context.Context, courseID, classID string, audit AuditEntry) error {
	pool, err := r.pool()
	if err != nil {
		return err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return err
	}
	defer sqliteRollback(ctx, tx)
	if _, err := tx.Exec(ctx, `INSERT INTO course_classes (course_id, class_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, courseID, classID); err != nil {
		return sqliteMapDBError(err)
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *SQLiteRepository) TeacherCanEditCourse(ctx context.Context, courseID, teacherID string) (bool, error) {
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
		return false, sqliteMapDBError(err)
	}
	return allowed, nil
}

func (r *SQLiteRepository) AssignTeacher(ctx context.Context, assignment CourseTeacher, audit AuditEntry) error {
	pool, err := r.pool()
	if err != nil {
		return err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return err
	}
	defer sqliteRollback(ctx, tx)
	if _, err := tx.Exec(ctx, `
INSERT INTO course_teachers (course_id, teacher_id, permission)
VALUES ($1, $2, $3)
ON CONFLICT (course_id, teacher_id) DO UPDATE SET permission = EXCLUDED.permission`, assignment.CourseID, assignment.TeacherID, assignment.Permission); err != nil {
		return sqliteMapDBError(err)
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *SQLiteRepository) EnrollStudent(ctx context.Context, enrollment Enrollment, audit AuditEntry) error {
	pool, err := r.pool()
	if err != nil {
		return err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return err
	}
	defer sqliteRollback(ctx, tx)
	var classID any
	if enrollment.ClassID != "" {
		classID = enrollment.ClassID
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO enrollments (course_id, class_id, student_id, status)
VALUES ($1, $2, $3, $4)
ON CONFLICT (course_id, student_id) DO UPDATE SET class_id = EXCLUDED.class_id, status = EXCLUDED.status`,
		enrollment.CourseID, classID, enrollment.StudentID, enrollment.Status); err != nil {
		return sqliteMapDBError(err)
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *SQLiteRepository) CreateRubricTemplate(ctx context.Context, template RubricTemplate, audit AuditEntry) (RubricTemplate, error) {
	pool, err := r.pool()
	if err != nil {
		return RubricTemplate{}, err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return RubricTemplate{}, err
	}
	defer sqliteRollback(ctx, tx)
	if err := tx.QueryRow(ctx, `
INSERT INTO rubric_templates (id, name, description, owner_id, scope, status)
VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6)
RETURNING id, name, COALESCE(description, ''), owner_id, scope, status, created_at, updated_at`,
		template.ID, template.Name, template.Description, template.OwnerID, template.Scope, template.Status,
	).Scan(&template.ID, &template.Name, &template.Description, &template.OwnerID, &template.Scope, &template.Status, &template.CreatedAt, &template.UpdatedAt); err != nil {
		return RubricTemplate{}, sqliteMapDBError(err)
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return RubricTemplate{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RubricTemplate{}, err
	}
	return template, nil
}

func (r *SQLiteRepository) RubricTemplateOwner(ctx context.Context, templateID string) (string, error) {
	pool, err := r.pool()
	if err != nil {
		return "", err
	}
	var ownerID string
	if err := pool.QueryRow(ctx, `SELECT owner_id FROM rubric_templates WHERE id = $1`, templateID).Scan(&ownerID); err != nil {
		return "", sqliteMapDBError(err)
	}
	return ownerID, nil
}

func (r *SQLiteRepository) CreateRubricVersion(ctx context.Context, version RubricTemplateVersion, metrics []Metric, audit AuditEntry) (RubricTemplateVersion, []Metric, error) {
	pool, err := r.pool()
	if err != nil {
		return RubricTemplateVersion{}, nil, err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return RubricTemplateVersion{}, nil, err
	}
	defer sqliteRollback(ctx, tx)
	var templateID string
	if err := tx.QueryRow(ctx, `SELECT id FROM rubric_templates WHERE id = $1 FOR UPDATE`, version.TemplateID).Scan(&templateID); err != nil {
		return RubricTemplateVersion{}, nil, sqliteMapDBError(err)
	}
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(version_no), 0) + 1 FROM rubric_template_versions WHERE template_id = $1`, version.TemplateID).Scan(&version.VersionNo); err != nil {
		return RubricTemplateVersion{}, nil, sqliteMapDBError(err)
	}
	if err := sqliteScanVersion(tx.QueryRow(ctx, `
INSERT INTO rubric_template_versions (id, template_id, version_no, status, weight_mode, total_weight_bps, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, template_id, version_no, status, weight_mode, total_weight_bps, published_at, created_by, created_at`,
		version.ID, version.TemplateID, version.VersionNo, version.Status, version.WeightMode, version.TotalWeightBPS, version.CreatedBy), &version); err != nil {
		return RubricTemplateVersion{}, nil, sqliteMapDBError(err)
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
			return RubricTemplateVersion{}, nil, sqliteMapDBError(err)
		}
		createdMetrics = append(createdMetrics, metric)
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return RubricTemplateVersion{}, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RubricTemplateVersion{}, nil, err
	}
	return version, createdMetrics, nil
}

func (r *SQLiteRepository) RubricVersionOwner(ctx context.Context, versionID string) (string, error) {
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
		return "", sqliteMapDBError(err)
	}
	return ownerID, nil
}

func (r *SQLiteRepository) PublishRubricVersion(ctx context.Context, versionID string, audit AuditEntry) (RubricTemplateVersion, error) {
	pool, err := r.pool()
	if err != nil {
		return RubricTemplateVersion{}, err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return RubricTemplateVersion{}, err
	}
	defer sqliteRollback(ctx, tx)
	var version RubricTemplateVersion
	if err := sqliteScanVersion(tx.QueryRow(ctx, `
SELECT id, template_id, version_no, status, weight_mode, total_weight_bps, published_at, created_by, created_at
FROM rubric_template_versions WHERE id = $1 FOR UPDATE`, versionID), &version); err != nil {
		return RubricTemplateVersion{}, sqliteMapDBError(err)
	}
	if version.Status != "draft" {
		return RubricTemplateVersion{}, conflictError("only draft rubric versions can be published")
	}
	metricInputs, err := sqliteLoadMetricInputs(ctx, tx, versionID)
	if err != nil {
		return RubricTemplateVersion{}, err
	}
	if err := ValidateMetrics(version.WeightMode, metricInputs); err != nil {
		return RubricTemplateVersion{}, err
	}
	if err := sqliteScanVersion(tx.QueryRow(ctx, `
UPDATE rubric_template_versions
SET status = 'published', published_at = now()
WHERE id = $1 AND status = 'draft'
RETURNING id, template_id, version_no, status, weight_mode, total_weight_bps, published_at, created_by, created_at`, versionID), &version); err != nil {
		return RubricTemplateVersion{}, sqliteMapDBError(err)
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return RubricTemplateVersion{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RubricTemplateVersion{}, err
	}
	return version, nil
}

func (r *SQLiteRepository) RubricVersionStatus(ctx context.Context, versionID string) (string, error) {
	pool, err := r.pool()
	if err != nil {
		return "", err
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM rubric_template_versions WHERE id = $1`, versionID).Scan(&status); err != nil {
		return "", sqliteMapDBError(err)
	}
	return status, nil
}

func (r *SQLiteRepository) CreateExperiment(ctx context.Context, experiment Experiment, audit AuditEntry) (Experiment, error) {
	pool, err := r.pool()
	if err != nil {
		return Experiment{}, err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return Experiment{}, err
	}
	defer sqliteRollback(ctx, tx)
	if err := sqliteScanExperiment(tx.QueryRow(ctx, `
INSERT INTO experiments (id, course_id, title, description, submission_spec, rubric_version_id, status, start_at, due_at, created_by)
VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9, $10)
RETURNING id, course_id, title, COALESCE(description, ''), submission_spec, rubric_version_id, status, start_at, due_at, published_at, created_by, created_at, updated_at`,
		experiment.ID, experiment.CourseID, experiment.Title, experiment.Description, experiment.SubmissionSpec, experiment.RubricVersionID, experiment.Status, experiment.StartAt, experiment.DueAt, experiment.CreatedBy), &experiment); err != nil {
		return Experiment{}, sqliteMapDBError(err)
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return Experiment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Experiment{}, err
	}
	return experiment, nil
}

func (r *SQLiteRepository) ExperimentCourseID(ctx context.Context, experimentID string) (string, error) {
	pool, err := r.pool()
	if err != nil {
		return "", err
	}
	var courseID string
	if err := pool.QueryRow(ctx, `SELECT course_id FROM experiments WHERE id = $1`, experimentID).Scan(&courseID); err != nil {
		return "", sqliteMapDBError(err)
	}
	return courseID, nil
}

func (r *SQLiteRepository) ListExperimentsForCourse(ctx context.Context, courseID string, limit int) ([]Experiment, error) {
	pool, err := r.pool()
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
SELECT id, course_id, title, COALESCE(description, ''), submission_spec, rubric_version_id,
       status, start_at, due_at, published_at, created_by, created_at, updated_at
FROM experiments
WHERE course_id = $1
ORDER BY created_at DESC, id
LIMIT $2`, courseID, limit)
	if err != nil {
		return nil, sqliteMapDBError(err)
	}
	defer func() { _ = rows.Close() }()
	experiments := make([]Experiment, 0)
	for rows.Next() {
		var experiment Experiment
		if err := sqliteScanExperiment(rows, &experiment); err != nil {
			return nil, sqliteMapDBError(err)
		}
		experiments = append(experiments, experiment)
	}
	if err := rows.Err(); err != nil {
		return nil, sqliteMapDBError(err)
	}
	return experiments, nil
}

func (r *SQLiteRepository) PublishExperiment(ctx context.Context, experimentID string, audit AuditEntry) (Experiment, error) {
	pool, err := r.pool()
	if err != nil {
		return Experiment{}, err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return Experiment{}, err
	}
	defer sqliteRollback(ctx, tx)
	var experiment Experiment
	if err := sqliteScanExperiment(tx.QueryRow(ctx, `
UPDATE experiments
SET status = 'published', published_at = now(), updated_at = now()
WHERE id = $1 AND status = 'draft'
RETURNING id, course_id, title, COALESCE(description, ''), submission_spec, rubric_version_id, status, start_at, due_at, published_at, created_by, created_at, updated_at`, experimentID), &experiment); err != nil {
		return Experiment{}, sqliteMapDBError(err)
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return Experiment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Experiment{}, err
	}
	return experiment, nil
}

func (r *SQLiteRepository) ExperimentSubmissionAccess(ctx context.Context, experimentID, studentID string) (ExperimentSubmissionAccess, error) {
	pool, err := r.pool()
	if err != nil {
		return ExperimentSubmissionAccess{}, err
	}
	var access ExperimentSubmissionAccess
	var dueAt sqliteNullableTime
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
		return ExperimentSubmissionAccess{}, sqliteMapDBError(err)
	}
	access.DueAt = sqliteNullableTimePtr(dueAt)
	return access, nil
}

func (r *SQLiteRepository) CreateSubmission(ctx context.Context, submission Submission, audit AuditEntry) (Submission, error) {
	pool, err := r.pool()
	if err != nil {
		return Submission{}, err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return Submission{}, err
	}
	defer sqliteRollback(ctx, tx)
	if err := sqliteScanSubmission(tx.QueryRow(ctx, `
INSERT INTO submissions (id, experiment_id, student_id, status, attempt_no)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, experiment_id, student_id, status, attempt_no, submitted_at, created_at, updated_at`,
		submission.ID, submission.ExperimentID, submission.StudentID, submission.Status, submission.AttemptNo), &submission); err != nil {
		return Submission{}, sqliteMapDBError(err)
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return Submission{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Submission{}, err
	}
	return submission, nil
}

func (r *SQLiteRepository) StudentOwnsSubmission(ctx context.Context, submissionID, studentID string) (bool, error) {
	pool, err := r.pool()
	if err != nil {
		return false, err
	}
	var owns bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM submissions WHERE id = $1 AND student_id = $2)`, submissionID, studentID).Scan(&owns); err != nil {
		return false, sqliteMapDBError(err)
	}
	return owns, nil
}

func (r *SQLiteRepository) SubmissionCourseID(ctx context.Context, submissionID string) (string, error) {
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
		return "", sqliteMapDBError(err)
	}
	return courseID, nil
}

func (r *SQLiteRepository) SubmissionArtifactCount(ctx context.Context, submissionID string) (int, error) {
	pool, err := r.pool()
	if err != nil {
		return 0, err
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM artifacts WHERE submission_id = $1`, submissionID).Scan(&count); err != nil {
		return 0, sqliteMapDBError(err)
	}
	return count, nil
}

func (r *SQLiteRepository) CreateArtifact(ctx context.Context, artifact Artifact, extraction ExtractedContent, job *QueuedJob, audit AuditEntry) (ArtifactWithExtraction, error) {
	pool, err := r.pool()
	if err != nil {
		return ArtifactWithExtraction{}, err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return ArtifactWithExtraction{}, err
	}
	defer sqliteRollback(ctx, tx)
	if err := sqliteScanArtifact(tx.QueryRow(ctx, `
INSERT INTO artifacts (id, submission_id, artifact_kind, original_name, content_type, byte_size, sha256_hex, storage_key, source_url, status, metadata, created_by)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), $10, $11, $12)
RETURNING id, submission_id, artifact_kind, original_name, COALESCE(content_type, ''), byte_size, COALESCE(sha256_hex, ''), COALESCE(storage_key, ''), COALESCE(source_url, ''), status, metadata, created_by, created_at`,
		artifact.ID, artifact.SubmissionID, artifact.Kind, artifact.OriginalName, artifact.ContentType, artifact.ByteSize, artifact.SHA256Hex, artifact.StorageKey, artifact.SourceURL, artifact.Status, defaultJSON(artifact.Metadata), artifact.CreatedBy), &artifact); err != nil {
		return ArtifactWithExtraction{}, sqliteMapDBError(err)
	}
	if err := sqliteScanExtraction(tx.QueryRow(ctx, `
INSERT INTO extracted_contents (id, artifact_id, status, text_excerpt, metadata, error)
VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))
RETURNING id, artifact_id, status, text_excerpt, metadata, COALESCE(error, ''), created_at, updated_at`,
		extraction.ID, extraction.ArtifactID, extraction.Status, extraction.TextExcerpt, defaultJSON(extraction.Metadata), extraction.Error), &extraction); err != nil {
		return ArtifactWithExtraction{}, sqliteMapDBError(err)
	}
	jobID := ""
	if job != nil {
		payload := defaultJSON(job.Payload)
		if _, err := tx.Exec(ctx, `
INSERT INTO jobs (id, job_type, status, payload)
VALUES ($1, $2, 'queued', $3)`, job.ID, job.Type, payload); err != nil {
			return ArtifactWithExtraction{}, sqliteMapDBError(err)
		}
		jobID = job.ID
	}
	nextStatus := "parsed"
	if extraction.Status == "queued" {
		nextStatus = "parsing"
	}
	if _, err := tx.Exec(ctx, `UPDATE submissions SET status = $2, updated_at = now() WHERE id = $1`, artifact.SubmissionID, nextStatus); err != nil {
		return ArtifactWithExtraction{}, sqliteMapDBError(err)
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return ArtifactWithExtraction{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ArtifactWithExtraction{}, err
	}
	return ArtifactWithExtraction{Artifact: artifact, Extraction: extraction, JobID: jobID}, nil
}

func (r *SQLiteRepository) ListSubmissionsForExperiment(ctx context.Context, experimentID string, limit int) ([]Submission, error) {
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
		return nil, sqliteMapDBError(err)
	}
	defer func() { _ = rows.Close() }()
	submissions := make([]Submission, 0)
	for rows.Next() {
		var submission Submission
		if err := sqliteScanSubmission(rows, &submission); err != nil {
			return nil, sqliteMapDBError(err)
		}
		submissions = append(submissions, submission)
	}
	if err := rows.Err(); err != nil {
		return nil, sqliteMapDBError(err)
	}
	return submissions, nil
}

func (r *SQLiteRepository) GetSubmissionDetail(ctx context.Context, submissionID string) (SubmissionDetail, error) {
	pool, err := r.pool()
	if err != nil {
		return SubmissionDetail{}, err
	}
	var detail SubmissionDetail
	if err := sqliteScanSubmission(pool.QueryRow(ctx, `
SELECT id, experiment_id, student_id, status, attempt_no, submitted_at, created_at, updated_at
FROM submissions
WHERE id = $1`, submissionID), &detail.Submission); err != nil {
		return SubmissionDetail{}, sqliteMapDBError(err)
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
		return SubmissionDetail{}, sqliteMapDBError(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var item ArtifactWithExtraction
		if err := sqliteScanArtifactAndExtraction(rows, &item.Artifact, &item.Extraction); err != nil {
			return SubmissionDetail{}, sqliteMapDBError(err)
		}
		detail.Artifacts = append(detail.Artifacts, item)
	}
	if err := rows.Err(); err != nil {
		return SubmissionDetail{}, sqliteMapDBError(err)
	}
	return detail, nil
}

func (r *SQLiteRepository) GetEvaluationContext(ctx context.Context, submissionID string) (EvaluationContext, error) {
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
	if err := sqliteScanExperiment(pool.QueryRow(ctx, `
SELECT experiments.id, experiments.course_id, experiments.title, COALESCE(experiments.description, ''),
       experiments.submission_spec, experiments.rubric_version_id, experiments.status,
       experiments.start_at, experiments.due_at, experiments.published_at, experiments.created_by,
       experiments.created_at, experiments.updated_at
FROM experiments
WHERE experiments.id = $1`, detail.Submission.ExperimentID), &evalCtx.Experiment); err != nil {
		return EvaluationContext{}, sqliteMapDBError(err)
	}
	rows, err := pool.Query(ctx, `
SELECT id, version_id, code, name, COALESCE(description, ''), weight_bps, max_score, sort_order, required_evidence
FROM rubric_metrics
WHERE version_id = $1
ORDER BY sort_order`, evalCtx.Experiment.RubricVersionID)
	if err != nil {
		return EvaluationContext{}, sqliteMapDBError(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var metric Metric
		if err := sqliteScanMetric(rows, &metric); err != nil {
			return EvaluationContext{}, sqliteMapDBError(err)
		}
		evalCtx.Metrics = append(evalCtx.Metrics, metric)
	}
	if err := rows.Err(); err != nil {
		return EvaluationContext{}, sqliteMapDBError(err)
	}
	return evalCtx, nil
}

func (r *SQLiteRepository) CreateInitialEvaluation(ctx context.Context, result EvaluationResult, findings []RuleCheckFinding, scores []MetricScore, llmLog *LLMCallLog, audit AuditEntry) (EvaluationResultDetail, error) {
	pool, err := r.pool()
	if err != nil {
		return EvaluationResultDetail{}, err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return EvaluationResultDetail{}, err
	}
	defer sqliteRollback(ctx, tx)
	if err := sqliteScanEvaluationResult(tx.QueryRow(ctx, `
INSERT INTO evaluation_results (id, submission_id, experiment_id, rubric_version_id, status, rule_status, llm_status,
                                prompt_version, evidence_snapshot, rule_summary, llm_summary, needs_teacher_review, error, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING id, submission_id, experiment_id, rubric_version_id, status, rule_status, llm_status, prompt_version,
          evidence_snapshot, rule_summary, llm_summary, needs_teacher_review, error, created_by, created_at, updated_at`,
		result.ID, result.SubmissionID, result.ExperimentID, result.RubricVersionID, result.Status, result.RuleStatus, result.LLMStatus,
		result.PromptVersion, defaultJSON(result.EvidenceSnapshot), defaultJSON(result.RuleSummary), result.LLMSummary, result.NeedsTeacherReview,
		result.Error, result.CreatedBy), &result); err != nil {
		return EvaluationResultDetail{}, sqliteMapDBError(err)
	}
	createdFindings := make([]RuleCheckFinding, 0, len(findings))
	for _, finding := range findings {
		finding.EvaluationResultID = result.ID
		if err := sqliteScanRuleFinding(tx.QueryRow(ctx, `
INSERT INTO rule_check_findings (id, evaluation_result_id, category, severity, message, evidence_ref)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, evaluation_result_id, category, severity, message, evidence_ref, created_at`,
			finding.ID, finding.EvaluationResultID, finding.Category, finding.Severity, finding.Message, finding.EvidenceRef), &finding); err != nil {
			return EvaluationResultDetail{}, sqliteMapDBError(err)
		}
		createdFindings = append(createdFindings, finding)
	}
	createdScores := make([]MetricScore, 0, len(scores))
	for _, score := range scores {
		score.EvaluationResultID = result.ID
		if err := sqliteScanMetricScore(tx.QueryRow(ctx, `
INSERT INTO metric_scores (id, evaluation_result_id, metric_id, metric_code, source, suggested_score, max_score,
                           confidence_bps, rationale, evidence_refs)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, evaluation_result_id, metric_id, metric_code, source, suggested_score, max_score, confidence_bps,
          rationale, evidence_refs, created_at`,
			score.ID, score.EvaluationResultID, score.MetricID, score.MetricCode, score.Source, score.SuggestedScore, score.MaxScore,
			score.ConfidenceBPS, score.Rationale, defaultJSON(score.EvidenceRefs)), &score); err != nil {
			return EvaluationResultDetail{}, sqliteMapDBError(err)
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
			return EvaluationResultDetail{}, sqliteMapDBError(err)
		}
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return EvaluationResultDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return EvaluationResultDetail{}, err
	}
	return EvaluationResultDetail{Result: result, Findings: createdFindings, Scores: createdScores}, nil
}

func (r *SQLiteRepository) GetLatestEvaluation(ctx context.Context, submissionID string) (EvaluationResultDetail, error) {
	pool, err := r.pool()
	if err != nil {
		return EvaluationResultDetail{}, err
	}
	var detail EvaluationResultDetail
	if err := sqliteScanEvaluationResult(pool.QueryRow(ctx, `
SELECT id, submission_id, experiment_id, rubric_version_id, status, rule_status, llm_status, prompt_version,
       evidence_snapshot, rule_summary, llm_summary, needs_teacher_review, error, created_by, created_at, updated_at
FROM evaluation_results
WHERE submission_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1`, submissionID), &detail.Result); err != nil {
		return EvaluationResultDetail{}, sqliteMapDBError(err)
	}
	findingRows, err := pool.Query(ctx, `
SELECT id, evaluation_result_id, category, severity, message, evidence_ref, created_at
FROM rule_check_findings
WHERE evaluation_result_id = $1
ORDER BY created_at, id`, detail.Result.ID)
	if err != nil {
		return EvaluationResultDetail{}, sqliteMapDBError(err)
	}
	defer func() { _ = findingRows.Close() }()
	for findingRows.Next() {
		var finding RuleCheckFinding
		if err := sqliteScanRuleFinding(findingRows, &finding); err != nil {
			return EvaluationResultDetail{}, sqliteMapDBError(err)
		}
		detail.Findings = append(detail.Findings, finding)
	}
	if err := findingRows.Err(); err != nil {
		return EvaluationResultDetail{}, sqliteMapDBError(err)
	}
	scoreRows, err := pool.Query(ctx, `
SELECT id, evaluation_result_id, metric_id, metric_code, source, suggested_score, max_score, confidence_bps,
       rationale, evidence_refs, created_at
FROM metric_scores
WHERE evaluation_result_id = $1
ORDER BY source, metric_code`, detail.Result.ID)
	if err != nil {
		return EvaluationResultDetail{}, sqliteMapDBError(err)
	}
	defer func() { _ = scoreRows.Close() }()
	for scoreRows.Next() {
		var score MetricScore
		if err := sqliteScanMetricScore(scoreRows, &score); err != nil {
			return EvaluationResultDetail{}, sqliteMapDBError(err)
		}
		detail.Scores = append(detail.Scores, score)
	}
	if err := scoreRows.Err(); err != nil {
		return EvaluationResultDetail{}, sqliteMapDBError(err)
	}
	return detail, nil
}

func (r *SQLiteRepository) EvaluationResultSubmissionID(ctx context.Context, evaluationResultID string) (string, error) {
	pool, err := r.pool()
	if err != nil {
		return "", err
	}
	var submissionID string
	if err := pool.QueryRow(ctx, `SELECT submission_id FROM evaluation_results WHERE id = $1`, evaluationResultID).Scan(&submissionID); err != nil {
		return "", sqliteMapDBError(err)
	}
	return submissionID, nil
}

func (r *SQLiteRepository) UpsertTeacherReview(ctx context.Context, review TeacherReview, scores []TeacherMetricScore, audit AuditEntry) (TeacherReviewDetail, error) {
	pool, err := r.pool()
	if err != nil {
		return TeacherReviewDetail{}, err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return TeacherReviewDetail{}, err
	}
	defer sqliteRollback(ctx, tx)
	var existingID sql.NullString
	var existingStatus sql.NullString
	if err := tx.QueryRow(ctx, `SELECT id, status FROM teacher_reviews WHERE submission_id = $1 FOR UPDATE`, review.SubmissionID).Scan(&existingID, &existingStatus); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return TeacherReviewDetail{}, sqliteMapDBError(err)
	}
	if existingStatus.Valid && existingStatus.String == string(TeacherReviewStatusPublished) {
		return TeacherReviewDetail{}, conflictError("published teacher review cannot be modified")
	}
	if existingID.Valid {
		review.ID = existingID.String
		for i := range scores {
			scores[i].TeacherReviewID = review.ID
		}
		if _, err := tx.Exec(ctx, `DELETE FROM teacher_metric_scores WHERE teacher_review_id = $1`, review.ID); err != nil {
			return TeacherReviewDetail{}, sqliteMapDBError(err)
		}
		if err := sqliteScanTeacherReview(tx.QueryRow(ctx, `
UPDATE teacher_reviews
SET evaluation_result_id = NULLIF($2, ''), experiment_id = $3, rubric_version_id = $4, total_score_bps = $5,
    teacher_comment = $6, updated_by = $7, updated_at = now()
WHERE id = $1 AND status = 'draft'
RETURNING id, submission_id, COALESCE(evaluation_result_id, ''), experiment_id, rubric_version_id, status,
          total_score_bps, teacher_comment, created_by, updated_by, published_by, published_at, created_at, updated_at`,
			review.ID, review.EvaluationResultID, review.ExperimentID, review.RubricVersionID, review.TotalScoreBPS, review.TeacherComment, review.UpdatedBy), &review); err != nil {
			return TeacherReviewDetail{}, sqliteMapDBError(err)
		}
	} else {
		if err := sqliteScanTeacherReview(tx.QueryRow(ctx, `
INSERT INTO teacher_reviews (id, submission_id, evaluation_result_id, experiment_id, rubric_version_id, status,
                             total_score_bps, teacher_comment, created_by, updated_by)
VALUES ($1, $2, NULLIF($3, ''), $4, $5, 'draft', $6, $7, $8, $9)
RETURNING id, submission_id, COALESCE(evaluation_result_id, ''), experiment_id, rubric_version_id, status,
          total_score_bps, teacher_comment, created_by, updated_by, published_by, published_at, created_at, updated_at`,
			review.ID, review.SubmissionID, review.EvaluationResultID, review.ExperimentID, review.RubricVersionID, review.TotalScoreBPS,
			review.TeacherComment, review.CreatedBy, review.UpdatedBy), &review); err != nil {
			return TeacherReviewDetail{}, sqliteMapDBError(err)
		}
	}
	createdScores, err := sqliteInsertTeacherMetricScores(ctx, tx, review.ID, scores)
	if err != nil {
		return TeacherReviewDetail{}, err
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return TeacherReviewDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return TeacherReviewDetail{}, err
	}
	return TeacherReviewDetail{Review: review, Scores: createdScores}, nil
}

func (r *SQLiteRepository) PublishTeacherReview(ctx context.Context, submissionID, actorID string, audit AuditEntry) (TeacherReviewDetail, error) {
	pool, err := r.pool()
	if err != nil {
		return TeacherReviewDetail{}, err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return TeacherReviewDetail{}, err
	}
	defer sqliteRollback(ctx, tx)
	var review TeacherReview
	if err := sqliteScanTeacherReview(tx.QueryRow(ctx, `
UPDATE teacher_reviews
SET status = 'published', published_by = $2, published_at = now(), updated_by = $2, updated_at = now()
WHERE submission_id = $1 AND status = 'draft'
RETURNING id, submission_id, COALESCE(evaluation_result_id, ''), experiment_id, rubric_version_id, status,
          total_score_bps, teacher_comment, created_by, updated_by, published_by, published_at, created_at, updated_at`, submissionID, actorID), &review); err != nil {
		return TeacherReviewDetail{}, sqliteMapDBError(err)
	}
	scores, err := sqliteLoadTeacherMetricScores(ctx, tx, review.ID)
	if err != nil {
		return TeacherReviewDetail{}, err
	}
	if len(scores) == 0 {
		return TeacherReviewDetail{}, conflictError("teacher review has no metric scores")
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return TeacherReviewDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return TeacherReviewDetail{}, err
	}
	return TeacherReviewDetail{Review: review, Scores: scores}, nil
}

func (r *SQLiteRepository) GetTeacherReview(ctx context.Context, submissionID string, publishedOnly bool) (TeacherReviewDetail, error) {
	pool, err := r.pool()
	if err != nil {
		return TeacherReviewDetail{}, err
	}
	query := `
SELECT id, submission_id, COALESCE(evaluation_result_id, ''), experiment_id, rubric_version_id, status,
       total_score_bps, teacher_comment, created_by, updated_by, published_by, published_at, created_at, updated_at
FROM teacher_reviews
WHERE submission_id = $1`
	args := []any{submissionID}
	if publishedOnly {
		query += ` AND status = 'published'`
	}
	query += ` ORDER BY updated_at DESC LIMIT 1`
	var detail TeacherReviewDetail
	if err := sqliteScanTeacherReview(pool.QueryRow(ctx, query, args...), &detail.Review); err != nil {
		return TeacherReviewDetail{}, sqliteMapDBError(err)
	}
	scores, err := sqliteLoadTeacherMetricScores(ctx, pool, detail.Review.ID)
	if err != nil {
		return TeacherReviewDetail{}, err
	}
	detail.Scores = scores
	return detail, nil
}

func (r *SQLiteRepository) CreateReportExport(ctx context.Context, export ReportExport, audit AuditEntry) (ReportExport, error) {
	pool, err := r.pool()
	if err != nil {
		return ReportExport{}, err
	}
	tx, err := pool.BeginTx(ctx, sql.TxOptions{})
	if err != nil {
		return ReportExport{}, err
	}
	defer sqliteRollback(ctx, tx)
	if err := sqliteScanReportExport(tx.QueryRow(ctx, `
INSERT INTO report_exports (id, report_type, scope_type, scope_id, format, status, filter_json, requested_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, report_type, scope_type, scope_id, format, status, storage_key, sha256_hex, byte_size,
          filter_json, error, requested_by, created_at, updated_at, completed_at`,
		export.ID, export.ReportType, export.ScopeType, export.ScopeID, export.Format, export.Status,
		defaultJSON(export.FilterJSON), export.RequestedBy), &export); err != nil {
		return ReportExport{}, sqliteMapDBError(err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO jobs (id, job_type, status, payload)
VALUES ($1, $2, 'queued', $3)`,
		NewID("job"), ReportExportJobType, mustJSON(map[string]any{"report_export_id": export.ID, "report_type": export.ReportType, "scope_type": export.ScopeType, "scope_id": export.ScopeID, "format": export.Format})); err != nil {
		return ReportExport{}, sqliteMapDBError(err)
	}
	if err := sqliteInsertAudit(ctx, tx, audit); err != nil {
		return ReportExport{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReportExport{}, err
	}
	return export, nil
}

func (r *SQLiteRepository) CompleteReportExport(ctx context.Context, export ReportExport) (ReportExport, error) {
	pool, err := r.pool()
	if err != nil {
		return ReportExport{}, err
	}
	if err := sqliteScanReportExport(pool.QueryRow(ctx, `
UPDATE report_exports
SET status = $2, storage_key = $3, sha256_hex = $4, byte_size = $5, error = $6,
    completed_at = now(), updated_at = now()
WHERE id = $1
RETURNING id, report_type, scope_type, scope_id, format, status, storage_key, sha256_hex, byte_size,
          filter_json, error, requested_by, created_at, updated_at, completed_at`,
		export.ID, export.Status, export.StorageKey, export.SHA256Hex, export.ByteSize, export.Error), &export); err != nil {
		return ReportExport{}, sqliteMapDBError(err)
	}
	return export, nil
}

func (r *SQLiteRepository) GetReportExport(ctx context.Context, exportID string) (ReportExport, error) {
	pool, err := r.pool()
	if err != nil {
		return ReportExport{}, err
	}
	var export ReportExport
	if err := sqliteScanReportExport(pool.QueryRow(ctx, `
SELECT id, report_type, scope_type, scope_id, format, status, storage_key, sha256_hex, byte_size,
       filter_json, error, requested_by, created_at, updated_at, completed_at
FROM report_exports
WHERE id = $1`, exportID), &export); err != nil {
		return ReportExport{}, sqliteMapDBError(err)
	}
	return export, nil
}

func sqliteRollback(ctx context.Context, tx sqliteTx) {
	_ = tx.Rollback(ctx)
}

func sqliteInsertTeacherMetricScores(ctx context.Context, tx sqliteTx, reviewID string, scores []TeacherMetricScore) ([]TeacherMetricScore, error) {
	created := make([]TeacherMetricScore, 0, len(scores))
	for _, score := range scores {
		score.TeacherReviewID = reviewID
		if err := sqliteScanTeacherMetricScore(tx.QueryRow(ctx, `
INSERT INTO teacher_metric_scores (id, teacher_review_id, metric_id, metric_code, final_score, max_score, weight_bps,
                                   source, source_metric_score_id, comment, adjustment_reason)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, ''), $10, $11)
RETURNING id, teacher_review_id, metric_id, metric_code, final_score, max_score, weight_bps, source,
          COALESCE(source_metric_score_id, ''), comment, adjustment_reason, created_at, updated_at`,
			score.ID, score.TeacherReviewID, score.MetricID, score.MetricCode, score.FinalScore, score.MaxScore, score.WeightBPS,
			score.Source, score.SourceMetricScoreID, score.Comment, score.AdjustmentReason), &score); err != nil {
			return nil, sqliteMapDBError(err)
		}
		created = append(created, score)
	}
	return created, nil
}

func sqliteLoadTeacherMetricScores(ctx context.Context, queryer interface {
	Query(context.Context, string, ...any) (sqliteRows, error)
}, reviewID string) ([]TeacherMetricScore, error) {
	rows, err := queryer.Query(ctx, `
SELECT id, teacher_review_id, metric_id, metric_code, final_score, max_score, weight_bps, source,
       COALESCE(source_metric_score_id, ''), comment, adjustment_reason, created_at, updated_at
FROM teacher_metric_scores
WHERE teacher_review_id = $1
ORDER BY metric_code`, reviewID)
	if err != nil {
		return nil, sqliteMapDBError(err)
	}
	defer func() { _ = rows.Close() }()
	scores := make([]TeacherMetricScore, 0)
	for rows.Next() {
		var score TeacherMetricScore
		if err := sqliteScanTeacherMetricScore(rows, &score); err != nil {
			return nil, sqliteMapDBError(err)
		}
		scores = append(scores, score)
	}
	if err := rows.Err(); err != nil {
		return nil, sqliteMapDBError(err)
	}
	return scores, nil
}

func sqliteInsertAudit(ctx context.Context, tx sqliteTx, audit AuditEntry) error {
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
	return sqliteMapDBError(err)
}

func sqliteLoadMetricInputs(ctx context.Context, tx sqliteTx, versionID string) ([]MetricInput, error) {
	rows, err := tx.Query(ctx, `
SELECT code, name, COALESCE(description, ''), weight_bps, max_score, sort_order, required_evidence
FROM rubric_metrics
WHERE version_id = $1
ORDER BY sort_order`, versionID)
	if err != nil {
		return nil, sqliteMapDBError(err)
	}
	defer func() { _ = rows.Close() }()
	metrics := make([]MetricInput, 0)
	for rows.Next() {
		var metric MetricInput
		if err := rows.Scan(&metric.Code, &metric.Name, &metric.Description, &metric.WeightBPS, &metric.MaxScore, &metric.SortOrder, &metric.RequiredEvidence); err != nil {
			return nil, sqliteMapDBError(err)
		}
		metrics = append(metrics, metric)
	}
	if err := rows.Err(); err != nil {
		return nil, sqliteMapDBError(err)
	}
	return metrics, nil
}

func sqliteScanVersion(row sqliteScanner, version *RubricTemplateVersion) error {
	var publishedAt sqliteNullableTime
	var mode string
	if err := row.Scan(&version.ID, &version.TemplateID, &version.VersionNo, &version.Status, &mode, &version.TotalWeightBPS, &publishedAt, &version.CreatedBy, &version.CreatedAt); err != nil {
		return err
	}
	version.WeightMode = WeightMode(mode)
	version.PublishedAt = sqliteNullableTimePtr(publishedAt)
	return nil
}

func sqliteScanExperiment(row sqliteScanner, experiment *Experiment) error {
	var startAt, dueAt, publishedAt sqliteNullableTime
	if err := row.Scan(&experiment.ID, &experiment.CourseID, &experiment.Title, &experiment.Description, &experiment.SubmissionSpec, &experiment.RubricVersionID, &experiment.Status, &startAt, &dueAt, &publishedAt, &experiment.CreatedBy, &experiment.CreatedAt, &experiment.UpdatedAt); err != nil {
		return err
	}
	experiment.StartAt = sqliteNullableTimePtr(startAt)
	experiment.DueAt = sqliteNullableTimePtr(dueAt)
	experiment.PublishedAt = sqliteNullableTimePtr(publishedAt)
	return nil
}

func sqliteScanSubmission(row sqliteScanner, submission *Submission) error {
	var submittedAt sqliteNullableTime
	if err := row.Scan(&submission.ID, &submission.ExperimentID, &submission.StudentID, &submission.Status, &submission.AttemptNo, &submittedAt, &submission.CreatedAt, &submission.UpdatedAt); err != nil {
		return err
	}
	submission.SubmittedAt = sqliteNullableTimePtr(submittedAt)
	return nil
}

func sqliteScanArtifact(row sqliteScanner, artifact *Artifact) error {
	var kind string
	if err := row.Scan(&artifact.ID, &artifact.SubmissionID, &kind, &artifact.OriginalName, &artifact.ContentType, &artifact.ByteSize, &artifact.SHA256Hex, &artifact.StorageKey, &artifact.SourceURL, &artifact.Status, &artifact.Metadata, &artifact.CreatedBy, &artifact.CreatedAt); err != nil {
		return err
	}
	artifact.Kind = ArtifactKind(kind)
	return nil
}

func sqliteScanExtraction(row sqliteScanner, extraction *ExtractedContent) error {
	if err := row.Scan(&extraction.ID, &extraction.ArtifactID, &extraction.Status, &extraction.TextExcerpt, &extraction.Metadata, &extraction.Error, &extraction.CreatedAt, &extraction.UpdatedAt); err != nil {
		return err
	}
	return nil
}

func sqliteScanArtifactAndExtraction(row sqliteScanner, artifact *Artifact, extraction *ExtractedContent) error {
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

func sqliteScanMetric(row sqliteScanner, metric *Metric) error {
	return row.Scan(&metric.ID, &metric.VersionID, &metric.Code, &metric.Name, &metric.Description, &metric.WeightBPS, &metric.MaxScore, &metric.SortOrder, &metric.RequiredEvidence)
}

func sqliteScanEvaluationResult(row sqliteScanner, result *EvaluationResult) error {
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

func sqliteScanRuleFinding(row sqliteScanner, finding *RuleCheckFinding) error {
	var severity string
	if err := row.Scan(&finding.ID, &finding.EvaluationResultID, &finding.Category, &severity, &finding.Message, &finding.EvidenceRef, &finding.CreatedAt); err != nil {
		return err
	}
	finding.Severity = FindingSeverity(severity)
	return nil
}

func sqliteScanMetricScore(row sqliteScanner, score *MetricScore) error {
	var source string
	if err := row.Scan(&score.ID, &score.EvaluationResultID, &score.MetricID, &score.MetricCode, &source, &score.SuggestedScore,
		&score.MaxScore, &score.ConfidenceBPS, &score.Rationale, &score.EvidenceRefs, &score.CreatedAt); err != nil {
		return err
	}
	score.Source = MetricScoreSource(source)
	return nil
}

func sqliteScanTeacherReview(row sqliteScanner, review *TeacherReview) error {
	var status string
	var publishedAt sqliteNullableTime
	if err := row.Scan(&review.ID, &review.SubmissionID, &review.EvaluationResultID, &review.ExperimentID, &review.RubricVersionID,
		&status, &review.TotalScoreBPS, &review.TeacherComment, &review.CreatedBy, &review.UpdatedBy,
		&review.PublishedBy, &publishedAt, &review.CreatedAt, &review.UpdatedAt); err != nil {
		return err
	}
	review.Status = TeacherReviewStatus(status)
	review.PublishedAt = sqliteNullableTimePtr(publishedAt)
	return nil
}

func sqliteScanTeacherMetricScore(row sqliteScanner, score *TeacherMetricScore) error {
	return row.Scan(&score.ID, &score.TeacherReviewID, &score.MetricID, &score.MetricCode, &score.FinalScore,
		&score.MaxScore, &score.WeightBPS, &score.Source, &score.SourceMetricScoreID, &score.Comment,
		&score.AdjustmentReason, &score.CreatedAt, &score.UpdatedAt)
}

func sqliteScanReportExport(row sqliteScanner, export *ReportExport) error {
	var reportType, scopeType, format, status string
	var completedAt sqliteNullableTime
	if err := row.Scan(&export.ID, &reportType, &scopeType, &export.ScopeID, &format, &status,
		&export.StorageKey, &export.SHA256Hex, &export.ByteSize, &export.FilterJSON, &export.Error,
		&export.RequestedBy, &export.CreatedAt, &export.UpdatedAt, &completedAt); err != nil {
		return err
	}
	export.ReportType = ReportType(reportType)
	export.ScopeType = ReportScopeType(scopeType)
	export.Format = ReportFormat(format)
	export.Status = ReportExportStatus(status)
	export.CompletedAt = sqliteNullableTimePtr(completedAt)
	return nil
}

func sqliteNullableTimePtr(value sqliteNullableTime) *time.Time {
	if !value.Valid {
		return nil
	}
	v := value.Time
	return &v
}

type sqliteNullableTime struct {
	Time  time.Time
	Valid bool
}

func (t *sqliteNullableTime) Scan(src any) error {
	switch value := src.(type) {
	case nil:
		t.Valid = false
		t.Time = time.Time{}
		return nil
	case time.Time:
		t.Time = value.UTC()
		t.Valid = true
		return nil
	case string:
		return t.parse(value)
	case []byte:
		return t.parse(string(value))
	default:
		return fmt.Errorf("unsupported sqlite time type %T", src)
	}
}

func (t *sqliteNullableTime) parse(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		t.Valid = false
		t.Time = time.Time{}
		return nil
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			t.Time = parsed.UTC()
			t.Valid = true
			return nil
		}
	}
	return fmt.Errorf("parse sqlite time %q", value)
}

func sqliteMapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return notFoundError("resource not found")
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unique constraint failed"):
		return conflictError("resource already exists")
	case strings.Contains(message, "foreign key constraint failed"):
		return validationError("referenced resource does not exist")
	case strings.Contains(message, "check constraint failed"):
		return validationError("database constraint failed")
	case strings.Contains(message, "published teacher reviews are immutable"),
		strings.Contains(message, "published teacher metric scores are immutable"),
		strings.Contains(message, "published rubric versions are immutable"),
		strings.Contains(message, "published rubric metrics are immutable"):
		return conflictError(err.Error())
	}
	return fmt.Errorf("sqlite teaching repository: %w", err)
}
