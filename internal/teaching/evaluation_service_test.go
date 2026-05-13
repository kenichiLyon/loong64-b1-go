package teaching

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/kenichiLyon/loong64-b1-go/internal/aigateway"
	"github.com/kenichiLyon/loong64-b1-go/internal/llm"
)

func TestCreateInitialEvaluationRuleOnly(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepo{teacherAllowed: true, evaluationContext: validEvaluationContext()}
	service := NewService(repo)

	detail, err := service.CreateInitialEvaluation(context.Background(), actor, "submission-1", CreateInitialEvaluationInput{Mode: RuleOnlyMode}, AuditEntry{})
	if err != nil {
		t.Fatalf("CreateInitialEvaluation should succeed: %v", err)
	}
	if detail.Result.SubmissionID != "submission-1" || detail.Result.RuleStatus != EvaluationStepSucceeded || detail.Result.LLMStatus != EvaluationStepSkipped {
		t.Fatalf("unexpected result: %+v", detail.Result)
	}
	if len(detail.Scores) != 1 || detail.Scores[0].Source != MetricScoreSourceRule {
		t.Fatalf("expected one rule score: %+v", detail.Scores)
	}
	if repo.createdEvaluation.Result.ID == "" {
		t.Fatal("repository should receive created evaluation")
	}
}

func TestCreateInitialEvaluationRequiresTeacherAccess(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{teacherAllowed: false, evaluationContext: validEvaluationContext()})
	_, err = service.CreateInitialEvaluation(context.Background(), actor, "submission-1", CreateInitialEvaluationInput{}, AuditEntry{})
	if ErrorKindOf(err) != KindForbidden {
		t.Fatalf("expected forbidden for unassigned teacher, got %v", err)
	}
}

func TestCreateInitialEvaluationWithLLMValidatesOutput(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{teacherAllowed: true, evaluationContext: validEvaluationContext()}, WithLLMClient(fakeLLMCompleter{
		content: `{"summary":"looks acceptable","metrics":[{"metric_code":"quality","suggested_score":18,"confidence_bps":8000,"rationale":"evidence matches rubric","evidence_refs":["artifact:artifact-1"]}],"risks":[]}`,
	}))
	detail, err := service.CreateInitialEvaluation(context.Background(), actor, "submission-1", CreateInitialEvaluationInput{Mode: RuleAndLLMMode}, AuditEntry{})
	if err != nil {
		t.Fatalf("CreateInitialEvaluation with llm should succeed: %v", err)
	}
	if detail.Result.LLMStatus != EvaluationStepSucceeded || detail.Result.LLMSummary == "" {
		t.Fatalf("unexpected llm result: %+v", detail.Result)
	}
	if !hasScoreSource(detail.Scores, MetricScoreSourceLLM) {
		t.Fatalf("expected llm score: %+v", detail.Scores)
	}
}

func TestCreateInitialEvaluationUsesAIGatewayEvaluatorWhenConfigured(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(
		&fakeRepo{teacherAllowed: true, evaluationContext: validEvaluationContext()},
		WithSubmissionEvaluator(fakeSubmissionEvaluator{
			response: aigateway.EvaluateSubmissionResponse{
				Summary: "gateway summary",
				MetricScores: []map[string]any{{
					"metric_code":     "quality",
					"suggested_score": 17,
					"confidence_bps":  7200,
					"rationale":       "gateway evidence",
					"evidence_refs":   []string{"artifact:artifact-1"},
				}},
				RawModelMeta: map[string]any{"engine": "gateway-stub"},
			},
		}),
	)
	detail, err := service.CreateInitialEvaluation(context.Background(), actor, "submission-1", CreateInitialEvaluationInput{Mode: RuleAndLLMMode}, AuditEntry{})
	if err != nil {
		t.Fatalf("CreateInitialEvaluation with ai gateway should succeed: %v", err)
	}
	if detail.Result.LLMStatus != EvaluationStepSucceeded || detail.Result.LLMSummary != "gateway summary" {
		t.Fatalf("unexpected gateway result: %+v", detail.Result)
	}
	if !hasScoreSource(detail.Scores, MetricScoreSourceLLM) {
		t.Fatalf("expected ai gateway score: %+v", detail.Scores)
	}
}

