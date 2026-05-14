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

func TestCreateInitialEvaluationUsesAIGatewayEvaluatorWhenConfigured(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepo{teacherAllowed: true, evaluationContext: validEvaluationContext()}
	service := NewService(
		repo,
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
				RawModelMeta: map[string]any{
					"engine": "gateway-stub",
					"retrieval_context": map[string]any{
						"hit_count": 2,
						"queries":   []string{"Code quality"},
						"citations": []map[string]any{{"evidence_ref": "artifact:artifact-1"}},
					},
				},
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
	if repo.lastLLMLog == nil {
		t.Fatal("expected llm log to be persisted")
	}
	var rawMeta map[string]any
	if err := json.Unmarshal(repo.lastLLMLog.Output, &rawMeta); err != nil {
		t.Fatalf("llm log output should be JSON: %v", err)
	}
	retrievalContext, ok := rawMeta["retrieval_context"].(map[string]any)
	if !ok {
		t.Fatalf("expected retrieval_context in llm log output: %+v", rawMeta)
	}
	if retrievalContext["hit_count"] != float64(2) {
		t.Fatalf("unexpected retrieval_context: %+v", retrievalContext)
	}
}

func TestGetLatestEvaluationIncludesLLMCallLog(t *testing.T) {
	actor, err := NewActor("teacher-1", []Role{RoleTeacher})
	if err != nil {
		t.Fatal(err)
	}
	repo := &fakeRepo{
		teacherAllowed: true,
		latestEvaluation: EvaluationResultDetail{
			Result: EvaluationResult{ID: "evaluation-1", SubmissionID: "submission-1"},
		},
		lastLLMLog: &LLMCallLog{
			ID:                 "llg-1",
			EvaluationResultID: "evaluation-1",
			Provider:           "python-ai-gateway",
			Output: mustJSON(map[string]any{
				"retrieval_context": map[string]any{"hit_count": 2},
			}),
		},
	}
	service := NewService(repo)
	detail, err := service.GetLatestEvaluation(context.Background(), actor, "submission-1")
	if err != nil {
		t.Fatalf("GetLatestEvaluation should succeed: %v", err)
	}
	if detail.Log == nil || detail.Log.ID != "llg-1" {
		t.Fatalf("expected llm log on latest evaluation: %+v", detail)
	}
}

...omitted unchanged remainder...
