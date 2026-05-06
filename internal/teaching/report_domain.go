package teaching

import (
	"encoding/json"
	"time"
)

const ReportExportJobType = "report_export"

type ReportFormat string

const (
	ReportFormatHTML ReportFormat = "html"
	ReportFormatCSV  ReportFormat = "csv"
	ReportFormatPDF  ReportFormat = "pdf"
)

type ReportExportStatus string

const (
	ReportExportStatusQueued    ReportExportStatus = "queued"
	ReportExportStatusRunning   ReportExportStatus = "running"
	ReportExportStatusSucceeded ReportExportStatus = "succeeded"
	ReportExportStatusFailed    ReportExportStatus = "failed"
)

type ReportType string

const (
	ReportTypeSubmissionReport  ReportType = "submission_report"
	ReportTypeExperimentSummary ReportType = "experiment_summary"
	ReportTypeCourseSummary     ReportType = "course_summary"
)

type ReportScopeType string

const (
	ReportScopeSubmission ReportScopeType = "submission"
	ReportScopeExperiment ReportScopeType = "experiment"
	ReportScopeCourse     ReportScopeType = "course"
)

type SubmissionReport struct {
	Submission  Submission               `json:"submission"`
	Experiment  Experiment               `json:"experiment"`
	Artifacts   []ArtifactWithExtraction `json:"artifacts"`
	Review      TeacherReviewDetail      `json:"review"`
	Evaluation  *EvaluationResultDetail  `json:"evaluation,omitempty"`
	GeneratedAt time.Time                `json:"generated_at"`
}

type ExperimentReportSummary struct {
	ExperimentID          string          `json:"experiment_id"`
	SubmissionCount       int             `json:"submission_count"`
	SubmittedCount        int             `json:"submitted_count"`
	PublishedReviewCount  int             `json:"published_review_count"`
	AverageScoreBPS       int             `json:"average_score_bps"`
	MinScoreBPS           int             `json:"min_score_bps"`
	MaxScoreBPS           int             `json:"max_score_bps"`
	ScoreBuckets          map[string]int  `json:"score_buckets"`
	SubmissionStatusCount map[string]int  `json:"submission_status_count"`
	ArtifactStatusCount   map[string]int  `json:"artifact_status_count"`
	MetricAverages        []MetricAverage `json:"metric_averages"`
	FindingCounts         []FindingCount  `json:"finding_counts"`
	GeneratedAt           time.Time       `json:"generated_at"`
	scoreSumBPS           int
}

type CourseReportSummary struct {
	CourseID              string                    `json:"course_id"`
	ExperimentCount       int                       `json:"experiment_count"`
	SubmissionCount       int                       `json:"submission_count"`
	SubmittedCount        int                       `json:"submitted_count"`
	PublishedReviewCount  int                       `json:"published_review_count"`
	AverageScoreBPS       int                       `json:"average_score_bps"`
	MinScoreBPS           int                       `json:"min_score_bps"`
	MaxScoreBPS           int                       `json:"max_score_bps"`
	ScoreBuckets          map[string]int            `json:"score_buckets"`
	SubmissionStatusCount map[string]int            `json:"submission_status_count"`
	ArtifactStatusCount   map[string]int            `json:"artifact_status_count"`
	MetricAverages        []MetricAverage           `json:"metric_averages"`
	FindingCounts         []FindingCount            `json:"finding_counts"`
	Experiments           []ExperimentReportSummary `json:"experiments"`
	GeneratedAt           time.Time                 `json:"generated_at"`
}

type MetricAverage struct {
	MetricCode        string `json:"metric_code"`
	AverageScore      int    `json:"average_score"`
	AveragePercentBPS int    `json:"average_percent_bps"`
	MaxScore          int    `json:"max_score"`
	Count             int    `json:"count"`
}

type FindingCount struct {
	Category string          `json:"category"`
	Severity FindingSeverity `json:"severity"`
	Count    int             `json:"count"`
}

type ReportExport struct {
	ID          string             `json:"id"`
	ReportType  ReportType         `json:"report_type"`
	ScopeType   ReportScopeType    `json:"scope_type"`
	ScopeID     string             `json:"scope_id"`
	Format      ReportFormat       `json:"format"`
	Status      ReportExportStatus `json:"status"`
	StorageKey  string             `json:"storage_key,omitempty"`
	SHA256Hex   string             `json:"sha256_hex,omitempty"`
	ByteSize    int64              `json:"byte_size"`
	FilterJSON  json.RawMessage    `json:"filter_json"`
	Error       string             `json:"error,omitempty"`
	RequestedBy string             `json:"requested_by"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	CompletedAt *time.Time         `json:"completed_at,omitempty"`
}

type CreateReportExportInput struct {
	Format  ReportFormat    `json:"format"`
	Filters json.RawMessage `json:"filters,omitempty"`
}

type ReportExportFile struct {
	Export      ReportExport
	Path        string
	ContentType string
	FileName    string
}
