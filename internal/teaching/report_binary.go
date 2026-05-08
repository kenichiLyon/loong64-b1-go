package teaching

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/phpdave11/gofpdf"
	"github.com/xuri/excelize/v2"
)

//go:embed assets/fonts/NotoSansSC-VF.ttf
var reportPDFFont []byte

func renderSubmissionReportXLSX(report SubmissionReport) ([]byte, error) {
	file := excelize.NewFile()
	summarySheet := "Summary"
	_ = file.SetSheetName(file.GetSheetName(file.GetActiveSheetIndex()), summarySheet)
	rows := [][]any{
		{"学生个人评价报告"},
		{"提交ID", report.Submission.ID},
		{"实验ID", report.Submission.ExperimentID},
		{"实验标题", report.Experiment.Title},
		{"学生ID", report.Submission.StudentID},
		{"最终分", scorePercent(report.Review.Review.TotalScoreBPS)},
		{"教师总评", report.Review.Review.TeacherComment},
		{"生成时间", report.GeneratedAt.Format(time.RFC3339)},
	}
	writeRows(file, summarySheet, rows)

	metricSheet := "Metrics"
	_, _ = file.NewSheet(metricSheet)
	metricRows := [][]any{{"指标", "得分", "满分", "权重BPS", "来源", "评语", "改分说明"}}
	for _, score := range report.Review.Scores {
		metricRows = append(metricRows, []any{score.MetricCode, score.FinalScore, score.MaxScore, score.WeightBPS, score.Source, score.Comment, score.AdjustmentReason})
	}
	writeRows(file, metricSheet, metricRows)

	artifactSheet := "Artifacts"
	_, _ = file.NewSheet(artifactSheet)
	artifactRows := [][]any{{"成果文件", "类型", "解析状态", "摘要或错误"}}
	for _, item := range report.Artifacts {
		artifactRows = append(artifactRows, []any{item.Artifact.OriginalName, string(item.Artifact.Kind), item.Extraction.Status, firstNonEmpty(item.Extraction.TextExcerpt, item.Extraction.Error)})
	}
	writeRows(file, artifactSheet, artifactRows)

	if report.Evaluation != nil {
		findingSheet := "Findings"
		_, _ = file.NewSheet(findingSheet)
		findingRows := [][]any{{"严重度", "类别", "消息"}}
		for _, finding := range report.Evaluation.Findings {
			findingRows = append(findingRows, []any{finding.Severity, finding.Category, finding.Message})
		}
		writeRows(file, findingSheet, findingRows)
	}
	return workbookBytes(file)
}

func renderExperimentSummaryXLSX(summary ExperimentReportSummary) ([]byte, error) {
	file := excelize.NewFile()
	sheet := "Summary"
	_ = file.SetSheetName(file.GetSheetName(file.GetActiveSheetIndex()), sheet)
	rows := [][]any{
		{"实验统计报表"},
		{"实验ID", summary.ExperimentID},
		{"提交数", summary.SubmissionCount},
		{"已提交数", summary.SubmittedCount},
		{"已发布评价", summary.PublishedReviewCount},
		{"平均分", scorePercent(summary.AverageScoreBPS)},
		{"最高分", scorePercent(summary.MaxScoreBPS)},
		{"最低分", scorePercent(summary.MinScoreBPS)},
	}
	writeRows(file, sheet, rows)

	buckets := "Buckets"
	_, _ = file.NewSheet(buckets)
	bucketRows := [][]any{{"分数区间", "数量"}}
	for _, label := range scoreBucketOrder() {
		bucketRows = append(bucketRows, []any{label, summary.ScoreBuckets[label]})
	}
	writeRows(file, buckets, bucketRows)

	metrics := "Metrics"
	_, _ = file.NewSheet(metrics)
	metricRows := [][]any{{"指标", "平均得分", "满分", "平均百分比", "样本数"}}
	for _, metric := range summary.MetricAverages {
		metricRows = append(metricRows, []any{metric.MetricCode, metric.AverageScore, metric.MaxScore, scorePercent(metric.AveragePercentBPS), metric.Count})
	}
	writeRows(file, metrics, metricRows)

	findings := "Findings"
	_, _ = file.NewSheet(findings)
	findingRows := [][]any{{"严重度", "类别", "数量"}}
	for _, finding := range summary.FindingCounts {
		findingRows = append(findingRows, []any{finding.Severity, finding.Category, finding.Count})
	}
	writeRows(file, findings, findingRows)
	return workbookBytes(file)
}

