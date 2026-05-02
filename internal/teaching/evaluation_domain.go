package teaching

import (
	"encoding/json"
	"time"
)

const (
	InitialEvaluationPromptVersion = "initial-evaluation-v1"
	RuleOnlyMode                   = "rule_only"
	RuleAndLLMMode                 = "rule_and_llm"
)

type EvaluationStatus string

const (
	EvaluationStatusCompleted   EvaluationStatus = "completed"
	EvaluationStatusNeedsReview EvaluationStatus = "needs_review"
	EvaluationStatusFailed      EvaluationStatus = "failed"
)

type EvaluationStepStatus string

const (
	EvaluationStepSucceeded     EvaluationStepStatus = "succeeded"
	EvaluationStepFailed        EvaluationStepStatus = "failed"
	EvaluationStepSkipped       EvaluationStepStatus = "skipped"
	EvaluationStepNotConfigured EvaluationStepStatus = "not_configured"
)

type FindingSeverity string

const (
	FindingInfo     FindingSeverity = "info"
	FindingLow      FindingSeverity = "low"
	FindingMedium   FindingSeverity = "medium"
	FindingHigh     FindingSeverity = "high"
	FindingCritical FindingSeverity = "critical"
)

type MetricScoreSource string

const (
	MetricScoreSourceRule MetricScoreSource = "rule"
	MetricScoreSourceLLM  MetricScoreSource = "llm"
)

type EvaluationContext struct {
	Submission Submission               `json:"submission"`
	Experiment Experiment               `json:"experiment"`
	Metrics    []Metric                 `json:"metrics"`
	Artifacts  []ArtifactWithExtraction `json:"artifacts"`
}

type EvaluationResult struct {
	ID                 string               `json:"id"`
	SubmissionID       string               `json:"submission_id"`
	ExperimentID       string               `json:"experiment_id"`
	RubricVersionID    string               `json:"rubric_version_id"`
	Status             EvaluationStatus     `json:"status"`
	RuleStatus         EvaluationStepStatus `json:"rule_status"`
	LLMStatus          EvaluationStepStatus `json:"llm_status"`
	PromptVersion      string               `json:"prompt_version"`
	EvidenceSnapshot   json.RawMessage      `json:"evidence_snapshot"`
	RuleSummary        json.RawMessage      `json:"rule_summary"`
	LLMSummary         string               `json:"llm_summary,omitempty"`
	NeedsTeacherReview bool                 `json:"needs_teacher_review"`
	Error              string               `json:"error,omitempty"`
	CreatedBy          string               `json:"created_by"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
}

type RuleCheckFinding struct {
	ID                 string          `json:"id"`
	EvaluationResultID string          `json:"evaluation_result_id"`
	Category           string          `json:"category"`
	Severity           FindingSeverity `json:"severity"`
	Message            string          `json:"message"`
	EvidenceRef        string          `json:"evidence_ref,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

type MetricScore struct {
	ID                 string            `json:"id"`
	EvaluationResultID string            `json:"evaluation_result_id"`
	MetricID           string            `json:"metric_id"`
	MetricCode         string            `json:"metric_code"`
	Source             MetricScoreSource `json:"source"`
	SuggestedScore     int               `json:"suggested_score"`
	MaxScore           int               `json:"max_score"`
	ConfidenceBPS      int               `json:"confidence_bps"`
	Rationale          string            `json:"rationale"`
	EvidenceRefs       json.RawMessage   `json:"evidence_refs"`
	CreatedAt          time.Time         `json:"created_at"`
}

type LLMCallLog struct {
	ID                 string          `json:"id"`
	EvaluationResultID string          `json:"evaluation_result_id"`
	Provider           string          `json:"provider"`
	Model              string          `json:"model"`
	PromptVersion      string          `json:"prompt_version"`
	InputHash          string          `json:"input_hash"`
	Output             json.RawMessage `json:"output"`
	Status             string          `json:"status"`
	Error              string          `json:"error,omitempty"`
	LatencyMS          int             `json:"latency_ms"`
	PromptTokens       int             `json:"prompt_tokens"`
	CompletionTokens   int             `json:"completion_tokens"`
	CreatedAt          time.Time       `json:"created_at"`
}

type EvaluationResultDetail struct {
	Result   EvaluationResult   `json:"result"`
	Findings []RuleCheckFinding `json:"findings"`
	Scores   []MetricScore      `json:"scores"`
}

type CreateInitialEvaluationInput struct {
	Mode  string `json:"mode,omitempty"`
	Force bool   `json:"force,omitempty"`
}

type SubmissionSpec struct {
	RequiredArtifacts []string `json:"required_artifacts"`
	RequiredSections  []string `json:"required_sections"`
	RequiredSteps     []string `json:"required_steps"`
	Keywords          []string `json:"keywords"`
}
