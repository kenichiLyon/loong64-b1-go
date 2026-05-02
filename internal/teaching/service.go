package teaching

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

type User struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email,omitempty"`
	StudentNo   string    `json:"student_no,omitempty"`
	EmployeeNo  string    `json:"employee_no,omitempty"`
	Status      string    `json:"status"`
	Roles       []Role    `json:"roles"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Class struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	GradeYear int       `json:"grade_year,omitempty"`
	Major     string    `json:"major,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Course struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Term      string    `json:"term"`
	Status    string    `json:"status"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CourseTeacher struct {
	CourseID   string `json:"course_id"`
	TeacherID  string `json:"teacher_id"`
	Permission string `json:"permission"`
}

type Enrollment struct {
	CourseID   string    `json:"course_id"`
	ClassID    string    `json:"class_id,omitempty"`
	StudentID  string    `json:"student_id"`
	Status     string    `json:"status"`
	EnrolledAt time.Time `json:"enrolled_at"`
}

type RubricTemplate struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	OwnerID     string    `json:"owner_id"`
	Scope       string    `json:"scope"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RubricTemplateVersion struct {
	ID             string     `json:"id"`
	TemplateID     string     `json:"template_id"`
	VersionNo      int        `json:"version_no"`
	Status         string     `json:"status"`
	WeightMode     WeightMode `json:"weight_mode"`
	TotalWeightBPS int        `json:"total_weight_bps"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	CreatedBy      string     `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
}

type Metric struct {
	ID               string          `json:"id"`
	VersionID        string          `json:"version_id"`
	Code             string          `json:"code"`
	Name             string          `json:"name"`
	Description      string          `json:"description,omitempty"`
	WeightBPS        int             `json:"weight_bps"`
	MaxScore         int             `json:"max_score"`
	SortOrder        int             `json:"sort_order"`
	RequiredEvidence json.RawMessage `json:"required_evidence"`
}

type Experiment struct {
	ID              string          `json:"id"`
	CourseID        string          `json:"course_id"`
	Title           string          `json:"title"`
	Description     string          `json:"description,omitempty"`
	SubmissionSpec  json.RawMessage `json:"submission_spec"`
	RubricVersionID string          `json:"rubric_version_id"`
	Status          string          `json:"status"`
	StartAt         *time.Time      `json:"start_at,omitempty"`
	DueAt           *time.Time      `json:"due_at,omitempty"`
	PublishedAt     *time.Time      `json:"published_at,omitempty"`
	CreatedBy       string          `json:"created_by"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type MetricInput struct {
	Code             string          `json:"code"`
	Name             string          `json:"name"`
	Description      string          `json:"description,omitempty"`
	WeightBPS        int             `json:"weight_bps"`
	MaxScore         int             `json:"max_score"`
	SortOrder        int             `json:"sort_order"`
	RequiredEvidence json.RawMessage `json:"required_evidence,omitempty"`
}

type AuditEntry struct {
	ActorID    string
	Action     string
	TargetType string
	TargetID   string
	Detail     json.RawMessage
	RequestID  string
}

type Repository interface {
	CreateUser(context.Context, User, []Role, AuditEntry) (User, error)
	ListUsers(context.Context, int) ([]User, error)
	SetUserRoles(context.Context, string, []Role, AuditEntry) error
	UserHasRole(context.Context, string, Role) (bool, error)
	CreateClass(context.Context, Class, AuditEntry) (Class, error)
	CreateCourse(context.Context, Course, AuditEntry) (Course, error)
	AddCourseClass(context.Context, string, string, AuditEntry) error
	TeacherCanEditCourse(context.Context, string, string) (bool, error)
	AssignTeacher(context.Context, CourseTeacher, AuditEntry) error
	EnrollStudent(context.Context, Enrollment, AuditEntry) error
	CreateRubricTemplate(context.Context, RubricTemplate, AuditEntry) (RubricTemplate, error)
	RubricTemplateOwner(context.Context, string) (string, error)
	CreateRubricVersion(context.Context, RubricTemplateVersion, []Metric, AuditEntry) (RubricTemplateVersion, []Metric, error)
	RubricVersionOwner(context.Context, string) (string, error)
	PublishRubricVersion(context.Context, string, AuditEntry) (RubricTemplateVersion, error)
	RubricVersionStatus(context.Context, string) (string, error)
	CreateExperiment(context.Context, Experiment, AuditEntry) (Experiment, error)
	ExperimentCourseID(context.Context, string) (string, error)
	PublishExperiment(context.Context, string, AuditEntry) (Experiment, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

type CreateUserInput struct {
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email,omitempty"`
	StudentNo   string   `json:"student_no,omitempty"`
	EmployeeNo  string   `json:"employee_no,omitempty"`
	Status      string   `json:"status,omitempty"`
	Roles       []string `json:"roles"`
}

