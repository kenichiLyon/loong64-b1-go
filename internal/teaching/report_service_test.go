package teaching

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestGetSubmissionReportStudentOnlyOwnPublished(t *testing.T) {
	actor, err := NewActor("student-1", []Role{RoleStudent})
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepo{
		ownsSubmission:    true,
		evaluationContext: validReportEvaluationContext(),
		teacherReview:     validPublishedReportReview(),
		latestEvaluation:  validReportEvaluationDetail(),
	}
	service := NewService(repo)

	report, err := service.GetSubmissionReport(context.Background(), actor, "submission-1")
	if err != nil {
		t.Fatalf("student should read own published report: %v", err)
	}
	if report.Review.Review.ID != "review-1" || report.Evaluation == nil || len(report.Artifacts) != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if !repo.lastGetTeacherReviewPublishedOnly {
		t.Fatal("student report must only load published reviews")
	}
}

func TestGetSubmissionReportStudentRejectsOtherSubmission(t *testing.T) {
	actor, err := NewActor("student-1", []Role{RoleStudent})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{ownsSubmission: false})
	_, err = service.GetSubmissionReport(context.Background(), actor, "submission-1")
	if ErrorKindOf(err) != KindForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestCreateSubmissionReportExportHTMLWritesFile(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	repo := &fakeRepo{
		teacherAllowed:    true,
		evaluationContext: validReportEvaluationContext(),
		teacherReview:     validPublishedReportReview(),
		latestEvaluation:  validReportEvaluationDetail(),
	}
	service := NewService(repo, WithArtifactStore(testStore{root: dir}))

	export, err := service.CreateSubmissionReportExport(context.Background(), actor, "submission-1", CreateReportExportInput{Format: ReportFormatHTML}, AuditEntry{})
	if err != nil {
		t.Fatalf("html export should succeed: %v", err)
	}
	if export.Status != ReportExportStatusSucceeded || export.StorageKey == "" || export.SHA256Hex == "" || export.ByteSize == 0 {
		t.Fatalf("unexpected export: %+v", export)
	}
	content, err := os.ReadFile(mustResolve(t, testStore{root: dir}, export.StorageKey))
	if err != nil {
		t.Fatalf("read export file: %v", err)
	}
	if !strings.Contains(string(content), "学生个人评价报告") || !strings.Contains(string(content), "quality") {
		t.Fatalf("html report missing expected content: %s", string(content))
	}
}

func TestCreateSubmissionReportExportCSVWithBOM(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	repo := &fakeRepo{
		teacherAllowed:    true,
		evaluationContext: validReportEvaluationContext(),
		teacherReview:     validPublishedReportReview(),
		latestEvaluation:  validReportEvaluationDetail(),
	}
	service := NewService(repo, WithArtifactStore(testStore{root: dir}))

	export, err := service.CreateSubmissionReportExport(context.Background(), actor, "submission-1", CreateReportExportInput{Format: ReportFormatCSV}, AuditEntry{})
	if err != nil {
		t.Fatalf("csv export should succeed: %v", err)
	}
	content, err := os.ReadFile(mustResolve(t, testStore{root: dir}, export.StorageKey))
	if err != nil {
		t.Fatalf("read export file: %v", err)
	}
	if !strings.HasPrefix(string(content), "\ufeff") || !strings.Contains(string(content), "个人评价报告") {
		t.Fatalf("csv report missing BOM or content: %q", string(content[:min(len(content), 64)]))
	}
}

func TestCreateSubmissionReportExportPDFWritesFile(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepo{
		teacherAllowed:    true,
		evaluationContext: validReportEvaluationContext(),
		teacherReview:     validPublishedReportReview(),
		latestEvaluation:  validReportEvaluationDetail(),
	}
	service := NewService(repo, WithArtifactStore(testStore{root: t.TempDir()}))

	export, err := service.CreateSubmissionReportExport(context.Background(), actor, "submission-1", CreateReportExportInput{Format: ReportFormatPDF}, AuditEntry{})
	if err != nil {
		t.Fatalf("pdf export should succeed: %v", err)
	}
	if export.Status != ReportExportStatusSucceeded || !strings.HasSuffix(export.StorageKey, ".pdf") || export.ByteSize == 0 {
		t.Fatalf("unexpected pdf export status: %+v", export)
	}
}

func TestCreateExperimentSummaryExportXLSXWritesWorkbook(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	repo := &fakeRepo{
		teacherAllowed:      true,
		experimentCourseID:  "course-1",
		experimentSummaries: validExperimentReportDataset(),
	}
	service := NewService(repo, WithArtifactStore(testStore{root: dir}))

	export, err := service.CreateExperimentSummaryExport(context.Background(), actor, "experiment-1", CreateReportExportInput{Format: ReportFormatXLSX}, AuditEntry{})
	if err != nil {
		t.Fatalf("xlsx export should succeed: %v", err)
	}
	workbook, err := excelize.OpenFile(mustResolve(t, testStore{root: dir}, export.StorageKey))
	if err != nil {
		t.Fatalf("open xlsx export: %v", err)
	}
	defer func() { _ = workbook.Close() }()
	value, err := workbook.GetCellValue("Summary", "A1")
	if err != nil {
		t.Fatalf("read workbook cell: %v", err)
	}
	if value != "实验统计报表" {
		t.Fatalf("unexpected workbook content: %q", value)
	}
}

