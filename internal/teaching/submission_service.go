package teaching

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

func (s *Service) CreateSubmission(ctx context.Context, actor Actor, experimentID string, _ CreateSubmissionInput, audit AuditEntry) (Submission, error) {
	if err := s.ready(); err != nil {
		return Submission{}, err
	}
	if err := actor.Require(RoleStudent); err != nil {
		return Submission{}, err
	}
	experimentID = strings.TrimSpace(experimentID)
	access, err := s.repo.ExperimentSubmissionAccess(ctx, experimentID, actor.ID)
	if err != nil {
		return Submission{}, err
	}
	if access.Status != "published" {
		return Submission{}, validationError("experiment is not open for submission")
	}
	if !access.Enrolled {
		return Submission{}, forbiddenError("student is not enrolled in the experiment course")
	}
	if access.DueAt != nil && time.Now().UTC().After(access.DueAt.UTC()) {
		return Submission{}, validationError("experiment due_at has passed")
	}
	submission := Submission{
		ID:           NewID("sub"),
		ExperimentID: experimentID,
		StudentID:    actor.ID,
		Status:       "draft",
		AttemptNo:    1,
	}
	audit.Action = "submission.create"
	audit.ActorID = actor.ID
	audit.TargetType = "submission"
	audit.TargetID = submission.ID
	return s.repo.CreateSubmission(ctx, submission, audit)
}

func (s *Service) UploadArtifact(ctx context.Context, actor Actor, submissionID string, input ArtifactUploadInput, audit AuditEntry) (ArtifactWithExtraction, error) {
	if err := s.ready(); err != nil {
		return ArtifactWithExtraction{}, err
	}
	if err := actor.Require(RoleStudent); err != nil {
		return ArtifactWithExtraction{}, err
	}
	submissionID = strings.TrimSpace(submissionID)
	if err := s.requireSubmissionOwner(ctx, submissionID, actor.ID); err != nil {
		return ArtifactWithExtraction{}, err
	}
	if err := s.requireArtifactCapacity(ctx, submissionID); err != nil {
		return ArtifactWithExtraction{}, err
	}
	artifactID := NewID("art")
	stored, err := s.storeUploadedArtifact(input, artifactID, submissionID)
	if err != nil {
		return ArtifactWithExtraction{}, err
	}
	stored, err = s.maybeParseWithArtifactParser(ctx, artifactID, stored)
	if err != nil {
		cleanupStoredArtifact(s.store, stored.StorageKey)
		return ArtifactWithExtraction{}, err
	}
	artifact := Artifact{
		ID:           artifactID,
		SubmissionID: submissionID,
		Kind:         stored.Kind,
		OriginalName: strings.TrimSpace(input.FileName),
		ContentType:  stored.ContentType,
		ByteSize:     stored.ByteSize,
		SHA256Hex:    stored.SHA256Hex,
		StorageKey:   stored.StorageKey,
		Status:       "stored",
		Metadata:     stored.Metadata,
		CreatedBy:    actor.ID,
	}
	extraction := ExtractedContent{
		ID:          NewID("ext"),
		ArtifactID:  artifactID,
		Status:      "queued",
		TextExcerpt: stored.TextExcerpt,
		Metadata:    mustJSON(map[string]any{"source": "upload", "mode": "metadata_ready"}),
	}
	job := &QueuedJob{
		ID:   NewID("job"),
		Type: ParseArtifactJobType,
		Payload: mustJSON(map[string]any{
			"artifact_id":   artifact.ID,
			"submission_id": submissionID,
			"storage_key":   stored.StorageKey,
			"kind":          stored.Kind,
		}),
	}
	audit.Action = "artifact.upload"
	audit.ActorID = actor.ID
	audit.TargetType = "artifact"
	audit.TargetID = artifact.ID
	audit.Detail = mustJSON(map[string]any{
		"submission_id": submissionID,
		"original_name": artifact.OriginalName,
		"byte_size":     artifact.ByteSize,
		"sha256_hex":    artifact.SHA256Hex,
	})
	created, err := s.repo.CreateArtifact(ctx, artifact, extraction, job, audit)
	if err != nil {
		cleanupStoredArtifact(s.store, stored.StorageKey)
		return ArtifactWithExtraction{}, err
	}
	return created, nil
}

