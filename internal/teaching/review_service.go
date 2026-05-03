package teaching

import (
	"context"
	"strings"
)

func (s *Service) UpsertTeacherReview(ctx context.Context, actor Actor, submissionID string, input UpsertTeacherReviewInput, audit AuditEntry) (TeacherReviewDetail, error) {
	if err := s.ready(); err != nil {
		return TeacherReviewDetail{}, err
	}
	submissionID = strings.TrimSpace(submissionID)
	if err := s.requireTeacherSubmissionAccess(ctx, actor, submissionID); err != nil {
		return TeacherReviewDetail{}, err
	}
	evalCtx, err := s.repo.GetEvaluationContext(ctx, submissionID)
	if err != nil {
		return TeacherReviewDetail{}, err
	}
	if len(evalCtx.Metrics) == 0 {
		return TeacherReviewDetail{}, validationError("submission experiment has no rubric metrics")
	}
	evaluationResultID := strings.TrimSpace(input.EvaluationResultID)
	if evaluationResultID != "" {
		linkedSubmissionID, err := s.repo.EvaluationResultSubmissionID(ctx, evaluationResultID)
		if err != nil {
			return TeacherReviewDetail{}, err
		}
		if linkedSubmissionID != submissionID {
			return TeacherReviewDetail{}, validationError("evaluation_result_id must belong to the submission")
		}
	}
	scores, total, err := buildTeacherMetricScores(NewID("trv"), evalCtx.Metrics, input.Scores)
	if err != nil {
		return TeacherReviewDetail{}, err
	}
	review := TeacherReview{
		ID:                 scores[0].TeacherReviewID,
		SubmissionID:       evalCtx.Submission.ID,
		EvaluationResultID: evaluationResultID,
		ExperimentID:       evalCtx.Experiment.ID,
		RubricVersionID:    evalCtx.Experiment.RubricVersionID,
		Status:             TeacherReviewStatusDraft,
		TotalScoreBPS:      total,
		TeacherComment:     strings.TrimSpace(input.TeacherComment),
		CreatedBy:          actor.ID,
		UpdatedBy:          actor.ID,
	}
	audit.Action = "teacher_review.upsert"
	audit.ActorID = actor.ID
	audit.TargetType = "submission"
	audit.TargetID = submissionID
	audit.Detail = mustJSON(map[string]any{"review_id": review.ID, "evaluation_result_id": review.EvaluationResultID})
	return s.repo.UpsertTeacherReview(ctx, review, scores, audit)
}

func (s *Service) PublishTeacherReview(ctx context.Context, actor Actor, submissionID string, input PublishTeacherReviewInput, audit AuditEntry) (TeacherReviewDetail, error) {
	if err := s.ready(); err != nil {
		return TeacherReviewDetail{}, err
	}
	if !input.Confirm {
		return TeacherReviewDetail{}, validationError("confirm must be true to publish teacher review")
	}
	submissionID = strings.TrimSpace(submissionID)
	if err := s.requireTeacherSubmissionAccess(ctx, actor, submissionID); err != nil {
		return TeacherReviewDetail{}, err
	}
	audit.Action = "teacher_review.publish"
	audit.ActorID = actor.ID
	audit.TargetType = "submission"
	audit.TargetID = submissionID
	return s.repo.PublishTeacherReview(ctx, submissionID, actor.ID, audit)
}

func (s *Service) GetTeacherReview(ctx context.Context, actor Actor, submissionID string) (TeacherReviewDetail, error) {
	if err := s.ready(); err != nil {
		return TeacherReviewDetail{}, err
	}
	submissionID = strings.TrimSpace(submissionID)
	if actor.Has(RoleStudent) && !actor.Has(RoleTeacher) && !actor.Has(RoleAdmin) {
		owns, err := s.repo.StudentOwnsSubmission(ctx, submissionID, actor.ID)
		if err != nil {
			return TeacherReviewDetail{}, err
		}
		if !owns {
			return TeacherReviewDetail{}, forbiddenError("student can only view own published review")
		}
		return s.repo.GetTeacherReview(ctx, submissionID, true)
	}
	if err := s.requireTeacherSubmissionAccess(ctx, actor, submissionID); err != nil {
		return TeacherReviewDetail{}, err
	}
	return s.repo.GetTeacherReview(ctx, submissionID, false)
}

func buildTeacherMetricScores(reviewID string, metrics []Metric, inputs []TeacherMetricScoreInput) ([]TeacherMetricScore, int, error) {
	if len(inputs) == 0 {
		return nil, 0, validationError("at least one metric score is required")
	}
	byID := make(map[string]Metric, len(metrics))
	byCode := make(map[string]Metric, len(metrics))
	for _, metric := range metrics {
		byID[metric.ID] = metric
		byCode[metric.Code] = metric
	}
	seen := make(map[string]struct{}, len(inputs))
	scores := make([]TeacherMetricScore, 0, len(inputs))
	weightedNumerator := 0
	weightTotal := 0
	for _, input := range inputs {
		metric, err := resolveMetricInput(input, byID, byCode)
		if err != nil {
			return nil, 0, err
		}
		if _, ok := seen[metric.ID]; ok {
			return nil, 0, validationError("metric score must not be duplicated")
		}
		seen[metric.ID] = struct{}{}
		if input.FinalScore < 0 || input.FinalScore > metric.MaxScore {
			return nil, 0, validationError("final_score must be within metric max_score")
		}
		source := normalizeReviewScoreSource(input.Source)
		if source != ManualScoreSource && source != string(MetricScoreSourceRule) && source != string(MetricScoreSourceLLM) {
			return nil, 0, validationError("invalid teacher metric score source")
		}
		scores = append(scores, TeacherMetricScore{
			ID:                  NewID("tms"),
			TeacherReviewID:     reviewID,
			MetricID:            metric.ID,
			MetricCode:          metric.Code,
			FinalScore:          input.FinalScore,
			MaxScore:            metric.MaxScore,
			WeightBPS:           metric.WeightBPS,
			Source:              source,
			SourceMetricScoreID: strings.TrimSpace(input.SourceMetricScoreID),
			Comment:             strings.TrimSpace(input.Comment),
			AdjustmentReason:    strings.TrimSpace(input.AdjustmentReason),
		})
		weightedNumerator += input.FinalScore * metric.WeightBPS * WeightTotalBPS / metric.MaxScore
		weightTotal += metric.WeightBPS
	}
	if len(scores) != len(metrics) {
		return nil, 0, validationError("teacher review must include every rubric metric")
	}
	if weightTotal <= 0 {
		return nil, 0, validationError("rubric metric weights must sum to more than zero")
	}
	return scores, weightedNumerator / weightTotal, nil
}

func resolveMetricInput(input TeacherMetricScoreInput, byID map[string]Metric, byCode map[string]Metric) (Metric, error) {
	metricID := strings.TrimSpace(input.MetricID)
	if metricID != "" {
		metric, ok := byID[metricID]
		if !ok {
			return Metric{}, validationError("metric_id does not belong to submission rubric")
		}
		return metric, nil
	}
	code := normalizeCode(input.MetricCode)
	if code == "" {
		return Metric{}, validationError("metric_id or metric_code is required")
	}
	metric, ok := byCode[code]
	if !ok {
		return Metric{}, validationError("metric_code does not belong to submission rubric")
	}
	return metric, nil
}

func normalizeReviewScoreSource(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		return ManualScoreSource
	}
	return source
}