func TestGetExperimentReportSummaryAggregatesPublishedReviews(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepo{
		teacherAllowed:      true,
		experimentCourseID:  "course-1",
		experimentSummaries: validExperimentReportDataset(),
	}
	service := NewService(repo)

	summary, err := service.GetExperimentReportSummary(context.Background(), actor, "experiment-1", 50)
	if err != nil {
		t.Fatalf("summary should succeed: %v", err)
	}
	if summary.SubmissionCount != 2 || summary.PublishedReviewCount != 2 || summary.AverageScoreBPS != 8500 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.ScoreBuckets["80-89%"] != 1 || summary.ScoreBuckets["90-100%"] != 1 {
		t.Fatalf("unexpected buckets: %+v", summary.ScoreBuckets)
	}
	if len(summary.MetricAverages) != 2 || summary.MetricAverages[0].Count != 2 {
		t.Fatalf("unexpected metrics: %+v", summary.MetricAverages)
	}
}

func TestGetCourseReportSummaryAggregatesExperiments(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepo{
		teacherAllowed:      true,
		courseExperiments:   []Experiment{{ID: "experiment-1", CourseID: "course-1"}, {ID: "experiment-2", CourseID: "course-1"}},
		experimentSummaries: validCourseReportDataset(),
	}
	service := NewService(repo)

	summary, err := service.GetCourseReportSummary(context.Background(), actor, "course-1", 50)
	if err != nil {
		t.Fatalf("course summary should succeed: %v", err)
	}
	if summary.ExperimentCount != 2 || summary.SubmissionCount != 3 || summary.PublishedReviewCount != 3 || summary.AverageScoreBPS != 8000 {
		t.Fatalf("unexpected course summary: %+v", summary)
	}
	if summary.ScoreBuckets["70-79%"] != 1 || summary.ScoreBuckets["80-89%"] != 1 || summary.ScoreBuckets["90-100%"] != 1 {
		t.Fatalf("unexpected course buckets: %+v", summary.ScoreBuckets)
	}
	if len(summary.Experiments) != 2 || len(summary.MetricAverages) != 2 {
		t.Fatalf("unexpected nested summaries: %+v", summary)
	}
}

func TestCreateCourseSummaryExportCSVWritesFile(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	repo := &fakeRepo{
		teacherAllowed:      true,
		courseExperiments:   []Experiment{{ID: "experiment-1", CourseID: "course-1"}, {ID: "experiment-2", CourseID: "course-1"}},
		experimentSummaries: validCourseReportDataset(),
	}
	service := NewService(repo, WithArtifactStore(testStore{root: dir}))

	export, err := service.CreateCourseSummaryExport(context.Background(), actor, "course-1", CreateReportExportInput{Format: ReportFormatCSV}, AuditEntry{})
	if err != nil {
		t.Fatalf("course csv export should succeed: %v", err)
	}
	content, err := os.ReadFile(mustResolve(t, testStore{root: dir}, export.StorageKey))
	if err != nil {
		t.Fatalf("read export file: %v", err)
	}
	if export.ReportType != ReportTypeCourseSummary || export.ScopeType != ReportScopeCourse || !strings.Contains(string(content), "课程统计报表") || !strings.Contains(string(content), "experiment-2") {
		t.Fatalf("unexpected course export: %+v content=%q", export, string(content[:min(len(content), 96)]))
	}
}

func validReportEvaluationContext() EvaluationContext {
	return EvaluationContext{
		Submission: Submission{ID: "submission-1", ExperimentID: "experiment-1", StudentID: "student-1", Status: "submitted"},
		Experiment: Experiment{ID: "experiment-1", RubricVersionID: "rubric-version-1", Title: "软件实训一"},
		Metrics: []Metric{
			{ID: "metric-1", Code: "quality", Name: "Code quality", MaxScore: 20, WeightBPS: 6000},
			{ID: "metric-2", Code: "docs", Name: "Documentation", MaxScore: 10, WeightBPS: 4000},
		},
		Artifacts: []ArtifactWithExtraction{{
			Artifact:   Artifact{ID: "artifact-1", SubmissionID: "submission-1", Kind: ArtifactKindReport, OriginalName: "report.pdf", Status: "stored"},
			Extraction: ExtractedContent{ID: "extract-1", ArtifactID: "artifact-1", Status: "succeeded", TextExcerpt: "实验步骤完整，包含运行截图。"},
		}},
	}
}