func renderCourseSummaryXLSX(summary CourseReportSummary) ([]byte, error) {
	file := excelize.NewFile()
	sheet := "Summary"
	_ = file.SetSheetName(file.GetSheetName(file.GetActiveSheetIndex()), sheet)
	rows := [][]any{
		{"课程统计报表"},
		{"课程ID", summary.CourseID},
		{"实验数", summary.ExperimentCount},
		{"提交数", summary.SubmissionCount},
		{"已提交数", summary.SubmittedCount},
		{"已发布评价", summary.PublishedReviewCount},
		{"平均分", scorePercent(summary.AverageScoreBPS)},
		{"最高分", scorePercent(summary.MaxScoreBPS)},
		{"最低分", scorePercent(summary.MinScoreBPS)},
	}
	writeRows(file, sheet, rows)

	experimentSheet := "Experiments"
	_, _ = file.NewSheet(experimentSheet)
	experimentRows := [][]any{{"实验ID", "提交数", "已提交数", "已发布评价", "平均分", "最高分", "最低分"}}
	for _, experiment := range summary.Experiments {
		experimentRows = append(experimentRows, []any{experiment.ExperimentID, experiment.SubmissionCount, experiment.SubmittedCount, experiment.PublishedReviewCount, scorePercent(experiment.AverageScoreBPS), scorePercent(experiment.MaxScoreBPS), scorePercent(experiment.MinScoreBPS)})
	}
	writeRows(file, experimentSheet, experimentRows)

	buckets := "Buckets"
	_, _ = file.NewSheet(buckets)
	bucketRows := [][]any{{"分数区间", "数量"}}
	for _, label := range scoreBucketOrder() {
		bucketRows = append(bucketRows, []any{label, summary.ScoreBuckets[label]})
	}
	writeRows(file, buckets, bucketRows)

	metrics := "Metrics"
	_, _ = file.NewSheet(metrics)
	metricRows := [][]any{{"指标", "平均得分", "满分", "平均百分比", "样本数"}}
	for _, metric := range summary.MetricAverages {
		metricRows = append(metricRows, []any{metric.MetricCode, metric.AverageScore, metric.MaxScore, scorePercent(metric.AveragePercentBPS), metric.Count})
	}
	writeRows(file, metrics, metricRows)

	findings := "Findings"
	_, _ = file.NewSheet(findings)
	findingRows := [][]any{{"严重度", "类别", "数量"}}
	for _, finding := range summary.FindingCounts {
		findingRows = append(findingRows, []any{finding.Severity, finding.Category, finding.Count})
	}
	writeRows(file, findings, findingRows)
	return workbookBytes(file)
}

func renderSubmissionReportPDF(report SubmissionReport) ([]byte, error) {
	doc := newReportPDF("submission-report")
	writePDFTitle(doc, "学生个人评价报告")
	writePDFFacts(doc, [][2]string{
		{"提交ID", report.Submission.ID},
		{"实验", report.Experiment.Title},
		{"学生ID", report.Submission.StudentID},
		{"最终分", scorePercent(report.Review.Review.TotalScoreBPS)},
		{"复核状态", string(report.Review.Review.Status)},
		{"生成时间", report.GeneratedAt.Format(time.RFC3339)},
	})
	writePDFSection(doc, "教师总评", report.Review.Review.TeacherComment)
	for _, score := range report.Review.Scores {
		writePDFSection(doc, "指标 "+score.MetricCode, fmt.Sprintf("得分 %d/%d，权重 %d，来源 %s。%s", score.FinalScore, score.MaxScore, score.WeightBPS, score.Source, strings.TrimSpace(score.Comment+" "+score.AdjustmentReason)))
	}
	for _, item := range report.Artifacts {
		writePDFSection(doc, "成果 "+item.Artifact.OriginalName, fmt.Sprintf("类型 %s，解析状态 %s。%s", item.Artifact.Kind, item.Extraction.Status, firstNonEmpty(item.Extraction.TextExcerpt, item.Extraction.Error)))
	}
	if report.Evaluation != nil {
		writePDFSection(doc, "智能核查摘要", report.Evaluation.Result.LLMSummary)
		for _, finding := range report.Evaluation.Findings {
			writePDFSection(doc, fmt.Sprintf("发现 %s/%s", finding.Severity, finding.Category), finding.Message)
		}
	}
	return pdfBytes(doc)
}

func renderExperimentSummaryPDF(summary ExperimentReportSummary) ([]byte, error) {
	doc := newReportPDF("experiment-summary")
	writePDFTitle(doc, "实验统计报表")
	writePDFFacts(doc, [][2]string{
		{"实验ID", summary.ExperimentID},
		{"提交数", fmt.Sprint(summary.SubmissionCount)},
		{"已提交数", fmt.Sprint(summary.SubmittedCount)},
		{"已发布评价", fmt.Sprint(summary.PublishedReviewCount)},
		{"平均分", scorePercent(summary.AverageScoreBPS)},
		{"最高分", scorePercent(summary.MaxScoreBPS)},
		{"最低分", scorePercent(summary.MinScoreBPS)},
	})
	writePDFSection(doc, "分数分布", formatBuckets(summary.ScoreBuckets))
	writePDFSection(doc, "指标均值", formatMetricAverages(summary.MetricAverages))
	writePDFSection(doc, "常见问题", formatFindingCounts(summary.FindingCounts))
	return pdfBytes(doc)
}

