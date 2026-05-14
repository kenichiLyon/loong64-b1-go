package teaching

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (s *Service) GetSubmissionReport(ctx context.Context, actor Actor, submissionID string) (SubmissionReport, error) {
	if err := s.ready(); err != nil {
		return SubmissionReport{}, err
	}
	return s.buildSubmissionReport(ctx, actor, strings.TrimSpace(submissionID))
}

func (s *Service) GetExperimentReportSummary(ctx context.Context, actor Actor, experimentID string, limit int) (ExperimentReportSummary, error) {
	if err := s.ready(); err != nil {
		return ExperimentReportSummary{}, err
	}
	experimentID = strings.TrimSpace(experimentID)
	if err := s.requireTeacherExperimentAccess(ctx, actor, experimentID); err != nil {
		return ExperimentReportSummary{}, err
	}
	return s.buildExperimentReportSummary(ctx, experimentID, limit)
}

func (s *Service) GetCourseReportSummary(ctx context.Context, actor Actor, courseID string, limit int) (CourseReportSummary, error) {
	if err := s.ready(); err != nil {
		return CourseReportSummary{}, err
	}
	courseID = strings.TrimSpace(courseID)
	if err := s.requireTeacherCourseAccess(ctx, actor, courseID); err != nil {
		return CourseReportSummary{}, err
	}
	experiments, err := s.repo.ListExperimentsForCourse(ctx, courseID, clampLimit(limit))
	if err != nil {
		return CourseReportSummary{}, err
	}
	summary := CourseReportSummary{
		CourseID:              courseID,
		ExperimentCount:       len(experiments),
		ScoreBuckets:          emptyScoreBuckets(),
		SubmissionStatusCount: make(map[string]int),
		ArtifactStatusCount:   make(map[string]int),
		GeneratedAt:           time.Now().UTC(),
	}
	metricStats := map[string]metricAggregate{}
	findingStats := map[string]FindingCount{}
	totalScore := 0
	for _, experiment := range experiments {
		experimentSummary, err := s.buildExperimentReportSummary(ctx, experiment.ID, 200)
		if err != nil {
			return CourseReportSummary{}, err
		}
		summary.Experiments = append(summary.Experiments, experimentSummary)
		totalScore += experimentSummary.scoreSumBPS
		summary.SubmissionCount += experimentSummary.SubmissionCount
		summary.SubmittedCount += experimentSummary.SubmittedCount
		summary.PublishedReviewCount += experimentSummary.PublishedReviewCount
		if experimentSummary.PublishedReviewCount > 0 {
			if summary.MinScoreBPS == 0 || experimentSummary.MinScoreBPS < summary.MinScoreBPS {
				summary.MinScoreBPS = experimentSummary.MinScoreBPS
			}
			if experimentSummary.MaxScoreBPS > summary.MaxScoreBPS {
				summary.MaxScoreBPS = experimentSummary.MaxScoreBPS
			}
		}
		mergeCountMap(summary.ScoreBuckets, experimentSummary.ScoreBuckets)
		mergeCountMap(summary.SubmissionStatusCount, experimentSummary.SubmissionStatusCount)
		mergeCountMap(summary.ArtifactStatusCount, experimentSummary.ArtifactStatusCount)
		for _, metric := range experimentSummary.MetricAverages {
			stat := metricStats[metric.MetricCode]
			stat.metricCode = metric.MetricCode
			stat.finalScoreSum += metric.AverageScore * metric.Count
			stat.percentBPSSum += metric.AveragePercentBPS * metric.Count
			stat.maxScore = metric.MaxScore
			stat.count += metric.Count
			metricStats[metric.MetricCode] = stat
		}
		for _, finding := range experimentSummary.FindingCounts {
			key := finding.Category + "|" + string(finding.Severity)
			entry := findingStats[key]
			entry.Category = finding.Category
			entry.Severity = finding.Severity
			entry.Count += finding.Count
			findingStats[key] = entry
		}
	}
	if summary.PublishedReviewCount > 0 {
		summary.AverageScoreBPS = roundedDivide(totalScore, summary.PublishedReviewCount)
	}
	summary.MetricAverages = buildMetricAverages(metricStats)
	summary.FindingCounts = buildFindingCounts(findingStats)
	return summary, nil
}