func validPublishedReportReview() TeacherReviewDetail {
	return TeacherReviewDetail{
		Review: TeacherReview{ID: "review-1", SubmissionID: "submission-1", ExperimentID: "experiment-1", Status: TeacherReviewStatusPublished, TotalScoreBPS: 8600, TeacherComment: "整体完成度较高。"},
		Scores: []TeacherMetricScore{
			{ID: "score-1", TeacherReviewID: "review-1", MetricID: "metric-1", MetricCode: "quality", FinalScore: 18, MaxScore: 20, WeightBPS: 6000, Source: "llm", Comment: "结构清晰"},
			{ID: "score-2", TeacherReviewID: "review-1", MetricID: "metric-2", MetricCode: "docs", FinalScore: 8, MaxScore: 10, WeightBPS: 4000, Source: "manual", AdjustmentReason: "截图说明略少"},
		},
	}
}

func validReportEvaluationDetail() EvaluationResultDetail {
	return EvaluationResultDetail{
		Result:   EvaluationResult{ID: "evaluation-1", SubmissionID: "submission-1", Status: EvaluationStatusCompleted, LLMSummary: "AI 建议整体良好。"},
		Findings: []RuleCheckFinding{{ID: "finding-1", Category: "document", Severity: FindingLow, Message: "报告总结可更详细。"}},
		Scores:   []MetricScore{{ID: "metric-score-1", MetricID: "metric-1", MetricCode: "quality", Source: MetricScoreSourceLLM, SuggestedScore: 18, MaxScore: 20, ConfidenceBPS: 8000, Rationale: "证据充足"}},
	}
}

func validExperimentReportDataset() map[string]experimentReportItem {
	return map[string]experimentReportItem{
		"submission-1": {
			detail:     SubmissionDetail{Submission: Submission{ID: "submission-1", ExperimentID: "experiment-1", Status: "submitted"}, Artifacts: []ArtifactWithExtraction{{Artifact: Artifact{Kind: ArtifactKindReport}, Extraction: ExtractedContent{Status: "succeeded"}}}},
			review:     TeacherReviewDetail{Review: TeacherReview{ID: "review-1", SubmissionID: "submission-1", Status: TeacherReviewStatusPublished, TotalScoreBPS: 8000}, Scores: []TeacherMetricScore{{MetricCode: "docs", FinalScore: 8, MaxScore: 10, WeightBPS: 4000}, {MetricCode: "quality", FinalScore: 16, MaxScore: 20, WeightBPS: 6000}}},
			evaluation: EvaluationResultDetail{Result: EvaluationResult{ID: "evaluation-1"}, Findings: []RuleCheckFinding{{Category: "steps", Severity: FindingMedium, Message: "步骤说明略少"}}},
		},
		"submission-2": {
			detail:     SubmissionDetail{Submission: Submission{ID: "submission-2", ExperimentID: "experiment-1", Status: "submitted"}, Artifacts: []ArtifactWithExtraction{{Artifact: Artifact{Kind: ArtifactKindDocument}, Extraction: ExtractedContent{Status: "failed"}}}},
			review:     TeacherReviewDetail{Review: TeacherReview{ID: "review-2", SubmissionID: "submission-2", Status: TeacherReviewStatusPublished, TotalScoreBPS: 9000}, Scores: []TeacherMetricScore{{MetricCode: "docs", FinalScore: 9, MaxScore: 10, WeightBPS: 4000}, {MetricCode: "quality", FinalScore: 18, MaxScore: 20, WeightBPS: 6000}}},
			evaluation: EvaluationResultDetail{Result: EvaluationResult{ID: "evaluation-2"}, Findings: []RuleCheckFinding{{Category: "parse", Severity: FindingLow, Message: "附件解析失败"}}},
		},
	}
}

func validCourseReportDataset() map[string]experimentReportItem {
	dataset := validExperimentReportDataset()
	dataset["submission-3"] = experimentReportItem{
		detail:     SubmissionDetail{Submission: Submission{ID: "submission-3", ExperimentID: "experiment-2", Status: "submitted"}, Artifacts: []ArtifactWithExtraction{{Artifact: Artifact{Kind: ArtifactKindReport}, Extraction: ExtractedContent{Status: "succeeded"}}}},
		review:     TeacherReviewDetail{Review: TeacherReview{ID: "review-3", SubmissionID: "submission-3", Status: TeacherReviewStatusPublished, TotalScoreBPS: 7000}, Scores: []TeacherMetricScore{{MetricCode: "docs", FinalScore: 7, MaxScore: 10, WeightBPS: 4000}, {MetricCode: "quality", FinalScore: 14, MaxScore: 20, WeightBPS: 6000}}},
		evaluation: EvaluationResultDetail{Result: EvaluationResult{ID: "evaluation-3"}, Findings: []RuleCheckFinding{{Category: "logic", Severity: FindingHigh, Message: "关键结论缺少证据"}}},
	}
	return dataset
}

type testStore struct {
	root string
}

func (s testStore) Resolve(key string) (string, error) {
	return filepath.Join(s.root, filepath.FromSlash(key)), nil
}

func mustResolve(t *testing.T, store testStore, key string) string {
	t.Helper()
	path, err := store.Resolve(key)
	if err != nil {
		t.Fatal(err)
	}
	return path
}