func (s *Service) CreateUser(ctx context.Context, actor Actor, input CreateUserInput, audit AuditEntry) (User, error) {
	if err := s.ready(); err != nil {
		return User{}, err
	}
	if err := actor.Require(RoleAdmin); err != nil {
		return User{}, err
	}
	roles, err := ParseRoleList(input.Roles)
	if err != nil {
		return User{}, err
	}
	user := User{
		ID:          NewID("usr"),
		Username:    strings.TrimSpace(input.Username),
		DisplayName: strings.TrimSpace(input.DisplayName),
		Email:       strings.TrimSpace(input.Email),
		StudentNo:   strings.TrimSpace(input.StudentNo),
		EmployeeNo:  strings.TrimSpace(input.EmployeeNo),
		Status:      normalizeStatus(input.Status, "active"),
	}
	if err := validateUser(user); err != nil {
		return User{}, err
	}
	audit.Action = "user.create"
	audit.ActorID = actor.ID
	audit.TargetType = "user"
	audit.TargetID = user.ID
	return s.repo.CreateUser(ctx, user, roles, audit)
}

func (s *Service) ListUsers(ctx context.Context, actor Actor, limit int) ([]User, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if err := actor.Require(RoleAdmin); err != nil {
		return nil, err
	}
	return s.repo.ListUsers(ctx, clampLimit(limit))
}

func (s *Service) SetUserRoles(ctx context.Context, actor Actor, userID string, roles []string, audit AuditEntry) error {
	if err := s.ready(); err != nil {
		return err
	}
	if err := actor.Require(RoleAdmin); err != nil {
		return err
	}
	parsed, err := ParseRoleList(roles)
	if err != nil {
		return err
	}
	audit.Action = "user.set_roles"
	audit.ActorID = actor.ID
	audit.TargetType = "user"
	audit.TargetID = userID
	return s.repo.SetUserRoles(ctx, strings.TrimSpace(userID), parsed, audit)
}

type CreateClassInput struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	GradeYear int    `json:"grade_year,omitempty"`
	Major     string `json:"major,omitempty"`
	Status    string `json:"status,omitempty"`
}

func (s *Service) CreateClass(ctx context.Context, actor Actor, input CreateClassInput, audit AuditEntry) (Class, error) {
	if err := s.ready(); err != nil {
		return Class{}, err
	}
	if err := actor.Require(RoleAdmin); err != nil {
		return Class{}, err
	}
	class := Class{ID: NewID("cls"), Code: strings.TrimSpace(input.Code), Name: strings.TrimSpace(input.Name), GradeYear: input.GradeYear, Major: strings.TrimSpace(input.Major), Status: normalizeStatus(input.Status, "active")}
	if err := validateClass(class); err != nil {
		return Class{}, err
	}
	audit.Action = "class.create"
	audit.ActorID = actor.ID
	audit.TargetType = "class"
	audit.TargetID = class.ID
	return s.repo.CreateClass(ctx, class, audit)
}

type CreateCourseInput struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Term   string `json:"term"`
	Status string `json:"status,omitempty"`
}

func (s *Service) CreateCourse(ctx context.Context, actor Actor, input CreateCourseInput, audit AuditEntry) (Course, error) {
	if err := s.ready(); err != nil {
		return Course{}, err
	}
	if err := actor.Require(RoleAdmin); err != nil {
		return Course{}, err
	}
	course := Course{ID: NewID("crs"), Code: strings.TrimSpace(input.Code), Name: strings.TrimSpace(input.Name), Term: strings.TrimSpace(input.Term), Status: normalizeStatus(input.Status, "draft"), CreatedBy: actor.ID}
	if err := validateCourse(course); err != nil {
		return Course{}, err
	}
	audit.Action = "course.create"
	audit.ActorID = actor.ID
	audit.TargetType = "course"
	audit.TargetID = course.ID
	return s.repo.CreateCourse(ctx, course, audit)
}