func (s *Service) buildExperimentReportSummary(ctx context.Context, experimentID string, limit int) (ExperimentReportSummary, error) {
	submissions, err := s.repo.ListSubmissionsForExperiment(ctx, experimentID, clampLimit(limit))
	if err != nil {
		return ExperimentReportSummary{}, err
	}
	summary := ExperimentReportSummary{
		ExperimentID:          experimentID,
		SubmissionCount:       len(submissions),
		ScoreBuckets:          emptyScoreBuckets(),
		SubmissionStatusCount: make(map[string]int),
		ArtifactStatusCount:   make(map[string]int),
		GeneratedAt:           time.Now().UTC(),
	}
	metricStats := map[string]metricAggregate{}
	findingStats := map[string]FindingCount{}
	totalScore := 0
	for _, submission := range submissions {
		summary.SubmissionStatusCount[submission.Status]++
		if submission.SubmittedAt != nil || submission.Status != "draft" {
			summary.SubmittedCount++
		}
		detail, err := s.repo.GetSubmissionDetail(ctx, submission.ID)
		if err != nil {
			return ExperimentReportSummary{}, err
		}
		for _, artifact := range detail.Artifacts {
			key := fmt.Sprintf("%s/%s", artifact.Artifact.Kind, artifact.Extraction.Status)
			summary.ArtifactStatusCount[key]++
		}
		if evaluation, err := s.repo.GetLatestEvaluation(ctx, submission.ID); err == nil {
			for _, finding := range evaluation.Findings {
				key := finding.Category + "|" + string(finding.Severity)
				entry := findingStats[key]
				entry.Category = finding.Category
				entry.Severity = finding.Severity
				entry.Count++
				findingStats[key] = entry
			}
		} else if ErrorKindOf(err) != KindNotFound {
			return ExperimentReportSummary{}, err
		}
		review, err := s.repo.GetTeacherReview(ctx, submission.ID, true)
		if err != nil {
			if ErrorKindOf(err) == KindNotFound {
				continue
			}
			return ExperimentReportSummary{}, err
		}
		summary.PublishedReviewCount++
		totalScore += review.Review.TotalScoreBPS
		summary.scoreSumBPS += review.Review.TotalScoreBPS
		if summary.PublishedReviewCount == 1 || review.Review.TotalScoreBPS < summary.MinScoreBPS {
			summary.MinScoreBPS = review.Review.TotalScoreBPS
		}
		if review.Review.TotalScoreBPS > summary.MaxScoreBPS {
			summary.MaxScoreBPS = review.Review.TotalScoreBPS
		}
		summary.ScoreBuckets[scoreBucketLabel(review.Review.TotalScoreBPS)]++
		for _, score := range review.Scores {
			stat := metricStats[score.MetricCode]
			stat.metricCode = score.MetricCode
			stat.finalScoreSum += score.FinalScore
			stat.maxScore = score.MaxScore
			stat.percentBPSSum += score.FinalScore * WeightTotalBPS / score.MaxScore
			stat.count++
			metricStats[score.MetricCode] = stat
		}
	}
	if summary.PublishedReviewCount > 0 {
		summary.AverageScoreBPS = roundedDivide(totalScore, summary.PublishedReviewCount)
	}
	summary.MetricAverages = buildMetricAverages(metricStats)
	summary.FindingCounts = buildFindingCounts(findingStats)
	return summary, nil
}

func (s *Service) CreateSubmissionReportExport(ctx context.Context, actor Actor, submissionID string, input CreateReportExportInput, audit AuditEntry) (ReportExport, error) {
	if err := s.ready(); err != nil {
		return ReportExport{}, err
	}
	submissionID = strings.TrimSpace(submissionID)
	format, err := normalizeReportFormat(input.Format)
	if err != nil {
		return ReportExport{}, err
	}
	if !actor.Has(RoleTeacher) && !actor.Has(RoleAdmin) {
		return ReportExport{}, forbiddenError("teacher or admin role is required")
	}
	if len(input.Filters) > 0 && !json.Valid(input.Filters) {
		return ReportExport{}, validationError("filters must be valid JSON")
	}
	report, err := s.buildSubmissionReport(ctx, actor, submissionID)
	if err != nil {
		return ReportExport{}, err
	}
	export := ReportExport{
		ID:          NewID("rpx"),
		ReportType:  ReportTypeSubmissionReport,
		ScopeType:   ReportScopeSubmission,
		ScopeID:     submissionID,
		Format:      format,
		Status:      ReportExportStatusQueued,
		FilterJSON:  mergeExportFilters(input.Filters, map[string]any{"submission_id": submissionID}),
		RequestedBy: actor.ID,
	}
	audit.Action = "report_export.create"
	audit.ActorID = actor.ID
	audit.TargetType = "submission"
	audit.TargetID = submissionID
	audit.Detail = mustJSON(map[string]any{"export_id": export.ID, "format": format, "report_type": export.ReportType})
	created, err := s.repo.CreateReportExport(ctx, export, audit)
	if err != nil {
		return ReportExport{}, err
	}
	payload, extension, err := renderSubmissionReport(report, format)
	if err != nil {
		return s.completeFailedExport(ctx, created, err.Error())
	}
	return s.writeAndCompleteExport(ctx, created, extension, payload)
}

func (s *Service) CreateExperimentSummaryExport(ctx context.Context, actor Actor, experimentID string, input CreateReportExportInput, audit AuditEntry) (ReportExport, error) {
	if err := s.ready(); err != nil {
		return ReportExport{}, err
	}
	experimentID = strings.TrimSpace(experimentID)
	format, err := normalizeReportFormat(input.Format)
	if err != nil {
		return ReportExport{}, err
	}
	if !actor.Has(RoleTeacher) && !actor.Has(RoleAdmin) {
		return ReportExport{}, forbiddenError("teacher or admin role is required")
	}
	if len(input.Filters) > 0 && !json.Valid(input.Filters) {
		return ReportExport{}, validationError("filters must be valid JSON")
	}
	summary, err := s.GetExperimentReportSummary(ctx, actor, experimentID, 200)
	if err != nil {
		return ReportExport{}, err
	}
	export := ReportExport{
		ID:          NewID("rpx"),
		ReportType:  ReportTypeExperimentSummary,
		ScopeType:   ReportScopeExperiment,
		ScopeID:     experimentID,
		Format:      format,
		Status:      ReportExportStatusQueued,
		FilterJSON:  mergeExportFilters(input.Filters, map[string]any{"experiment_id": experimentID}),
		RequestedBy: actor.ID,
	}
	audit.Action = "report_export.create"
	audit.ActorID = actor.ID
	audit.TargetType = "experiment"
	audit.TargetID = experimentID
	audit.Detail = mustJSON(map[string]any{"export_id": export.ID, "format": format, "report_type": export.ReportType})
	created, err := s.repo.CreateReportExport(ctx, export, audit)
	if err != nil {
		return ReportExport{}, err
	}
	payload, extension, err := renderExperimentSummary(summary, format)
	if err != nil {
		return s.completeFailedExport(ctx, created, err.Error())
	}
	return s.writeAndCompleteExport(ctx, created, extension, payload)
}

