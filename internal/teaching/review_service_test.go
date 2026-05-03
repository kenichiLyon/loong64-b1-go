package teaching

import (
	"context"
	"testing"
)

func TestUpsertTeacherReviewRequiresAllMetricsAndComputesTotal(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	evalCtx := validReviewEvaluationContext()
	repo := &fakeRepo{teacherAllowed: true, evaluationContext: evalCtx}
	service := NewService(repo)

	detail, err := service.UpsertTeacherReview(context.Background(), actor, "submission-1", UpsertTeacherReviewInput{
		EvaluationResultID: "evaluation-1",
		TeacherComment:     "good work",
		Scores: []TeacherMetricScoreInput{
			{MetricCode: "quality", FinalScore: 18, Source: "llm", SourceMetricScoreID: "metric-score-1", AdjustmentReason: "verified evidence"},
			{MetricCode: "docs", FinalScore: 8, Source: "manual", Comment: "needs clearer screenshots"},
		},
	}, AuditEntry{})
	if err != nil {
		t.Fatalf("UpsertTeacherReview should succeed: %v", err)
	}
	if detail.Review.Status != TeacherReviewStatusDraft || detail.Review.TotalScoreBPS != 8600 {
		t.Fatalf("unexpected review: %+v", detail.Review)
	}
	if len(detail.Scores) != 2 || detail.Scores[0].TeacherReviewID == "" {
		t.Fatalf("unexpected scores: %+v", detail.Scores)
	}
}

func TestUpsertTeacherReviewRejectsMissingMetric(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{teacherAllowed: true, evaluationContext: validReviewEvaluationContext()})
	_, err = service.UpsertTeacherReview(context.Background(), actor, "submission-1", UpsertTeacherReviewInput{
		Scores: []TeacherMetricScoreInput{{MetricCode: "quality", FinalScore: 18}},
	}, AuditEntry{})
	if ErrorKindOf(err) != KindValidation {
		t.Fatalf("expected validation for incomplete metric set, got %v", err)
	}
}

func TestUpsertTeacherReviewRequiresTeacherAccess(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{teacherAllowed: false, evaluationContext: validReviewEvaluationContext()})
	_, err = service.UpsertTeacherReview(context.Background(), actor, "submission-1", UpsertTeacherReviewInput{
		Scores: []TeacherMetricScoreInput{{MetricCode: "quality", FinalScore: 18}, {MetricCode: "docs", FinalScore: 8}},
	}, AuditEntry{})
	if ErrorKindOf(err) != KindForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestPublishTeacherReviewRequiresConfirmation(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{teacherAllowed: true})
	_, err = service.PublishTeacherReview(context.Background(), actor, "submission-1", PublishTeacherReviewInput{}, AuditEntry{})
	if ErrorKindOf(err) != KindValidation {
		t.Fatalf("expected validation when confirm is false, got %v", err)
	}
}

func TestGetTeacherReviewStudentOnlyPublishedOwnReview(t *testing.T) {
	actor, err := NewActor("student-1", []Role{RoleStudent})
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepo{ownsSubmission: true, teacherReview: TeacherReviewDetail{Review: TeacherReview{ID: "review-1", SubmissionID: "submission-1", Status: TeacherReviewStatusPublished}}}
	service := NewService(repo)
	detail, err := service.GetTeacherReview(context.Background(), actor, "submission-1")
	if err != nil {
		t.Fatalf("student should read own published review: %v", err)
	}
	if detail.Review.ID != "review-1" {
		t.Fatalf("unexpected review: %+v", detail)
	}
}

func validReviewEvaluationContext() EvaluationContext {
	return EvaluationContext{
		Submission: Submission{ID: "submission-1", ExperimentID: "experiment-1", StudentID: "student-1"},
		Experiment: Experiment{ID: "experiment-1", RubricVersionID: "rubric-version-1", Title: "Lab"},
		Metrics: []Metric{
			{ID: "metric-1", Code: "quality", Name: "Code quality", MaxScore: 20, WeightBPS: 6000},
			{ID: "metric-2", Code: "docs", Name: "Documentation", MaxScore: 10, WeightBPS: 4000},
		},
	}
}
