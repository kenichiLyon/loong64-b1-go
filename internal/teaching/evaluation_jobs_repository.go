package teaching

import (
	"encoding/json"
	"fmt"
	"time"
)

type evaluationJobPayload struct {
	SubmissionID string                       `json:"submission_id"`
	ActorID      string                       `json:"actor_id"`
	ActorRoles   []Role                       `json:"actor_roles,omitempty"`
	Input        CreateInitialEvaluationInput `json:"input"`
	Result       *EvaluationResultDetail      `json:"result,omitempty"`
}

func marshalEvaluationJobPayload(job EvaluationJob) json.RawMessage {
	return mustJSON(evaluationJobPayload{
		SubmissionID: job.SubmissionID,
		ActorID:      job.ActorID,
		ActorRoles:   job.ActorRoles,
		Input:        job.Input,
		Result:       job.Result,
	})
}

func buildEvaluationJobFromPayload(id, status, errorMessage string, payloadRaw []byte, createdAt, updatedAt time.Time, startedAt, finishedAt *time.Time) (EvaluationJob, error) {
	if len(payloadRaw) == 0 {
		payloadRaw = []byte(`{}`)
	}
	var payload evaluationJobPayload
	if err := json.Unmarshal(payloadRaw, &payload); err != nil {
		return EvaluationJob{}, fmt.Errorf("decode evaluation job payload: %w", err)
	}
	job := EvaluationJob{
		ID:           id,
		SubmissionID: payload.SubmissionID,
		ActorID:      payload.ActorID,
		ActorRoles:   append([]Role(nil), payload.ActorRoles...),
		Status:       EvaluationJobStatus(status),
		Input:        payload.Input,
		Result:       payload.Result,
		Error:        errorMessage,
		CreatedAt:    createdAt,
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		UpdatedAt:    updatedAt,
	}
	return cloneEvaluationJob(job), nil
}