func (s *Service) CreateCourseSummaryExport(ctx context.Context, actor Actor, courseID string, input CreateReportExportInput, audit AuditEntry) (ReportExport, error) {
	if err := s.ready(); err != nil {
		return ReportExport{}, err
	}
	courseID = strings.TrimSpace(courseID)
	format, err := normalizeReportFormat(input.Format)
	if err != nil {
		return ReportExport{}, err
	}
	if !actor.Has(RoleTeacher) && !actor.Has(RoleAdmin) {
		return ReportExport{}, forbiddenError("teacher or admin role is required")
	}
	if len(input.Filters) > 0 && !json.Valid(input.Filters) {
		return ReportExport{}, validationError("filters must be valid JSON")
	}
	summary, err := s.GetCourseReportSummary(ctx, actor, courseID, 200)
	if err != nil {
		return ReportExport{}, err
	}
	export := ReportExport{
		ID:          NewID("rpx"),
		ReportType:  ReportTypeCourseSummary,
		ScopeType:   ReportScopeCourse,
		ScopeID:     courseID,
		Format:      format,
		Status:      ReportExportStatusQueued,
		FilterJSON:  mergeExportFilters(input.Filters, map[string]any{"course_id": courseID}),
		RequestedBy: actor.ID,
	}
	audit.Action = "report_export.create"
	audit.ActorID = actor.ID
	audit.TargetType = "course"
	audit.TargetID = courseID
	audit.Detail = mustJSON(map[string]any{"export_id": export.ID, "format": format, "report_type": export.ReportType})
	created, err := s.repo.CreateReportExport(ctx, export, audit)
	if err != nil {
		return ReportExport{}, err
	}
	payload, extension, err := renderCourseSummary(summary, format)
	if err != nil {
		return s.completeFailedExport(ctx, created, err.Error())
	}
	return s.writeAndCompleteExport(ctx, created, extension, payload)
}

func (s *Service) GetReportExport(ctx context.Context, actor Actor, exportID string) (ReportExport, error) {
	if err := s.ready(); err != nil {
		return ReportExport{}, err
	}
	export, err := s.repo.GetReportExport(ctx, strings.TrimSpace(exportID))
	if err != nil {
		return ReportExport{}, err
	}
	if err := requireReportExportAccess(actor, export); err != nil {
		return ReportExport{}, err
	}
	return export, nil
}

func (s *Service) OpenReportExport(ctx context.Context, actor Actor, exportID string) (ReportExportFile, error) {
	export, err := s.GetReportExport(ctx, actor, exportID)
	if err != nil {
		return ReportExportFile{}, err
	}
	if export.Status != ReportExportStatusSucceeded || export.StorageKey == "" {
		return ReportExportFile{}, conflictError("report export is not ready for download")
	}
	if s.store == nil {
		return ReportExportFile{}, unavailableError("report storage is not configured", nil)
	}
	path, err := s.store.Resolve(export.StorageKey)
	if err != nil {
		return ReportExportFile{}, unavailableError("report storage key cannot be resolved", err)
	}
	return ReportExportFile{Export: export, Path: path, ContentType: contentTypeForReportFormat(export.Format), FileName: reportExportFileName(export)}, nil
}

func (s *Service) buildSubmissionReport(ctx context.Context, actor Actor, submissionID string) (SubmissionReport, error) {
	if submissionID == "" {
		return SubmissionReport{}, validationError("submission_id is required")
	}
	publishedOnly := false
	if actor.Has(RoleStudent) && !actor.Has(RoleTeacher) && !actor.Has(RoleAdmin) {
		owns, err := s.repo.StudentOwnsSubmission(ctx, submissionID, actor.ID)
		if err != nil {
			return SubmissionReport{}, err
		}
		if !owns {
			return SubmissionReport{}, forbiddenError("student can only view own published report")
		}
		publishedOnly = true
	} else if err := s.requireTeacherSubmissionAccess(ctx, actor, submissionID); err != nil {
		return SubmissionReport{}, err
	}
	evalCtx, err := s.repo.GetEvaluationContext(ctx, submissionID)
	if err != nil {
		return SubmissionReport{}, err
	}
	review, err := s.repo.GetTeacherReview(ctx, submissionID, publishedOnly)
	if err != nil {
		return SubmissionReport{}, err
	}
	var evaluation *EvaluationResultDetail
	latest, err := s.repo.GetLatestEvaluation(ctx, submissionID)
	if err == nil {
		evaluation = &latest
	} else if ErrorKindOf(err) != KindNotFound {
		return SubmissionReport{}, err
	}
	return SubmissionReport{
		Submission:  evalCtx.Submission,
		Experiment:  evalCtx.Experiment,
		Artifacts:   evalCtx.Artifacts,
		Review:      review,
		Evaluation:  evaluation,
		GeneratedAt: time.Now().UTC(),
	}, nil
}