func TestCreateInitialEvaluationAcceptsGranularEvidenceRefs(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	ctx := validEvaluationContext()
	ctx.Artifacts[0].Artifact.Metadata = mustJSON(map[string]any{
		"sections": []map[string]any{{
			"title":   "Overview",
			"content": "implemented api and tests",
		}},
	})
	service := NewService(
		&fakeRepo{teacherAllowed: true, evaluationContext: ctx},
		WithSubmissionEvaluator(fakeSubmissionEvaluator{
			response: aigateway.EvaluateSubmissionResponse{
				Summary: "section-backed summary",
				MetricScores: []map[string]any{{
					"metric_code":     "quality",
					"suggested_score": 19,
					"confidence_bps":  8100,
					"rationale":       "section evidence",
					"evidence_refs":   []string{"artifact:artifact-1#section:1"},
				}},
				RawModelMeta: map[string]any{"engine": "gateway-stub"},
			},
		}),
	)
	detail, err := service.CreateInitialEvaluation(context.Background(), actor, "submission-1", CreateInitialEvaluationInput{Mode: RuleAndLLMMode}, AuditEntry{})
	if err != nil {
		t.Fatalf("CreateInitialEvaluation should accept granular evidence refs: %v", err)
	}
	if detail.Result.LLMSummary != "section-backed summary" {
		t.Fatalf("unexpected detail: %+v", detail.Result)
	}
	var refs []string
	if err := json.Unmarshal(detail.Scores[len(detail.Scores)-1].EvidenceRefs, &refs); err != nil {
		t.Fatalf("evidence refs should be json array: %v", err)
	}
	if len(refs) != 1 || refs[0] != "artifact:artifact-1#section:1" {
		t.Fatalf("unexpected granular refs: %+v", refs)
	}
}

func TestCreateInitialEvaluationFallsBackToGoLLMWhenAIGatewayFails(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(
		&fakeRepo{teacherAllowed: true, evaluationContext: validEvaluationContext()},
		WithSubmissionEvaluator(fakeSubmissionEvaluator{err: errors.New("gateway unavailable")}),
		WithLLMClient(fakeLLMCompleter{
			content: `{"summary":"fallback llm","metrics":[{"metric_code":"quality","suggested_score":18,"confidence_bps":8000,"rationale":"fallback used","evidence_refs":["artifact:artifact-1"]}],"risks":[]}`,
		}),
	)
	detail, err := service.CreateInitialEvaluation(context.Background(), actor, "submission-1", CreateInitialEvaluationInput{Mode: RuleAndLLMMode}, AuditEntry{})
	if err != nil {
		t.Fatalf("CreateInitialEvaluation should fall back to Go llm: %v", err)
	}
	if detail.Result.LLMStatus != EvaluationStepSucceeded || detail.Result.LLMSummary != "fallback llm" {
		t.Fatalf("unexpected fallback result: %+v", detail.Result)
	}
	if !hasScoreSource(detail.Scores, MetricScoreSourceLLM) {
		t.Fatalf("expected fallback llm score: %+v", detail.Scores)
	}
}

func TestCreateInitialEvaluationMarksMalformedLLMForReview(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(&fakeRepo{teacherAllowed: true, evaluationContext: validEvaluationContext()}, WithLLMClient(fakeLLMCompleter{
		content: `{"summary":"bad","metrics":[{"metric_code":"quality","suggested_score":99,"confidence_bps":8000,"rationale":"too high","evidence_refs":["artifact:artifact-1"]}]}`,
	}))
	detail, err := service.CreateInitialEvaluation(context.Background(), actor, "submission-1", CreateInitialEvaluationInput{Mode: RuleAndLLMMode}, AuditEntry{})
	if err != nil {
		t.Fatalf("malformed llm output should be persisted for review, got %v", err)
	}
	if detail.Result.Status != EvaluationStatusNeedsReview || detail.Result.LLMStatus != EvaluationStepFailed {
		t.Fatalf("unexpected failed llm status: %+v", detail.Result)
	}
	if hasScoreSource(detail.Scores, MetricScoreSourceLLM) {
		t.Fatalf("invalid llm scores should not be stored: %+v", detail.Scores)
	}
}

func validEvaluationContext() EvaluationContext {
	return EvaluationContext{
		Submission: Submission{ID: "submission-1", ExperimentID: "experiment-1", StudentID: "student-1"},
		Experiment: Experiment{ID: "experiment-1", RubricVersionID: "rubric-version-1", Title: "Lab"},
		Metrics:    []Metric{{ID: "metric-1", Code: "quality", Name: "Code quality", MaxScore: 20, WeightBPS: 10000}},
		Artifacts: []ArtifactWithExtraction{{
			Artifact:   Artifact{ID: "artifact-1", SubmissionID: "submission-1", Kind: ArtifactKindReport, OriginalName: "report.txt", Status: "stored"},
			Extraction: ExtractedContent{ID: "extraction-1", ArtifactID: "artifact-1", Status: "succeeded", TextExcerpt: "implemented api and tests"},
		}},
	}
}

type fakeLLMCompleter struct {
	content string
	err     error
}

func (f fakeLLMCompleter) CompleteJSON(context.Context, llm.CompletionRequest) (llm.CompletionResponse, error) {
	return llm.CompletionResponse{Model: "mock-model", Content: f.content, PromptTokens: 10, CompletionTokens: 20, Latency: time.Millisecond}, f.err
}

type fakeSubmissionEvaluator struct {
	response aigateway.EvaluateSubmissionResponse
	err      error
}

func (f fakeSubmissionEvaluator) EvaluateSubmission(context.Context, aigateway.EvaluateSubmissionRequest) (aigateway.EvaluateSubmissionResponse, error) {
	return f.response, f.err
}

func hasScoreSource(scores []MetricScore, source MetricScoreSource) bool {
	for _, score := range scores {
		if score.Source == source {
			return true
		}
	}
	return false
}
