package teaching

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
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

func TestCreateSubmissionRequiresEnrollmentAndPublishedExperiment(t *testing.T) {
	actor, err := NewActor("student-1", []Role{RoleStudent})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{submissionAccess: ExperimentSubmissionAccess{Status: "published", Enrolled: false}})
	_, err = service.CreateSubmission(context.Background(), actor, "experiment-1", CreateSubmissionInput{}, AuditEntry{})
	if ErrorKindOf(err) != KindForbidden {
		t.Fatalf("unenrolled student should be forbidden, got %v", err)
	}

	service = NewService(&fakeRepo{submissionAccess: ExperimentSubmissionAccess{Status: "draft", Enrolled: true}})
	_, err = service.CreateSubmission(context.Background(), actor, "experiment-1", CreateSubmissionInput{}, AuditEntry{})
	if ErrorKindOf(err) != KindValidation {
		t.Fatalf("unpublished experiment should fail validation, got %v", err)
	}

	pastDue := time.Now().Add(-1 * time.Hour)
	service = NewService(&fakeRepo{submissionAccess: ExperimentSubmissionAccess{Status: "published", Enrolled: true, DueAt: &pastDue}})
	_, err = service.CreateSubmission(context.Background(), actor, "experiment-1", CreateSubmissionInput{}, AuditEntry{})
	if ErrorKindOf(err) != KindValidation {
		t.Fatalf("past due experiment should fail validation, got %v", err)
	}

	futureDue := time.Now().Add(time.Hour)
	service = NewService(&fakeRepo{submissionAccess: ExperimentSubmissionAccess{Status: "published", Enrolled: true, DueAt: &futureDue}})
	submission, err := service.CreateSubmission(context.Background(), actor, "experiment-1", CreateSubmissionInput{}, AuditEntry{})
	if err != nil {
		t.Fatalf("published experiment should allow enrolled student submission: %v", err)
	}
	if submission.ExperimentID != "experiment-1" || submission.StudentID != "student-1" || submission.Status != "draft" || submission.AttemptNo != 1 {
		t.Fatalf("unexpected submission: %+v", submission)
	}
}

func TestCreateGitLinkArtifactRequiresOwnedSubmission(t *testing.T) {
	actor, err := NewActor("student-1", []Role{RoleStudent})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{ownsSubmission: false})
	_, err = service.CreateGitLinkArtifact(context.Background(), actor, "submission-1", CreateGitLinkInput{URL: "https://example.edu/repo.git"}, AuditEntry{})
	if ErrorKindOf(err) != KindForbidden {
		t.Fatalf("student should not attach artifacts to another submission, got %v", err)
	}
}

func TestCreateGitLinkArtifactSuccess(t *testing.T) {
	actor, err := NewActor("student-1", []Role{RoleStudent})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{ownsSubmission: true}, WithUploadLimits(DefaultMaxUploadBytes, 3))
	result, err := service.CreateGitLinkArtifact(context.Background(), actor, "submission-1", CreateGitLinkInput{
		URL:       "https://example.edu/repo.git",
		CommitSHA: "0123456789abcdef0123456789abcdef01234567",
	}, AuditEntry{})
	if err != nil {
		t.Fatalf("CreateGitLinkArtifact should succeed: %v", err)
	}
	if result.Artifact.Kind != ArtifactKindGitLink || result.Artifact.Status != "stored" {
		t.Fatalf("unexpected artifact: %+v", result.Artifact)
	}
	if result.Extraction.Status != "succeeded" || result.Extraction.TextExcerpt == "" {
		t.Fatalf("unexpected extraction: %+v", result.Extraction)
	}
	var metadata map[string]string
	if err := json.Unmarshal(result.Artifact.Metadata, &metadata); err != nil {
		t.Fatalf("metadata should be JSON object: %v", err)
	}
	if metadata["url_host"] != "example.edu" || metadata["commit_sha"] != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("unexpected git metadata: %+v", metadata)
	}
}

