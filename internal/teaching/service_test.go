package teaching

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestValidateMetricsStrictAndNormalized(t *testing.T) {
	strict := []MetricInput{
		{Code: "quality", Name: "Code quality", WeightBPS: 6000, MaxScore: 100, SortOrder: 1},
		{Code: "docs", Name: "Documentation", WeightBPS: 4000, MaxScore: 100, SortOrder: 2},
	}
	if err := ValidateMetrics(WeightModeStrict100, strict); err != nil {
		t.Fatalf("strict metrics should pass: %v", err)
	}

	badStrict := []MetricInput{{Code: "quality", Name: "Code quality", WeightBPS: 9999, MaxScore: 100, SortOrder: 1}}
	if err := ValidateMetrics(WeightModeStrict100, badStrict); ErrorKindOf(err) != KindValidation {
		t.Fatalf("strict metrics with non-100%% total should fail validation, got %v", err)
	}

	normalized := []MetricInput{{Code: "quality", Name: "Code quality", WeightBPS: 1, MaxScore: 100, SortOrder: 1}}
	if err := ValidateMetrics(WeightModeNormalized, normalized); err != nil {
		t.Fatalf("normalized metrics should pass when total weight is positive: %v", err)
	}

	zeroNormalized := []MetricInput{{Code: "quality", Name: "Code quality", WeightBPS: 0, MaxScore: 100, SortOrder: 1}}
	if err := ValidateMetrics(WeightModeNormalized, zeroNormalized); ErrorKindOf(err) != KindValidation {
		t.Fatalf("normalized metrics with zero total should fail validation, got %v", err)
	}
}

func TestValidateMetricsValidationFailures(t *testing.T) {
	tests := []struct {
		name    string
		mode    WeightMode
		metrics []MetricInput
	}{
		{
			name: "duplicate codes",
			mode: WeightModeNormalized,
			metrics: []MetricInput{
				{Code: "quality", Name: "Code quality", WeightBPS: 5000, MaxScore: 100, SortOrder: 1},
				{Code: "quality", Name: "Quality duplicate", WeightBPS: 5000, MaxScore: 100, SortOrder: 2},
			},
		},
		{
			name: "duplicate sort order",
			mode: WeightModeNormalized,
			metrics: []MetricInput{
				{Code: "quality", Name: "Code quality", WeightBPS: 5000, MaxScore: 100, SortOrder: 1},
				{Code: "docs", Name: "Documentation", WeightBPS: 5000, MaxScore: 100, SortOrder: 1},
			},
		},
		{
			name: "negative weight",
			mode: WeightModeNormalized,
			metrics: []MetricInput{
				{Code: "quality", Name: "Code quality", WeightBPS: -100, MaxScore: 100, SortOrder: 1},
				{Code: "docs", Name: "Documentation", WeightBPS: 10100, MaxScore: 100, SortOrder: 2},
			},
		},
		{
			name: "non-positive max score",
			mode: WeightModeNormalized,
			metrics: []MetricInput{
				{Code: "quality", Name: "Code quality", WeightBPS: 5000, MaxScore: 0, SortOrder: 1},
				{Code: "docs", Name: "Documentation", WeightBPS: 5000, MaxScore: 100, SortOrder: 2},
			},
		},
		{
			name: "invalid required evidence JSON",
			mode: WeightModeNormalized,
			metrics: []MetricInput{
				{Code: "quality", Name: "Code quality", WeightBPS: 10000, MaxScore: 100, SortOrder: 1, RequiredEvidence: json.RawMessage(`{invalid`)},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateMetrics(tc.mode, tc.metrics)
			if ErrorKindOf(err) != KindValidation {
				t.Fatalf("expected KindValidation error, got %v", err)
			}
		})
	}
}

func TestCreateRubricVersionRequiresTemplateOwner(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{rubricTemplateOwner: "teacher-2"})
	_, err = service.CreateRubricVersion(context.Background(), actor, "template-1", validRubricVersionInput(), AuditEntry{})
	if ErrorKindOf(err) != KindForbidden {
		t.Fatalf("expected forbidden when teacher edits another owner's template, got %v", err)
	}
}

func TestCreateRubricVersionBindsImmutableDraftVersion(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepo{rubricTemplateOwner: "teacher-1"}
	service := NewService(repo)
	created, err := service.CreateRubricVersion(context.Background(), actor, "template-1", validRubricVersionInput(), AuditEntry{})
	if err != nil {
		t.Fatalf("create rubric version: %v", err)
	}
	if created.Version.TemplateID != "template-1" || created.Version.Status != "draft" || created.Version.TotalWeightBPS != WeightTotalBPS {
		t.Fatalf("unexpected version: %+v", created.Version)
	}
	if len(created.Metrics) != 2 || created.Metrics[0].VersionID != created.Version.ID {
		t.Fatalf("metrics should be bound to created version: %+v", created.Metrics)
	}
	if !repo.createRubricVersionCalled {
		t.Fatal("repository CreateRubricVersion was not called")
	}
}