func (s *Service) writeAndCompleteExport(ctx context.Context, export ReportExport, extension string, payload []byte) (ReportExport, error) {
	if s.store == nil {
		return s.completeFailedExport(ctx, export, "report storage is not configured")
	}
	storageKey := fmt.Sprintf("reports/%s/%s/%s.%s", export.ScopeType, safeStoragePart(export.ScopeID), export.ID, extension)
	path, err := s.store.Resolve(storageKey)
	if err != nil {
		return s.completeFailedExport(ctx, export, err.Error())
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return s.completeFailedExport(ctx, export, err.Error())
	}
	if err := os.WriteFile(path, payload, 0o640); err != nil {
		return s.completeFailedExport(ctx, export, err.Error())
	}
	export.Status = ReportExportStatusSucceeded
	export.StorageKey = storageKey
	export.SHA256Hex = sha256Hex(payload)
	export.ByteSize = int64(len(payload))
	export.Error = ""
	return s.repo.CompleteReportExport(ctx, export)
}

func (s *Service) completeFailedExport(ctx context.Context, export ReportExport, message string) (ReportExport, error) {
	export.Status = ReportExportStatusFailed
	export.StorageKey = ""
	export.SHA256Hex = ""
	export.ByteSize = 0
	export.Error = sanitizeEvidenceText(message, 500)
	return s.repo.CompleteReportExport(ctx, export)
}

func normalizeReportFormat(format ReportFormat) (ReportFormat, error) {
	value := ReportFormat(strings.ToLower(strings.TrimSpace(string(format))))
	if value == "" {
		return ReportFormatHTML, nil
	}
	switch value {
	case ReportFormatHTML, ReportFormatCSV, ReportFormatXLSX, ReportFormatPDF:
		return value, nil
	default:
		return "", validationError("invalid report export format")
	}
}

func mergeExportFilters(raw json.RawMessage, enforced map[string]any) json.RawMessage {
	filters := map[string]any{}
	if len(raw) > 0 && json.Valid(raw) {
		_ = json.Unmarshal(raw, &filters)
	}
	for key, value := range enforced {
		filters[key] = value
	}
	return mustJSON(filters)
}

func requireReportExportAccess(actor Actor, export ReportExport) error {
	if actor.Has(RoleAdmin) {
		return nil
	}
	if err := actor.Require(RoleTeacher); err != nil {
		return err
	}
	if export.RequestedBy != actor.ID {
		return forbiddenError("teacher can only access own report exports")
	}
	return nil
}

