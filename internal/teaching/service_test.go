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
	lastLLMLog                        *LLMCallLog
	teacherReview                     TeacherReviewDetail
	publishedReview                   TeacherReviewDetail
	lastGetTeacherReviewPublishedOnly bool
	reportExports                     map[string]ReportExport
	experimentSummaries               map[string]experimentReportItem
	courseExperiments                 []Experiment
	classes                           []Class
	courses                           []Course
	teacherCourses                    []Course
	rubricTemplates                   []RubricTemplate
	rubricVersions                    []RubricTemplateVersion
	studentExperiments                []Experiment
	studentSubmissions                []Submission
	userCount                         int
	lastPasswordHash                  string
}

...omitted unchanged remainder...