func (s *Service) CreateGitLinkArtifact(ctx context.Context, actor Actor, submissionID string, input CreateGitLinkInput, audit AuditEntry) (ArtifactWithExtraction, error) {
	if err := s.ready(); err != nil {
		return ArtifactWithExtraction{}, err
	}
	if err := actor.Require(RoleStudent); err != nil {
		return ArtifactWithExtraction{}, err
	}
	submissionID = strings.TrimSpace(submissionID)
	if err := s.requireSubmissionOwner(ctx, submissionID, actor.ID); err != nil {
		return ArtifactWithExtraction{}, err
	}
	if err := s.requireArtifactCapacity(ctx, submissionID); err != nil {
		return ArtifactWithExtraction{}, err
	}
	metadata, err := validateGitLink(input)
	if err != nil {
		return ArtifactWithExtraction{}, err
	}
	artifactID := NewID("art")
	artifact := Artifact{
		ID:           artifactID,
		SubmissionID: submissionID,
		Kind:         ArtifactKindGitLink,
		OriginalName: "git-link",
		SourceURL:    strings.TrimSpace(input.URL),
		Status:       "stored",
		Metadata:     metadata,
		CreatedBy:    actor.ID,
	}
	extraction := ExtractedContent{
		ID:          NewID("ext"),
		ArtifactID:  artifactID,
		Status:      "succeeded",
		TextExcerpt: "Git link metadata recorded. Repository fetching and code inspection are deferred to a sandboxed worker.",
		Metadata:    mustJSON(map[string]any{"source": "git_link", "mode": "metadata_only"}),
	}
	audit.Action = "artifact.link_create"
	audit.ActorID = actor.ID
	audit.TargetType = "artifact"
	audit.TargetID = artifact.ID
	audit.Detail = mustJSON(map[string]any{"submission_id": submissionID, "url": artifact.SourceURL})
	return s.repo.CreateArtifact(ctx, artifact, extraction, nil, audit)
}

func (s *Service) ListSubmissionsForExperiment(ctx context.Context, actor Actor, experimentID string, limit int) ([]Submission, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if err := s.requireTeacherExperimentAccess(ctx, actor, strings.TrimSpace(experimentID)); err != nil {
		return nil, err
	}
	return s.repo.ListSubmissionsForExperiment(ctx, strings.TrimSpace(experimentID), clampLimit(limit))
}

func (s *Service) ListStudentSubmissions(ctx context.Context, actor Actor, experimentID string, limit int) ([]Submission, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if err := actor.Require(RoleStudent); err != nil {
		return nil, err
	}
	return s.repo.ListSubmissionsForStudent(ctx, actor.ID, strings.TrimSpace(experimentID), clampLimit(limit))
}

func (s *Service) GetSubmissionDetail(ctx context.Context, actor Actor, submissionID string) (SubmissionDetail, error) {
	if err := s.ready(); err != nil {
		return SubmissionDetail{}, err
	}
	submissionID = strings.TrimSpace(submissionID)
	if !actor.Has(RoleAdmin) && !actor.Has(RoleTeacher) && !actor.Has(RoleStudent) {
		return SubmissionDetail{}, forbiddenError("teacher, admin or owning student role is required")
	}
	if !actor.Has(RoleAdmin) {
		if actor.Has(RoleStudent) {
			owns, err := s.repo.StudentOwnsSubmission(ctx, submissionID, actor.ID)
			if err != nil {
				return SubmissionDetail{}, err
			}
			if owns {
				return s.repo.GetSubmissionDetail(ctx, submissionID)
			}
		}
		if !actor.Has(RoleTeacher) {
			return SubmissionDetail{}, forbiddenError("student can only view own submission")
		}
		courseID, err := s.repo.SubmissionCourseID(ctx, submissionID)
		if err != nil {
			return SubmissionDetail{}, err
		}
		allowed, err := s.repo.TeacherCanEditCourse(ctx, courseID, actor.ID)
		if err != nil {
			return SubmissionDetail{}, err
		}
		if !allowed {
			return SubmissionDetail{}, forbiddenError("teacher is not assigned to this course")
		}
	}
	return s.repo.GetSubmissionDetail(ctx, submissionID)
}

func (s *Service) requireSubmissionOwner(ctx context.Context, submissionID, studentID string) error {
	owns, err := s.repo.StudentOwnsSubmission(ctx, submissionID, studentID)
	if err != nil {
		return err
	}
	if !owns {
		return forbiddenError("student can only operate on own submission")
	}
	return nil
}

func (s *Service) requireArtifactCapacity(ctx context.Context, submissionID string) error {
	count, err := s.repo.SubmissionArtifactCount(ctx, submissionID)
	if err != nil {
		return err
	}
	if count >= s.maxArtifactsPerSubmission {
		return validationError(fmt.Sprintf("submission cannot contain more than %d artifacts", s.maxArtifactsPerSubmission))
	}
	return nil
}

func (s *Service) requireTeacherExperimentAccess(ctx context.Context, actor Actor, experimentID string) error {
	if actor.Has(RoleAdmin) {
		return nil
	}
	if err := actor.Require(RoleTeacher); err != nil {
		return err
	}
	courseID, err := s.repo.ExperimentCourseID(ctx, experimentID)
	if err != nil {
		return err
	}
	allowed, err := s.repo.TeacherCanEditCourse(ctx, courseID, actor.ID)
	if err != nil {
		return err
	}
	if !allowed {
		return forbiddenError("teacher is not assigned to this course")
	}
	return nil
}

func parseArtifactKindField(formValue string) string {
	return strings.ToLower(strings.TrimSpace(formValue))
}

func cleanupStoredArtifact(store ArtifactStore, key string) {
	if store == nil || key == "" {
		return
	}
	path, err := store.Resolve(key)
	if err != nil {
		return
	}
	_ = os.Remove(path)
}