func (s *Service) requireTeacherCourseAccess(ctx context.Context, actor Actor, courseID string) error {
	if courseID == "" {
		return validationError("course_id is required")
	}
	if actor.Has(RoleAdmin) {
		return nil
	}
	if err := actor.Require(RoleTeacher); err != nil {
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

func renderSubmissionReport(report SubmissionReport, format ReportFormat) ([]byte, string, error) {
	switch format {
	case ReportFormatHTML:
		return []byte(renderSubmissionReportHTML(report)), "html", nil
	case ReportFormatCSV:
		return renderSubmissionReportCSV(report), "csv", nil
	case ReportFormatXLSX:
		payload, err := renderSubmissionReportXLSX(report)
		return payload, "xlsx", err
	case ReportFormatPDF:
		payload, err := renderSubmissionReportPDF(report)
		return payload, "pdf", err
	default:
		return nil, "", validationError("unsupported report format")
	}
}

func renderExperimentSummary(summary ExperimentReportSummary, format ReportFormat) ([]byte, string, error) {
	switch format {
	case ReportFormatHTML:
		return []byte(renderExperimentSummaryHTML(summary)), "html", nil
	case ReportFormatCSV:
		return renderExperimentSummaryCSV(summary), "csv", nil
	case ReportFormatXLSX:
		payload, err := renderExperimentSummaryXLSX(summary)
		return payload, "xlsx", err
	case ReportFormatPDF:
		payload, err := renderExperimentSummaryPDF(summary)
		return payload, "pdf", err
	default:
		return nil, "", validationError("unsupported report format")
	}
}

func renderCourseSummary(summary CourseReportSummary, format ReportFormat) ([]byte, string, error) {
	switch format {
	case ReportFormatHTML:
		return []byte(renderCourseSummaryHTML(summary)), "html", nil
	case ReportFormatCSV:
		return renderCourseSummaryCSV(summary), "csv", nil
	case ReportFormatXLSX:
		payload, err := renderCourseSummaryXLSX(summary)
		return payload, "xlsx", err
	case ReportFormatPDF:
		payload, err := renderCourseSummaryPDF(summary)
		return payload, "pdf", err
	default:
		return nil, "", validationError("unsupported report format")
	}
}

func renderSubmissionReportHTML(report SubmissionReport) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><title>Submission Report</title>")
	b.WriteString(reportStyle())
	b.WriteString("</head><body><main><header><p>LoongArch Training Evaluation</p><h1>学生个人评价报告</h1></header>")
	b.WriteString("<section class=\"summary\">")
	writeFact(&b, "提交ID", report.Submission.ID)
	writeFact(&b, "实验", report.Experiment.Title)
	writeFact(&b, "学生ID", report.Submission.StudentID)
	writeFact(&b, "最终分", scorePercent(report.Review.Review.TotalScoreBPS))
	writeFact(&b, "复核状态", string(report.Review.Review.Status))
	writeFact(&b, "生成时间", report.GeneratedAt.Format(time.RFC3339))
	b.WriteString("</section>")
	b.WriteString("<section><h2>教师总评</h2><p>")
	b.WriteString(html.EscapeString(report.Review.Review.TeacherComment))
	b.WriteString("</p></section><section><h2>指标得分</h2><table><thead><tr><th>指标</th><th>得分</th><th>权重</th><th>来源</th><th>说明</th></tr></thead><tbody>")
	for _, score := range report.Review.Scores {
		b.WriteString("<tr><td>")
		b.WriteString(html.EscapeString(score.MetricCode))
		b.WriteString("</td><td>")
		fmt.Fprintf(&b, "%d / %d", score.FinalScore, score.MaxScore)
		b.WriteString("</td><td>")
		fmt.Fprintf(&b, "%d", score.WeightBPS)
		b.WriteString("</td><td>")
		b.WriteString(html.EscapeString(score.Source))
		b.WriteString("</td><td>")
		b.WriteString(html.EscapeString(strings.TrimSpace(score.Comment + " " + score.AdjustmentReason)))
		b.WriteString("</td></tr>")
	}
	b.WriteString("</tbody></table></section><section><h2>成果证据</h2><table><thead><tr><th>文件</th><th>类型</th><th>解析状态</th><th>摘要</th></tr></thead><tbody>")
	for _, item := range report.Artifacts {
		b.WriteString("<tr><td>")
		b.WriteString(html.EscapeString(item.Artifact.OriginalName))
		b.WriteString("</td><td>")
		b.WriteString(html.EscapeString(string(item.Artifact.Kind)))
		b.WriteString("</td><td>")
		b.WriteString(html.EscapeString(item.Extraction.Status))
		b.WriteString("</td><td>")
		b.WriteString(html.EscapeString(firstNonEmpty(item.Extraction.TextExcerpt, item.Extraction.Error)))
		b.WriteString("</td></tr>")
	}
	b.WriteString("</tbody></table></section>")
	if report.Evaluation != nil {
		b.WriteString("<section><h2>智能核查摘要</h2><p>")
		b.WriteString(html.EscapeString(report.Evaluation.Result.LLMSummary))
		b.WriteString("</p><table><thead><tr><th>等级</th><th>类别</th><th>问题</th><th>证据引用</th></tr></thead><tbody>")
		for _, finding := range report.Evaluation.Findings {
			b.WriteString("<tr><td>")
			b.WriteString(html.EscapeString(string(finding.Severity)))
			b.WriteString("</td><td>")
			b.WriteString(html.EscapeString(finding.Category))
			b.WriteString("</td><td>")
			b.WriteString(html.EscapeString(finding.Message))
			b.WriteString("</td><td>")
			b.WriteString(html.EscapeString(strings.TrimSpace(finding.EvidenceRef)))
			b.WriteString("</td></tr>")
		}
		b.WriteString("</tbody></table><h3>AI 指标建议</h3><table><thead><tr><th>指标</th><th>建议分</th><th>理由</th><th>证据引用</th></tr></thead><tbody>")
		for _, score := range report.Evaluation.Scores {
			b.WriteString("<tr><td>")
			b.WriteString(html.EscapeString(score.MetricCode))
			b.WriteString("</td><td>")
			fmt.Fprintf(&b, "%d / %d", score.SuggestedScore, score.MaxScore)
			b.WriteString("</td><td>")
			b.WriteString(html.EscapeString(score.Rationale))
			b.WriteString("</td><td>")
			b.WriteString(html.EscapeString(strings.Join(parseEvidenceRefs(score.EvidenceRefs), ", ")))
			b.WriteString("</td></tr>")
		}
		b.WriteString("</tbody></table></section>")
	}
	b.WriteString("</main></body></html>")
	return b.String()
}

func renderSubmissionReportCSV(report SubmissionReport) []byte {
	var b bytes.Buffer
	b.WriteString("\ufeff")
	writer := csv.NewWriter(&b)
	writeCSV(writer, []string{"个人评价报告"})
	writeCSV(writer, []string{"提交ID", report.Submission.ID})
	writeCSV(writer, []string{"实验ID", report.Submission.ExperimentID})
	writeCSV(writer, []string{"实验标题", report.Experiment.Title})
	writeCSV(writer, []string{"学生ID", report.Submission.StudentID})
	writeCSV(writer, []string{"最终分", scorePercent(report.Review.Review.TotalScoreBPS)})
	writeCSV(writer, []string{"教师总评", report.Review.Review.TeacherComment})
	writeCSV(writer, []string{})
	writeCSV(writer, []string{"指标", "得分", "满分", "权重BPS", "来源", "评语", "改分说明"})
	for _, score := range report.Review.Scores {
		writeCSV(writer, []string{score.MetricCode, fmt.Sprint(score.FinalScore), fmt.Sprint(score.MaxScore), fmt.Sprint(score.WeightBPS), score.Source, score.Comment, score.AdjustmentReason})
	}
	writeCSV(writer, []string{})
	writeCSV(writer, []string{"成果文件", "类型", "解析状态", "摘要或错误"})
	for _, item := range report.Artifacts {
		writeCSV(writer, []string{item.Artifact.OriginalName, string(item.Artifact.Kind), item.Extraction.Status, firstNonEmpty(item.Extraction.TextExcerpt, item.Extraction.Error)})
	}
	if report.Evaluation != nil {
		writeCSV(writer, []string{})
		writeCSV(writer, []string{"智能核查发现", "严重度", "类别", "消息", "证据引用"})
		for _, finding := range report.Evaluation.Findings {
			writeCSV(writer, []string{"finding", string(finding.Severity), finding.Category, finding.Message, finding.EvidenceRef})
		}
		writeCSV(writer, []string{})
		writeCSV(writer, []string{"智能指标建议", "指标", "建议分", "满分", "理由", "证据引用"})
		for _, score := range report.Evaluation.Scores {
			writeCSV(writer, []string{"metric_score", score.MetricCode, fmt.Sprint(score.SuggestedScore), fmt.Sprint(score.MaxScore), score.Rationale, strings.Join(parseEvidenceRefs(score.EvidenceRefs), " | ")})
		}
	}
	writer.Flush()
	return b.Bytes()
}

func renderExperimentSummaryHTML(summary ExperimentReportSummary) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><title>Experiment Summary</title>")
	b.WriteString(reportStyle())
	b.WriteString("</head><body><main><header><p>LoongArch Training Evaluation</p><h1>实验统计报表</h1></header><section class=\"summary\">")
	writeFact(&b, "实验ID", summary.ExperimentID)
	writeFact(&b, "提交数", fmt.Sprint(summary.SubmissionCount))
	writeFact(&b, "已发布评价", fmt.Sprint(summary.PublishedReviewCount))
	writeFact(&b, "平均分", scorePercent(summary.AverageScoreBPS))
	writeFact(&b, "最高分", scorePercent(summary.MaxScoreBPS))
	writeFact(&b, "最低分", scorePercent(summary.MinScoreBPS))
	b.WriteString("</section><section><h2>分数分布</h2><table><thead><tr><th>区间</th><th>数量</th></tr></thead><tbody>")
	for _, label := range scoreBucketOrder() {
		b.WriteString("<tr><td>")
		b.WriteString(html.EscapeString(label))
		b.WriteString("</td><td>")
		fmt.Fprint(&b, summary.ScoreBuckets[label])
		b.WriteString("</td></tr>")
	}
	b.WriteString("</tbody></table></section><section><h2>指标均值</h2><table><thead><tr><th>指标</th><th>平均得分</th><th>满分</th><th>平均百分比</th><th>样本数</th></tr></thead><tbody>")
	for _, metric := range summary.MetricAverages {
		b.WriteString("<tr><td>")
		b.WriteString(html.EscapeString(metric.MetricCode))
		b.WriteString("</td><td>")
		fmt.Fprint(&b, metric.AverageScore)
		b.WriteString("</td><td>")
		fmt.Fprint(&b, metric.MaxScore)
		b.WriteString("</td><td>")
		b.WriteString(scorePercent(metric.AveragePercentBPS))
		b.WriteString("</td><td>")
		fmt.Fprint(&b, metric.Count)
		b.WriteString("</td></tr>")
	}
	b.WriteString("</tbody></table></section><section><h2>常见问题</h2><table><thead><tr><th>严重度</th><th>类别</th><th>数量</th></tr></thead><tbody>")
	for _, finding := range summary.FindingCounts {
		b.WriteString("<tr><td>")
		b.WriteString(html.EscapeString(string(finding.Severity)))
		b.WriteString("</td><td>")
		b.WriteString(html.EscapeString(finding.Category))
		b.WriteString("</td><td>")
		fmt.Fprint(&b, finding.Count)
		b.WriteString("</td></tr>")
	}
	b.WriteString("</tbody></table></section></main></body></html>")
	return b.String()
}