func (s *Service) AddCourseClass(ctx context.Context, actor Actor, courseID, classID string, audit AuditEntry) error {
	if err := s.ready(); err != nil {
		return err
	}
	if err := actor.Require(RoleAdmin); err != nil {
		return err
	}
	audit.Action = "course.add_class"
	audit.ActorID = actor.ID
	audit.TargetType = "course"
	audit.TargetID = courseID
	return s.repo.AddCourseClass(ctx, strings.TrimSpace(courseID), strings.TrimSpace(classID), audit)
}

type AssignTeacherInput struct {
	TeacherID  string `json:"teacher_id"`
	Permission string `json:"permission,omitempty"`
}

func (s *Service) AssignTeacher(ctx context.Context, actor Actor, courseID string, input AssignTeacherInput, audit AuditEntry) error {
	if err := s.ready(); err != nil {
		return err
	}
	if err := actor.Require(RoleAdmin); err != nil {
		return err
	}
	teacherID := strings.TrimSpace(input.TeacherID)
	hasRole, err := s.repo.UserHasRole(ctx, teacherID, RoleTeacher)
	if err != nil {
		return err
	}
	if !hasRole {
		return validationError("teacher_id must reference a user with teacher role")
	}
	assignment := CourseTeacher{CourseID: strings.TrimSpace(courseID), TeacherID: teacherID, Permission: normalizePermission(input.Permission)}
	if err := validatePermission(assignment.Permission); err != nil {
		return err
	}
	audit.Action = "course.assign_teacher"
	audit.ActorID = actor.ID
	audit.TargetType = "course"
	audit.TargetID = assignment.CourseID
	return s.repo.AssignTeacher(ctx, assignment, audit)
}

type EnrollStudentInput struct {
	ClassID   string `json:"class_id,omitempty"`
	StudentID string `json:"student_id"`
	Status    string `json:"status,omitempty"`
}

func (s *Service) EnrollStudent(ctx context.Context, actor Actor, courseID string, input EnrollStudentInput, audit AuditEntry) error {
	if err := s.ready(); err != nil {
		return err
	}
	if err := actor.Require(RoleAdmin); err != nil {
		return err
	}
	studentID := strings.TrimSpace(input.StudentID)
	hasRole, err := s.repo.UserHasRole(ctx, studentID, RoleStudent)
	if err != nil {
		return err
	}
	if !hasRole {
		return validationError("student_id must reference a user with student role")
	}
	enrollment := Enrollment{CourseID: strings.TrimSpace(courseID), ClassID: strings.TrimSpace(input.ClassID), StudentID: studentID, Status: normalizeStatus(input.Status, "active")}
	if enrollment.Status != "active" && enrollment.Status != "dropped" {
		return validationError("invalid enrollment status")
	}
	audit.Action = "enrollment.upsert"
	audit.ActorID = actor.ID
	audit.TargetType = "course"
	audit.TargetID = enrollment.CourseID
	return s.repo.EnrollStudent(ctx, enrollment, audit)
}

type CreateRubricTemplateInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Scope       string `json:"scope,omitempty"`
	Status      string `json:"status,omitempty"`
}

func (s *Service) CreateRubricTemplate(ctx context.Context, actor Actor, input CreateRubricTemplateInput, audit AuditEntry) (RubricTemplate, error) {
	if err := s.ready(); err != nil {
		return RubricTemplate{}, err
	}
	if !actor.Has(RoleTeacher) && !actor.Has(RoleAdmin) {
		return RubricTemplate{}, forbiddenError("teacher or admin role is required")
	}
	template := RubricTemplate{ID: NewID("rbt"), Name: strings.TrimSpace(input.Name), Description: strings.TrimSpace(input.Description), OwnerID: actor.ID, Scope: normalizeStatus(input.Scope, "private"), Status: normalizeStatus(input.Status, "draft")}
	if err := validateRubricTemplate(template); err != nil {
		return RubricTemplate{}, err
	}
	if template.Scope == "global" && !actor.Has(RoleAdmin) {
		return RubricTemplate{}, forbiddenError("admin role is required for global rubric templates")
	}
	audit.Action = "rubric_template.create"
	audit.ActorID = actor.ID
	audit.TargetType = "rubric_template"
	audit.TargetID = template.ID
	return s.repo.CreateRubricTemplate(ctx, template, audit)
}

type CreateRubricVersionInput struct {
	WeightMode string        `json:"weight_mode"`
	Metrics    []MetricInput `json:"metrics"`
}

type RubricVersionWithMetrics struct {
	Version RubricTemplateVersion `json:"version"`
	Metrics []Metric              `json:"metrics"`
}

