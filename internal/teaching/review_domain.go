package teaching

import "time"

const ManualScoreSource = "manual"

type TeacherReviewStatus string

const (
	TeacherReviewStatusDraft     TeacherReviewStatus = "draft"
	TeacherReviewStatusPublished TeacherReviewStatus = "published"
)

type TeacherReview struct {
	ID                 string              `json:"id"`
	SubmissionID       string              `json:"submission_id"`
	EvaluationResultID string              `json:"evaluation_result_id,omitempty"`
	ExperimentID       string              `json:"experiment_id"`
	RubricVersionID    string              `json:"rubric_version_id"`
	Status             TeacherReviewStatus `json:"status"`
	TotalScoreBPS      int                 `json:"total_score_bps"`
	TeacherComment     string              `json:"teacher_comment,omitempty"`
	CreatedBy          string              `json:"created_by"`
	UpdatedBy          string              `json:"updated_by"`
	PublishedBy        string              `json:"published_by,omitempty"`
	PublishedAt        *time.Time          `json:"published_at,omitempty"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
}

type TeacherMetricScore struct {
	ID                  string    `json:"id"`
	TeacherReviewID     string    `json:"teacher_review_id"`
	MetricID            string    `json:"metric_id"`
	MetricCode          string    `json:"metric_code"`
	FinalScore          int       `json:"final_score"`
	MaxScore            int       `json:"max_score"`
	WeightBPS           int       `json:"weight_bps"`
	Source              string    `json:"source"`
	SourceMetricScoreID string    `json:"source_metric_score_id,omitempty"`
	Comment             string    `json:"comment,omitempty"`
	AdjustmentReason    string    `json:"adjustment_reason,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type TeacherReviewDetail struct {
	Review TeacherReview           `json:"review"`
	Scores []TeacherMetricScore    `json:"scores"`
	AI     *EvaluationResultDetail `json:"ai,omitempty"`
}

type TeacherMetricScoreInput struct {
	MetricID            string `json:"metric_id,omitempty"`
	MetricCode          string `json:"metric_code,omitempty"`
	FinalScore          int    `json:"final_score"`
	Source              string `json:"source,omitempty"`
	SourceMetricScoreID string `json:"source_metric_score_id,omitempty"`
	Comment             string `json:"comment,omitempty"`
	AdjustmentReason    string `json:"adjustment_reason,omitempty"`
}

type UpsertTeacherReviewInput struct {
	EvaluationResultID string                    `json:"evaluation_result_id,omitempty"`
	TeacherComment     string                    `json:"teacher_comment,omitempty"`
	Scores             []TeacherMetricScoreInput `json:"scores"`
}

type PublishTeacherReviewInput struct {
	Confirm bool `json:"confirm"`
}