func renderExperimentSummaryCSV(summary ExperimentReportSummary) []byte {
	var b bytes.Buffer
	b.WriteString("\ufeff")
	writer := csv.NewWriter(&b)
	writeCSV(writer, []string{"实验统计报表"})
	writeCSV(writer, []string{"实验ID", summary.ExperimentID})
	writeCSV(writer, []string{"提交数", fmt.Sprint(summary.SubmissionCount)})
	writeCSV(writer, []string{"已提交数", fmt.Sprint(summary.SubmittedCount)})
	writeCSV(writer, []string{"已发布评价", fmt.Sprint(summary.PublishedReviewCount)})
	writeCSV(writer, []string{"平均分", scorePercent(summary.AverageScoreBPS)})
	writeCSV(writer, []string{"最高分", scorePercent(summary.MaxScoreBPS)})
	writeCSV(writer, []string{"最低分", scorePercent(summary.MinScoreBPS)})
	writeCSV(writer, []string{})
	writeCSV(writer, []string{"分数区间", "数量"})
	for _, label := range scoreBucketOrder() {
		writeCSV(writer, []string{label, fmt.Sprint(summary.ScoreBuckets[label])})
	}
	writeCSV(writer, []string{})
	writeCSV(writer, []string{"指标", "平均得分", "满分", "平均百分比", "样本数"})
	for _, metric := range summary.MetricAverages {
		writeCSV(writer, []string{metric.MetricCode, fmt.Sprint(metric.AverageScore), fmt.Sprint(metric.MaxScore), scorePercent(metric.AveragePercentBPS), fmt.Sprint(metric.Count)})
	}
	writeCSV(writer, []string{})
	writeCSV(writer, []string{"问题严重度", "类别", "数量"})
	for _, finding := range summary.FindingCounts {
		writeCSV(writer, []string{string(finding.Severity), finding.Category, fmt.Sprint(finding.Count)})
	}
	writer.Flush()
	return b.Bytes()
}