func (s *Service) CreateRubricVersion(ctx context.Context, actor Actor, templateID string, input CreateRubricVersionInput, audit AuditEntry) (RubricVersionWithMetrics, error) {
	if err := s.ready(); err != nil {
		return RubricVersionWithMetrics{}, err
	}
	if !actor.Has(RoleTeacher) && !actor.Has(RoleAdmin) {
		return RubricVersionWithMetrics{}, forbiddenError("teacher or admin role is required")
	}
	if !actor.Has(RoleAdmin) {
		ownerID, err := s.repo.RubricTemplateOwner(ctx, strings.TrimSpace(templateID))
		if err != nil {
			return RubricVersionWithMetrics{}, err
		}
		if ownerID != actor.ID {
			return RubricVersionWithMetrics{}, forbiddenError("teacher can only modify owned rubric templates")
		}
	}
	mode, err := ParseWeightMode(input.WeightMode)
	if err != nil {
		return RubricVersionWithMetrics{}, err
	}
	if err := ValidateMetrics(mode, input.Metrics); err != nil {
		return RubricVersionWithMetrics{}, err
	}
	metrics := make([]Metric, 0, len(input.Metrics))
	total := 0
	for _, inputMetric := range input.Metrics {
		total += inputMetric.WeightBPS
		metrics = append(metrics, Metric{ID: NewID("rbm"), Code: normalizeCode(inputMetric.Code), Name: strings.TrimSpace(inputMetric.Name), Description: strings.TrimSpace(inputMetric.Description), WeightBPS: inputMetric.WeightBPS, MaxScore: inputMetric.MaxScore, SortOrder: inputMetric.SortOrder, RequiredEvidence: defaultJSON(inputMetric.RequiredEvidence)})
	}
	version := RubricTemplateVersion{ID: NewID("rbv"), TemplateID: strings.TrimSpace(templateID), Status: "draft", WeightMode: mode, TotalWeightBPS: total, CreatedBy: actor.ID}
	audit.Action = "rubric_version.create"
	audit.ActorID = actor.ID
	audit.TargetType = "rubric_template"
	audit.TargetID = version.TemplateID
	createdVersion, createdMetrics, err := s.repo.CreateRubricVersion(ctx, version, metrics, audit)
	if err != nil {
		return RubricVersionWithMetrics{}, err
	}
	return RubricVersionWithMetrics{Version: createdVersion, Metrics: createdMetrics}, nil
}

func (s *Service) PublishRubricVersion(ctx context.Context, actor Actor, versionID string, audit AuditEntry) (RubricTemplateVersion, error) {
	if err := s.ready(); err != nil {
		return RubricTemplateVersion{}, err
	}
	if !actor.Has(RoleTeacher) && !actor.Has(RoleAdmin) {
		return RubricTemplateVersion{}, forbiddenError("teacher or admin role is required")
	}
	if !actor.Has(RoleAdmin) {
		ownerID, err := s.repo.RubricVersionOwner(ctx, strings.TrimSpace(versionID))
		if err != nil {
			return RubricTemplateVersion{}, err
		}
		if ownerID != actor.ID {
			return RubricTemplateVersion{}, forbiddenError("teacher can only publish owned rubric versions")
		}
	}
	audit.Action = "rubric.publish_version"
	audit.ActorID = actor.ID
	audit.TargetType = "rubric_template_version"
	audit.TargetID = versionID
	return s.repo.PublishRubricVersion(ctx, strings.TrimSpace(versionID), audit)
}

type CreateExperimentInput struct {
	Title           string          `json:"title"`
	Description     string          `json:"description,omitempty"`
	SubmissionSpec  json.RawMessage `json:"submission_spec,omitempty"`
	RubricVersionID string          `json:"rubric_version_id"`
	StartAt         *time.Time      `json:"start_at,omitempty"`
	DueAt           *time.Time      `json:"due_at,omitempty"`
}