func TestCreateExperimentRequiresPublishedRubricVersion(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{teacherAllowed: true, rubricVersionStatus: "draft"})
	_, err = service.CreateExperiment(context.Background(), actor, "course-1", CreateExperimentInput{Title: "Lab 1", RubricVersionID: "version-1"}, AuditEntry{})
	if ErrorKindOf(err) != KindValidation {
		t.Fatalf("draft rubric version should not bind to experiment, got %v", err)
	}
}

func TestPublishExperimentRequiresAssignedTeacher(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{experimentCourseID: "course-1", teacherAllowed: false})
	_, err = service.PublishExperiment(context.Background(), actor, "experiment-1", AuditEntry{})
	if ErrorKindOf(err) != KindForbidden {
		t.Fatalf("unassigned teacher should not publish experiment, got %v", err)
	}
}

func validRubricVersionInput() CreateRubricVersionInput {
	return CreateRubricVersionInput{
		WeightMode: string(WeightModeStrict100),
		Metrics: []MetricInput{
			{Code: "quality", Name: "Code quality", WeightBPS: 6000, MaxScore: 100, SortOrder: 1},
			{Code: "docs", Name: "Documentation", WeightBPS: 4000, MaxScore: 100, SortOrder: 2},
		},
	}
}

type fakeRepo struct {
	rubricTemplateOwner       string
	rubricVersionOwner        string
	rubricVersionStatus       string
	experimentCourseID        string
	teacherAllowed            bool
	createRubricVersionCalled bool
}

func (f *fakeRepo) CreateUser(context.Context, User, []Role, AuditEntry) (User, error) {
	return User{}, errors.New("not implemented")
}
func (f *fakeRepo) ListUsers(context.Context, int) ([]User, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeRepo) SetUserRoles(context.Context, string, []Role, AuditEntry) error {
	return errors.New("not implemented")
}
func (f *fakeRepo) UserHasRole(context.Context, string, Role) (bool, error) { return false, nil }
func (f *fakeRepo) CreateClass(context.Context, Class, AuditEntry) (Class, error) {
	return Class{}, errors.New("not implemented")
}
func (f *fakeRepo) CreateCourse(context.Context, Course, AuditEntry) (Course, error) {
	return Course{}, errors.New("not implemented")
}
func (f *fakeRepo) AddCourseClass(context.Context, string, string, AuditEntry) error { return nil }
func (f *fakeRepo) TeacherCanEditCourse(context.Context, string, string) (bool, error) {
	return f.teacherAllowed, nil
}
func (f *fakeRepo) AssignTeacher(context.Context, CourseTeacher, AuditEntry) error { return nil }
func (f *fakeRepo) EnrollStudent(context.Context, Enrollment, AuditEntry) error    { return nil }
func (f *fakeRepo) CreateRubricTemplate(context.Context, RubricTemplate, AuditEntry) (RubricTemplate, error) {
	return RubricTemplate{}, errors.New("not implemented")
}
func (f *fakeRepo) RubricTemplateOwner(context.Context, string) (string, error) {
	return f.rubricTemplateOwner, nil
}
func (f *fakeRepo) CreateRubricVersion(_ context.Context, version RubricTemplateVersion, metrics []Metric, _ AuditEntry) (RubricTemplateVersion, []Metric, error) {
	f.createRubricVersionCalled = true
	version.VersionNo = 1
	for i := range metrics {
		metrics[i].VersionID = version.ID
	}
	return version, metrics, nil
}
func (f *fakeRepo) RubricVersionOwner(context.Context, string) (string, error) {
	if f.rubricVersionOwner != "" {
		return f.rubricVersionOwner, nil
	}
	return f.rubricTemplateOwner, nil
}
func (f *fakeRepo) PublishRubricVersion(context.Context, string, AuditEntry) (RubricTemplateVersion, error) {
	return RubricTemplateVersion{}, nil
}
func (f *fakeRepo) RubricVersionStatus(context.Context, string) (string, error) {
	return f.rubricVersionStatus, nil
}
func (f *fakeRepo) CreateExperiment(_ context.Context, experiment Experiment, _ AuditEntry) (Experiment, error) {
	return experiment, nil
}
func (f *fakeRepo) ExperimentCourseID(context.Context, string) (string, error) {
	return f.experimentCourseID, nil
}
func (f *fakeRepo) PublishExperiment(_ context.Context, experimentID string, _ AuditEntry) (Experiment, error) {
	return Experiment{ID: experimentID, Status: "published"}, nil
}