func renderCourseSummaryPDF(summary CourseReportSummary) ([]byte, error) {
	doc := newReportPDF("course-summary")
	writePDFTitle(doc, "课程统计报表")
	writePDFFacts(doc, [][2]string{
		{"课程ID", summary.CourseID},
		{"实验数", fmt.Sprint(summary.ExperimentCount)},
		{"提交数", fmt.Sprint(summary.SubmissionCount)},
		{"已提交数", fmt.Sprint(summary.SubmittedCount)},
		{"已发布评价", fmt.Sprint(summary.PublishedReviewCount)},
		{"平均分", scorePercent(summary.AverageScoreBPS)},
		{"最高分", scorePercent(summary.MaxScoreBPS)},
		{"最低分", scorePercent(summary.MinScoreBPS)},
	})
	var experimentLines []string
	for _, experiment := range summary.Experiments {
		experimentLines = append(experimentLines, fmt.Sprintf("%s: 提交 %d，已发布 %d，平均分 %s", experiment.ExperimentID, experiment.SubmissionCount, experiment.PublishedReviewCount, scorePercent(experiment.AverageScoreBPS)))
	}
	writePDFSection(doc, "实验对比", strings.Join(experimentLines, "\n"))
	writePDFSection(doc, "分数分布", formatBuckets(summary.ScoreBuckets))
	writePDFSection(doc, "指标均值", formatMetricAverages(summary.MetricAverages))
	writePDFSection(doc, "常见问题", formatFindingCounts(summary.FindingCounts))
	return pdfBytes(doc)
}

func workbookBytes(file *excelize.File) ([]byte, error) {
	defer func() { _ = file.Close() }()
	var b bytes.Buffer
	if err := file.Write(&b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func writeRows(file *excelize.File, sheet string, rows [][]any) {
	for rowIndex, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, rowIndex+1)
		_ = file.SetSheetRow(sheet, cell, &row)
	}
}

func newReportPDF(subject string) *gofpdf.Fpdf {
	doc := gofpdf.New("P", "mm", "A4", "")
	doc.SetMargins(12, 12, 12)
	doc.SetAutoPageBreak(true, 12)
	doc.AddUTF8FontFromBytes("noto", "", reportPDFFont)
	doc.SetCreator("loong64-b1-go", true)
	doc.SetSubject(subject, true)
	doc.AddPage()
	doc.SetFont("noto", "", 16)
	return doc
}

func writePDFTitle(doc *gofpdf.Fpdf, title string) {
	doc.CellFormat(0, 10, title, "", 1, "L", false, 0, "")
	doc.Ln(1)
}

func writePDFFacts(doc *gofpdf.Fpdf, facts [][2]string) {
	doc.SetFont("noto", "", 11)
	for _, fact := range facts {
		doc.MultiCell(0, 6, fact[0]+"： "+sanitizePDFText(fact[1]), "", "L", false)
	}
	doc.Ln(1)
}

func writePDFSection(doc *gofpdf.Fpdf, title, body string) {
	title = sanitizePDFText(title)
	body = sanitizePDFText(body)
	if strings.TrimSpace(body) == "" {
		return
	}
	doc.SetFont("noto", "", 12)
	doc.CellFormat(0, 7, title, "", 1, "L", false, 0, "")
	doc.SetFont("noto", "", 10)
	doc.MultiCell(0, 5, body, "", "L", false)
	doc.Ln(1)
}

func pdfBytes(doc *gofpdf.Fpdf) ([]byte, error) {
	var b bytes.Buffer
	if err := doc.Output(&b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func sanitizePDFText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "\t", " ")
	return strings.TrimSpace(value)
}

func formatBuckets(buckets map[string]int) string {
	var parts []string
	for _, label := range scoreBucketOrder() {
		parts = append(parts, fmt.Sprintf("%s: %d", label, buckets[label]))
	}
	return strings.Join(parts, "\n")
}

func formatMetricAverages(metrics []MetricAverage) string {
	var parts []string
	for _, metric := range metrics {
		parts = append(parts, fmt.Sprintf("%s: %d/%d, %s, 样本 %d", metric.MetricCode, metric.AverageScore, metric.MaxScore, scorePercent(metric.AveragePercentBPS), metric.Count))
	}
	return strings.Join(parts, "\n")
}

func formatFindingCounts(findings []FindingCount) string {
	var parts []string
	for _, finding := range findings {
		parts = append(parts, fmt.Sprintf("%s/%s: %d", finding.Severity, finding.Category, finding.Count))
	}
	return strings.Join(parts, "\n")
}