func (s *Service) CreateExperiment(ctx context.Context, actor Actor, courseID string, input CreateExperimentInput, audit AuditEntry) (Experiment, error) {
	if err := s.ready(); err != nil {
		return Experiment{}, err
	}
	if !actor.Has(RoleTeacher) && !actor.Has(RoleAdmin) {
		return Experiment{}, forbiddenError("teacher or admin role is required")
	}
	if !actor.Has(RoleAdmin) {
		allowed, err := s.repo.TeacherCanEditCourse(ctx, strings.TrimSpace(courseID), actor.ID)
		if err != nil {
			return Experiment{}, err
		}
		if !allowed {
			return Experiment{}, forbiddenError("teacher is not assigned to this course")
		}
	}
	if err := ValidateTimeWindow(input.StartAt, input.DueAt); err != nil {
		return Experiment{}, err
	}
	status, err := s.repo.RubricVersionStatus(ctx, strings.TrimSpace(input.RubricVersionID))
	if err != nil {
		return Experiment{}, err
	}
	if status != "published" {
		return Experiment{}, validationError("experiment must bind a published rubric version")
	}
	experiment := Experiment{ID: NewID("exp"), CourseID: strings.TrimSpace(courseID), Title: strings.TrimSpace(input.Title), Description: strings.TrimSpace(input.Description), SubmissionSpec: defaultJSON(input.SubmissionSpec), RubricVersionID: strings.TrimSpace(input.RubricVersionID), Status: "draft", StartAt: input.StartAt, DueAt: input.DueAt, CreatedBy: actor.ID}
	if err := validateExperiment(experiment); err != nil {
		return Experiment{}, err
	}
	audit.Action = "experiment.create"
	audit.ActorID = actor.ID
	audit.TargetType = "experiment"
	audit.TargetID = experiment.ID
	return s.repo.CreateExperiment(ctx, experiment, audit)
}

func (s *Service) PublishExperiment(ctx context.Context, actor Actor, experimentID string, audit AuditEntry) (Experiment, error) {
	if err := s.ready(); err != nil {
		return Experiment{}, err
	}
	if !actor.Has(RoleTeacher) && !actor.Has(RoleAdmin) {
		return Experiment{}, forbiddenError("teacher or admin role is required")
	}
	if !actor.Has(RoleAdmin) {
		courseID, err := s.repo.ExperimentCourseID(ctx, strings.TrimSpace(experimentID))
		if err != nil {
			return Experiment{}, err
		}
		allowed, err := s.repo.TeacherCanEditCourse(ctx, courseID, actor.ID)
		if err != nil {
			return Experiment{}, err
		}
		if !allowed {
			return Experiment{}, forbiddenError("teacher is not assigned to this course")
		}
	}
	audit.Action = "experiment.publish"
	audit.ActorID = actor.ID
	audit.TargetType = "experiment"
	audit.TargetID = experimentID
	return s.repo.PublishExperiment(ctx, strings.TrimSpace(experimentID), audit)
}

func (s *Service) ready() error {
	if s == nil || s.repo == nil {
		return unavailableError("teaching repository is not configured", nil)
	}
	return nil
}

func validateUser(user User) error {
	if user.Username == "" {
		return validationError("username is required")
	}
	if user.DisplayName == "" {
		return validationError("display_name is required")
	}
	if user.Status != "active" && user.Status != "disabled" {
		return validationError("invalid user status")
	}
	return nil
}

func validateClass(class Class) error {
	if class.Code == "" || class.Name == "" {
		return validationError("class code and name are required")
	}
	if class.Status != "active" && class.Status != "archived" {
		return validationError("invalid class status")
	}
	return nil
}

func validateCourse(course Course) error {
	if course.Code == "" || course.Name == "" || course.Term == "" {
		return validationError("course code, name and term are required")
	}
	if course.Status != "draft" && course.Status != "active" && course.Status != "archived" {
		return validationError("invalid course status")
	}
	return nil
}

func validateRubricTemplate(template RubricTemplate) error {
	if template.Name == "" {
		return validationError("rubric template name is required")
	}
	if template.Scope != "private" && template.Scope != "course" && template.Scope != "global" {
		return validationError("invalid rubric template scope")
	}
	if template.Status != "draft" && template.Status != "active" && template.Status != "archived" {
		return validationError("invalid rubric template status")
	}
	return nil
}

func validateExperiment(experiment Experiment) error {
	if experiment.CourseID == "" || experiment.Title == "" || experiment.RubricVersionID == "" {
		return validationError("course_id, title and rubric_version_id are required")
	}
	if len(experiment.SubmissionSpec) > 0 && !json.Valid(experiment.SubmissionSpec) {
		return validationError("submission_spec must be valid JSON")
	}
	return nil
}

func validatePermission(permission string) error {
	switch permission {
	case "owner", "editor", "viewer":
		return nil
	default:
		return validationError("invalid teacher permission")
	}
}

func normalizePermission(permission string) string {
	permission = strings.ToLower(strings.TrimSpace(permission))
	if permission == "" {
		return "editor"
	}
	return permission
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}