func renderCourseSummaryHTML(summary CourseReportSummary) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><title>Course Summary</title>")
	b.WriteString(reportStyle())
	b.WriteString("</head><body><main><header><p>LoongArch Training Evaluation</p><h1>课程统计报表</h1></header><section class=\"summary\">")
	writeFact(&b, "课程ID", summary.CourseID)
	writeFact(&b, "实验数", fmt.Sprint(summary.ExperimentCount))
	writeFact(&b, "提交数", fmt.Sprint(summary.SubmissionCount))
	writeFact(&b, "已发布评价", fmt.Sprint(summary.PublishedReviewCount))
	writeFact(&b, "平均分", scorePercent(summary.AverageScoreBPS))
	writeFact(&b, "最高分", scorePercent(summary.MaxScoreBPS))
	writeFact(&b, "最低分", scorePercent(summary.MinScoreBPS))
	b.WriteString("</section><section><h2>实验对比</h2><table><thead><tr><th>实验ID</th><th>提交数</th><th>已提交</th><th>已发布评价</th><th>平均分</th><th>最高分</th><th>最低分</th></tr></thead><tbody>")
	for _, experiment := range summary.Experiments {
		b.WriteString("<tr><td>")
		b.WriteString(html.EscapeString(experiment.ExperimentID))
		b.WriteString("</td><td>")
		fmt.Fprint(&b, experiment.SubmissionCount)
		b.WriteString("</td><td>")
		fmt.Fprint(&b, experiment.SubmittedCount)
		b.WriteString("</td><td>")
		fmt.Fprint(&b, experiment.PublishedReviewCount)
		b.WriteString("</td><td>")
		b.WriteString(scorePercent(experiment.AverageScoreBPS))
		b.WriteString("</td><td>")
		b.WriteString(scorePercent(experiment.MaxScoreBPS))
		b.WriteString("</td><td>")
		b.WriteString(scorePercent(experiment.MinScoreBPS))
		b.WriteString("</td></tr>")
	}
	b.WriteString("</tbody></table></section><section><h2>课程分数分布</h2><table><thead><tr><th>区间</th><th>数量</th></tr></thead><tbody>")
	for _, label := range scoreBucketOrder() {
		b.WriteString("<tr><td>")
		b.WriteString(html.EscapeString(label))
		b.WriteString("</td><td>")
		fmt.Fprint(&b, summary.ScoreBuckets[label])
		b.WriteString("</td></tr>")
	}
	b.WriteString("</tbody></table></section><section><h2>课程指标均值</h2><table><thead><tr><th>指标</th><th>平均得分</th><th>满分</th><th>平均百分比</th><th>样本数</th></tr></thead><tbody>")
	for _, metric := range summary.MetricAverages {
		b.WriteString("<tr><td>")
		b.WriteString(html.EscapeString(metric.MetricCode))
		b.WriteString("</td><td>")
		fmt.Fprint(&b, metric.AverageScore)
		b.WriteString("</td><td>")
		fmt.Fprint(&b, metric.MaxScore)
		b.WriteString("</td><td>")
		b.WriteString(scorePercent(metric.AveragePercentBPS))
		b.WriteString("</td><td>")
		fmt.Fprint(&b, metric.Count)
		b.WriteString("</td></tr>")
	}
	b.WriteString("</tbody></table></section><section><h2>常见问题</h2><table><thead><tr><th>严重度</th><th>类别</th><th>数量</th></tr></thead><tbody>")
	for _, finding := range summary.FindingCounts {
		b.WriteString("<tr><td>")
		b.WriteString(html.EscapeString(string(finding.Severity)))
		b.WriteString("</td><td>")
		b.WriteString(html.EscapeString(finding.Category))
		b.WriteString("</td><td>")
		fmt.Fprint(&b, finding.Count)
		b.WriteString("</td></tr>")
	}
	b.WriteString("</tbody></table></section></main></body></html>")
	return b.String()
}

func renderCourseSummaryCSV(summary CourseReportSummary) []byte {
	var b bytes.Buffer
	b.WriteString("\ufeff")
	writer := csv.NewWriter(&b)
	writeCSV(writer, []string{"课程统计报表"})
	writeCSV(writer, []string{"课程ID", summary.CourseID})
	writeCSV(writer, []string{"实验数", fmt.Sprint(summary.ExperimentCount)})
	writeCSV(writer, []string{"提交数", fmt.Sprint(summary.SubmissionCount)})
	writeCSV(writer, []string{"已提交数", fmt.Sprint(summary.SubmittedCount)})
	writeCSV(writer, []string{"已发布评价", fmt.Sprint(summary.PublishedReviewCount)})
	writeCSV(writer, []string{"平均分", scorePercent(summary.AverageScoreBPS)})
	writeCSV(writer, []string{"最高分", scorePercent(summary.MaxScoreBPS)})
	writeCSV(writer, []string{"最低分", scorePercent(summary.MinScoreBPS)})
	writeCSV(writer, []string{})
	writeCSV(writer, []string{"实验ID", "提交数", "已提交数", "已发布评价", "平均分", "最高分", "最低分"})
	for _, experiment := range summary.Experiments {
		writeCSV(writer, []string{experiment.ExperimentID, fmt.Sprint(experiment.SubmissionCount), fmt.Sprint(experiment.SubmittedCount), fmt.Sprint(experiment.PublishedReviewCount), scorePercent(experiment.AverageScoreBPS), scorePercent(experiment.MaxScoreBPS), scorePercent(experiment.MinScoreBPS)})
	}
	writeCSV(writer, []string{})
	writeCSV(writer, []string{"课程分数区间", "数量"})
	for _, label := range scoreBucketOrder() {
		writeCSV(writer, []string{label, fmt.Sprint(summary.ScoreBuckets[label])})
	}
	writeCSV(writer, []string{})
	writeCSV(writer, []string{"指标", "平均得分", "满分", "平均百分比", "样本数"})
	for _, metric := range summary.MetricAverages {
		writeCSV(writer, []string{metric.MetricCode, fmt.Sprint(metric.AverageScore), fmt.Sprint(metric.MaxScore), scorePercent(metric.AveragePercentBPS), fmt.Sprint(metric.Count)})
	}
	writeCSV(writer, []string{})
	writeCSV(writer, []string{"问题严重度", "类别", "数量"})
	for _, finding := range summary.FindingCounts {
		writeCSV(writer, []string{string(finding.Severity), finding.Category, fmt.Sprint(finding.Count)})
	}
	writer.Flush()
	return b.Bytes()
}

