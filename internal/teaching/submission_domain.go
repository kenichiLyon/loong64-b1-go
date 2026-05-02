package teaching

import (
	"context"
	"encoding/json"
	"io"
	"time"
)

const (
	DefaultMaxUploadBytes            int64 = 50 * 1024 * 1024
	DefaultMaxArtifactsPerSubmission       = 20
	ParseArtifactJobType                   = "submission_artifact_parse"
)

type ArtifactKind string

const (
	ArtifactKindDocument    ArtifactKind = "document"
	ArtifactKindReport      ArtifactKind = "report"
	ArtifactKindScreenshot  ArtifactKind = "screenshot"
	ArtifactKindCodeArchive ArtifactKind = "code_archive"
	ArtifactKindGitLink     ArtifactKind = "git_link"
	ArtifactKindOther       ArtifactKind = "other"
)

type Submission struct {
	ID           string     `json:"id"`
	ExperimentID string     `json:"experiment_id"`
	StudentID    string     `json:"student_id"`
	Status       string     `json:"status"`
	AttemptNo    int        `json:"attempt_no"`
	SubmittedAt  *time.Time `json:"submitted_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Artifact struct {
	ID           string          `json:"id"`
	SubmissionID string          `json:"submission_id"`
	Kind         ArtifactKind    `json:"kind"`
	OriginalName string          `json:"original_name"`
	ContentType  string          `json:"content_type,omitempty"`
	ByteSize     int64           `json:"byte_size"`
	SHA256Hex    string          `json:"sha256_hex,omitempty"`
	StorageKey   string          `json:"storage_key,omitempty"`
	SourceURL    string          `json:"source_url,omitempty"`
	Status       string          `json:"status"`
	Metadata     json.RawMessage `json:"metadata"`
	CreatedBy    string          `json:"created_by"`
	CreatedAt    time.Time       `json:"created_at"`
}

type ExtractedContent struct {
	ID          string          `json:"id"`
	ArtifactID  string          `json:"artifact_id"`
	Status      string          `json:"status"`
	TextExcerpt string          `json:"text_excerpt,omitempty"`
	Metadata    json.RawMessage `json:"metadata"`
	Error       string          `json:"error,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type ArtifactWithExtraction struct {
	Artifact   Artifact         `json:"artifact"`
	Extraction ExtractedContent `json:"extraction"`
	JobID      string           `json:"job_id,omitempty"`
}

type SubmissionDetail struct {
	Submission Submission               `json:"submission"`
	Artifacts  []ArtifactWithExtraction `json:"artifacts"`
}

type ExperimentSubmissionAccess struct {
	CourseID string
	Status   string
	DueAt    *time.Time
	Enrolled bool
}

type QueuedJob struct {
	ID      string
	Type    string
	Payload json.RawMessage
}

type CreateSubmissionInput struct {
	Note string `json:"note,omitempty"`
}

type ArtifactUploadInput struct {
	FileName     string
	ContentType  string
	DeclaredKind string
	Reader       io.Reader
}

type CreateGitLinkInput struct {
	URL       string `json:"url"`
	CommitSHA string `json:"commit_sha,omitempty"`
	Note      string `json:"note,omitempty"`
}

type SubmissionRepository interface {
	ExperimentSubmissionAccess(context.Context, string, string) (ExperimentSubmissionAccess, error)
	CreateSubmission(context.Context, Submission, AuditEntry) (Submission, error)
	StudentOwnsSubmission(context.Context, string, string) (bool, error)
	SubmissionCourseID(context.Context, string) (string, error)
	SubmissionArtifactCount(context.Context, string) (int, error)
	CreateArtifact(context.Context, Artifact, ExtractedContent, *QueuedJob, AuditEntry) (ArtifactWithExtraction, error)
	ListSubmissionsForExperiment(context.Context, string, int) ([]Submission, error)
	GetSubmissionDetail(context.Context, string) (SubmissionDetail, error)
}