func TestCreateGitLinkArtifactValidationErrors(t *testing.T) {
	actor, err := NewActor("student-1", []Role{RoleStudent})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{ownsSubmission: true})
	tests := []struct {
		name  string
		input CreateGitLinkInput
	}{
		{name: "relative", input: CreateGitLinkInput{URL: "repo.git"}},
		{name: "scheme", input: CreateGitLinkInput{URL: "ftp://example.edu/repo.git"}},
		{name: "host", input: CreateGitLinkInput{URL: "https:///repo.git"}},
		{name: "short sha", input: CreateGitLinkInput{URL: "https://example.edu/repo.git", CommitSHA: "abc123"}},
		{name: "non hex sha", input: CreateGitLinkInput{URL: "https://example.edu/repo.git", CommitSHA: "zzzzzzzz"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.CreateGitLinkArtifact(context.Background(), actor, "submission-1", tc.input, AuditEntry{})
			if ErrorKindOf(err) != KindValidation {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}

func TestCreateGitLinkArtifactRespectsArtifactLimit(t *testing.T) {
	actor, err := NewActor("student-1", []Role{RoleStudent})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{ownsSubmission: true, artifactCount: 3}, WithUploadLimits(DefaultMaxUploadBytes, 3))
	_, err = service.CreateGitLinkArtifact(context.Background(), actor, "submission-1", CreateGitLinkInput{URL: "https://example.edu/repo.git"}, AuditEntry{})
	if ErrorKindOf(err) != KindValidation {
		t.Fatalf("expected validation when artifact limit is reached, got %v", err)
	}
}

func TestSetUserPasswordRequiresAdmin(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{})
	err = service.SetUserPassword(context.Background(), actor, "user-1", SetUserPasswordInput{Password: "test-pass"}, AuditEntry{})
	if ErrorKindOf(err) != KindForbidden {
		t.Fatalf("expected forbidden, got %v", err)
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
	rubricTemplateOwner               string
	rubricVersionOwner                string
	rubricVersionStatus               string
	experimentCourseID                string
	teacherAllowed                    bool
	createRubricVersionCalled         bool
	submissionAccess                  ExperimentSubmissionAccess
	ownsSubmission                    bool
	artifactCount                     int
	evaluationContext                 EvaluationContext
	latestEvaluation                  EvaluationResultDetail
	evaluationResultSubmissionID      string
	createdEvaluation                 EvaluationResultDetail
	teacherReview                     TeacherReviewDetail
	publishedReview                   TeacherReviewDetail
	lastGetTeacherReviewPublishedOnly bool
	reportExports                     map[string]ReportExport
	experimentSummaries               map[string]experimentReportItem
	courseExperiments                 []Experiment
	userCount                         int
	lastPasswordHash                  string
}

type experimentReportItem struct {
	detail     SubmissionDetail
	review     TeacherReviewDetail
	evaluation EvaluationResultDetail
}

func (f *fakeRepo) CreateUser(context.Context, User, []Role, AuditEntry) (User, error) {
	return User{}, errors.New("not implemented")
}
func (f *fakeRepo) CountUsers(context.Context) (int, error) {
	return f.userCount, nil
}
func (f *fakeRepo) ListUsers(context.Context, int) ([]User, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeRepo) SetUserRoles(context.Context, string, []Role, AuditEntry) error {
	return errors.New("not implemented")
}
func (f *fakeRepo) SetUserPassword(_ context.Context, _ string, passwordHash string, _ AuditEntry) error {
	f.lastPasswordHash = passwordHash
	return nil
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
func (f *fakeRepo) ExperimentSubmissionAccess(context.Context, string, string) (ExperimentSubmissionAccess, error) {
	return f.submissionAccess, nil
}
func (f *fakeRepo) ListExperimentsForCourse(context.Context, string, int) ([]Experiment, error) {
	return f.courseExperiments, nil
}
func (f *fakeRepo) CreateSubmission(_ context.Context, submission Submission, _ AuditEntry) (Submission, error) {
	return submission, nil
}
func (f *fakeRepo) StudentOwnsSubmission(context.Context, string, string) (bool, error) {
	return f.ownsSubmission, nil
}
func (f *fakeRepo) SubmissionCourseID(context.Context, string) (string, error) {
	return "course-1", nil
}
func (f *fakeRepo) SubmissionArtifactCount(context.Context, string) (int, error) {
	return f.artifactCount, nil
}
func (f *fakeRepo) CreateArtifact(_ context.Context, artifact Artifact, extraction ExtractedContent, job *QueuedJob, _ AuditEntry) (ArtifactWithExtraction, error) {
	result := ArtifactWithExtraction{Artifact: artifact, Extraction: extraction}
	if job != nil {
		result.JobID = job.ID
	}
	return result, nil
}
func (f *fakeRepo) ListSubmissionsForExperiment(_ context.Context, experimentID string, _ int) ([]Submission, error) {
	if len(f.experimentSummaries) > 0 {
		submissions := make([]Submission, 0, len(f.experimentSummaries))
		for _, item := range f.experimentSummaries {
			if item.detail.Submission.ExperimentID == experimentID {
				submissions = append(submissions, item.detail.Submission)
			}
		}
		return submissions, nil
	}
	return nil, nil
}
func (f *fakeRepo) GetSubmissionDetail(_ context.Context, submissionID string) (SubmissionDetail, error) {
	if item, ok := f.experimentSummaries[submissionID]; ok {
		return item.detail, nil
	}
	return SubmissionDetail{}, nil
}
func (f *fakeRepo) GetEvaluationContext(context.Context, string) (EvaluationContext, error) {
	return f.evaluationContext, nil
}
func (f *fakeRepo) CreateInitialEvaluation(_ context.Context, result EvaluationResult, findings []RuleCheckFinding, scores []MetricScore, _ *LLMCallLog, _ AuditEntry) (EvaluationResultDetail, error) {
	f.createdEvaluation = EvaluationResultDetail{Result: result, Findings: findings, Scores: scores}
	return f.createdEvaluation, nil
}
func (f *fakeRepo) GetLatestEvaluation(_ context.Context, submissionID string) (EvaluationResultDetail, error) {
	if item, ok := f.experimentSummaries[submissionID]; ok {
		return item.evaluation, nil
	}
	return f.latestEvaluation, nil
}
func (f *fakeRepo) EvaluationResultSubmissionID(context.Context, string) (string, error) {
	if f.evaluationResultSubmissionID != "" {
		return f.evaluationResultSubmissionID, nil
	}
	return "submission-1", nil
}
func (f *fakeRepo) UpsertTeacherReview(_ context.Context, review TeacherReview, scores []TeacherMetricScore, _ AuditEntry) (TeacherReviewDetail, error) {
	f.teacherReview = TeacherReviewDetail{Review: review, Scores: scores}
	return f.teacherReview, nil
}
func (f *fakeRepo) PublishTeacherReview(_ context.Context, submissionID, actorID string, _ AuditEntry) (TeacherReviewDetail, error) {
	f.publishedReview.Review.SubmissionID = submissionID
	f.publishedReview.Review.PublishedBy = actorID
	f.publishedReview.Review.Status = TeacherReviewStatusPublished
	return f.publishedReview, nil
}
func (f *fakeRepo) GetTeacherReview(_ context.Context, submissionID string, publishedOnly bool) (TeacherReviewDetail, error) {
	f.lastGetTeacherReviewPublishedOnly = publishedOnly
	if item, ok := f.experimentSummaries[submissionID]; ok {
		return item.review, nil
	}
	return f.teacherReview, nil
}
func (f *fakeRepo) CreateReportExport(_ context.Context, export ReportExport, _ AuditEntry) (ReportExport, error) {
	if f.reportExports == nil {
		f.reportExports = map[string]ReportExport{}
	}
	f.reportExports[export.ID] = export
	return export, nil
}
func (f *fakeRepo) CompleteReportExport(_ context.Context, export ReportExport) (ReportExport, error) {
	if f.reportExports == nil {
		f.reportExports = map[string]ReportExport{}
	}
	f.reportExports[export.ID] = export
	return export, nil
}
func (f *fakeRepo) GetReportExport(_ context.Context, exportID string) (ReportExport, error) {
	if export, ok := f.reportExports[exportID]; ok {
		return export, nil
	}
	return ReportExport{}, notFoundError("resource not found")
}