func writeCSV(writer *csv.Writer, record []string) {
	_ = writer.Write(record)
}

func writeFact(b *strings.Builder, label, value string) {
	b.WriteString("<div><span>")
	b.WriteString(html.EscapeString(label))
	b.WriteString("</span><strong>")
	b.WriteString(html.EscapeString(value))
	b.WriteString("</strong></div>")
}

func reportStyle() string {
	return `<style>body{margin:0;background:#f5efe4;color:#21160e;font-family:"Noto Serif SC","Source Han Serif SC",serif}main{max-width:1100px;margin:0 auto;padding:40px}header{border-bottom:4px solid #d84a27;margin-bottom:24px}header p{color:#d84a27;font-weight:700;letter-spacing:.12em;text-transform:uppercase}h1{font-size:42px;margin:0 0 18px}section{background:#fffaf0;border:1px solid rgba(84,64,42,.18);border-radius:24px;padding:22px;margin:16px 0}.summary{display:grid;grid-template-columns:repeat(auto-fit,minmax(170px,1fr));gap:12px}.summary div{background:#f4efe4;border-radius:16px;padding:14px}.summary span{display:block;color:#776958;font-size:13px}table{width:100%;border-collapse:collapse}th,td{border-bottom:1px solid rgba(84,64,42,.18);padding:10px;text-align:left;vertical-align:top}th{color:#29556b}</style>`
}

func scorePercent(bps int) string {
	return fmt.Sprintf("%.2f%%", float64(bps)/100)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type metricAggregate struct {
	metricCode    string
	finalScoreSum int
	percentBPSSum int
	maxScore      int
	count         int
}

func buildMetricAverages(stats map[string]metricAggregate) []MetricAverage {
	metrics := make([]MetricAverage, 0, len(stats))
	for _, stat := range stats {
		if stat.count == 0 {
			continue
		}
		metrics = append(metrics, MetricAverage{MetricCode: stat.metricCode, AverageScore: roundedDivide(stat.finalScoreSum, stat.count), AveragePercentBPS: roundedDivide(stat.percentBPSSum, stat.count), MaxScore: stat.maxScore, Count: stat.count})
	}
	sort.Slice(metrics, func(i, j int) bool { return metrics[i].MetricCode < metrics[j].MetricCode })
	return metrics
}

func buildFindingCounts(stats map[string]FindingCount) []FindingCount {
	findings := make([]FindingCount, 0, len(stats))
	for _, finding := range stats {
		findings = append(findings, finding)
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Severity == findings[j].Severity {
			return findings[i].Category < findings[j].Category
		}
		return severityRank(findings[i].Severity) > severityRank(findings[j].Severity)
	})
	return findings
}

func mergeCountMap(dst, src map[string]int) {
	for key, count := range src {
		dst[key] += count
	}
}

func severityRank(severity FindingSeverity) int {
	switch severity {
	case FindingCritical:
		return 5
	case FindingHigh:
		return 4
	case FindingMedium:
		return 3
	case FindingLow:
		return 2
	default:
		return 1
	}
}

func roundedDivide(sum, count int) int {
	if count <= 0 {
		return 0
	}
	return (sum + count/2) / count
}

func emptyScoreBuckets() map[string]int {
	buckets := make(map[string]int, len(scoreBucketOrder()))
	for _, label := range scoreBucketOrder() {
		buckets[label] = 0
	}
	return buckets
}

func scoreBucketOrder() []string {
	return []string{"0-59%", "60-69%", "70-79%", "80-89%", "90-100%"}
}

func scoreBucketLabel(scoreBPS int) string {
	switch {
	case scoreBPS < 6000:
		return "0-59%"
	case scoreBPS < 7000:
		return "60-69%"
	case scoreBPS < 8000:
		return "70-79%"
	case scoreBPS < 9000:
		return "80-89%"
	default:
		return "90-100%"
	}
}

func safeStoragePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, "/", "_")
	return value
}

func contentTypeForReportFormat(format ReportFormat) string {
	switch format {
	case ReportFormatCSV:
		return "text/csv; charset=utf-8"
	case ReportFormatXLSX:
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ReportFormatPDF:
		return "application/pdf"
	default:
		return "text/html; charset=utf-8"
	}
}

func reportExportFileName(export ReportExport) string {
	ext := string(export.Format)
	if ext == "" {
		ext = "html"
	}
	return fmt.Sprintf("%s-%s.%s", export.ReportType, export.ID, ext)
}

func parseEvidenceRefs(raw json.RawMessage) []string {
	var refs []string
	if len(raw) == 0 {
		return refs
	}
	if err := json.Unmarshal(raw, &refs); err != nil {
		return nil
	}
	filtered := make([]string, 0, len(refs))
	for _, ref := range refs {
		if strings.TrimSpace(ref) != "" {
			filtered = append(filtered, ref)
		}
	}
	return filtered
}
