package teaching

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const maxEvidenceExcerpt = 600

var sensitivePattern = regexp.MustCompile(`(?i)(api[_-]?key|secret|password|token)\s*[:=]\s*[^\s,;]+`)

func EvaluateRules(ctx EvaluationContext, evaluationID string) ([]RuleCheckFinding, []MetricScore, json.RawMessage, json.RawMessage) {
	refs, snapshot := buildEvidenceSnapshot(ctx)
	spec := parseSubmissionSpec(ctx.Experiment.SubmissionSpec)
	findings := collectRuleFindings(ctx, spec, refs)
	summary := summarizeFindings(findings)
	scores := buildRuleMetricScores(evaluationID, ctx.Metrics, findings, refs)
	for i := range findings {
		findings[i].EvaluationResultID = evaluationID
	}
	return findings, scores, snapshot, summary
}

func parseSubmissionSpec(raw json.RawMessage) SubmissionSpec {
	var spec SubmissionSpec
	if len(raw) == 0 || !json.Valid(raw) {
		return spec
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		return SubmissionSpec{}
	}
	spec.RequiredArtifacts = normalizeStringList(spec.RequiredArtifacts)
	spec.RequiredSections = normalizeStringList(spec.RequiredSections)
	spec.RequiredSteps = normalizeStringList(spec.RequiredSteps)
	spec.Keywords = normalizeStringList(spec.Keywords)
	return spec
}

func buildEvidenceSnapshot(ctx EvaluationContext) (map[string]string, json.RawMessage) {
	type artifactEvidence struct {
		Ref              string `json:"ref"`
		ArtifactID       string `json:"artifact_id"`
		Kind             string `json:"kind"`
		OriginalName     string `json:"original_name"`
		ContentType      string `json:"content_type,omitempty"`
		ByteSize         int64  `json:"byte_size"`
		Status           string `json:"status"`
		ExtractionStatus string `json:"extraction_status"`
		HasText          bool   `json:"has_text"`
		SourceURLPresent bool   `json:"source_url_present"`
		Excerpt          string `json:"excerpt,omitempty"`
	}
	type snapshot struct {
		SubmissionID string             `json:"submission_id"`
		ExperimentID string             `json:"experiment_id"`
		Artifacts    []artifactEvidence `json:"artifacts"`
	}
	refs := make(map[string]string)
	artifacts := make([]artifactEvidence, 0, len(ctx.Artifacts))
	for i, item := range ctx.Artifacts {
		ref := fmt.Sprintf("artifact:%s", item.Artifact.ID)
		if item.Artifact.ID == "" {
			ref = fmt.Sprintf("artifact:%d", i+1)
		}
		excerpt := sanitizeEvidenceText(item.Extraction.TextExcerpt, maxEvidenceExcerpt)
		refs[ref] = excerpt
		artifacts = append(artifacts, artifactEvidence{
			Ref:              ref,
			ArtifactID:       item.Artifact.ID,
			Kind:             string(item.Artifact.Kind),
			OriginalName:     item.Artifact.OriginalName,
			ContentType:      item.Artifact.ContentType,
			ByteSize:         item.Artifact.ByteSize,
			Status:           item.Artifact.Status,
			ExtractionStatus: item.Extraction.Status,
			HasText:          strings.TrimSpace(item.Extraction.TextExcerpt) != "",
			SourceURLPresent: item.Artifact.SourceURL != "",
			Excerpt:          excerpt,
		})
	}
	payload := snapshot{SubmissionID: ctx.Submission.ID, ExperimentID: ctx.Experiment.ID, Artifacts: artifacts}
	return refs, mustJSON(payload)
}

func collectRuleFindings(ctx EvaluationContext, spec SubmissionSpec, refs map[string]string) []RuleCheckFinding {
	findings := make([]RuleCheckFinding, 0)
	if len(ctx.Artifacts) == 0 {
		findings = append(findings, newFinding("completeness", FindingHigh, "submission has no artifacts", ""))
		return findings
	}
	kindCounts := make(map[ArtifactKind]int)
	combinedText := strings.ToLower(combinedEvidenceText(ctx.Artifacts))
	hasText := strings.TrimSpace(combinedText) != ""
	for _, item := range ctx.Artifacts {
		kindCounts[item.Artifact.Kind]++
		ref := "artifact:" + item.Artifact.ID
		switch item.Extraction.Status {
		case "queued", "running":
			findings = append(findings, newFinding("parsing", FindingMedium, fmt.Sprintf("artifact %s has extraction status %s", item.Artifact.OriginalName, item.Extraction.Status), ref))
		case "failed":
			findings = append(findings, newFinding("parsing", FindingHigh, fmt.Sprintf("artifact %s extraction failed", item.Artifact.OriginalName), ref))
		}
		if (item.Artifact.Kind == ArtifactKindReport || item.Artifact.Kind == ArtifactKindDocument) && strings.TrimSpace(item.Extraction.TextExcerpt) == "" {
			findings = append(findings, newFinding("evidence", FindingMedium, fmt.Sprintf("%s artifact has no text excerpt yet", item.Artifact.Kind), ref))
		}
	}
	for _, required := range spec.RequiredArtifacts {
		kind := ArtifactKind(normalizeCode(required))
		if kindCounts[kind] == 0 {
			findings = append(findings, newFinding("completeness", FindingHigh, fmt.Sprintf("required artifact kind %s is missing", required), ""))
		}
	}
	if kindCounts[ArtifactKindCodeArchive] == 0 && kindCounts[ArtifactKindGitLink] == 0 {
		severity := FindingLow
		if requiresCodeEvidence(spec.RequiredArtifacts) {
			severity = FindingHigh
		}
		findings = append(findings, newFinding("evidence", severity, "code archive or git link evidence is missing", ""))
	}
	if !hasText && (kindCounts[ArtifactKindReport] > 0 || kindCounts[ArtifactKindDocument] > 0) {
		findings = append(findings, newFinding("evidence", FindingMedium, "no extracted text is available for document/report review", firstRef(refs)))
	}
	for _, section := range spec.RequiredSections {
		if hasText && !strings.Contains(combinedText, strings.ToLower(section)) {
			findings = append(findings, newFinding("completeness", FindingMedium, fmt.Sprintf("required report section %s is not found in excerpts", section), firstRef(refs)))
		}
	}
	for _, step := range spec.RequiredSteps {
		if hasText && !strings.Contains(combinedText, strings.ToLower(step)) {
			findings = append(findings, newFinding("steps", FindingMedium, fmt.Sprintf("required step %s is not covered by extracted evidence", step), firstRef(refs)))
		}
	}
	for _, keyword := range spec.Keywords {
		if hasText && !strings.Contains(combinedText, strings.ToLower(keyword)) {
			findings = append(findings, newFinding("logic", FindingLow, fmt.Sprintf("expected keyword %s is not found in evidence excerpts", keyword), firstRef(refs)))
		}
	}
	if containsPromptInjection(combinedText) {
		findings = append(findings, newFinding("security", FindingHigh, "evidence contains prompt-injection-like instructions", firstRef(refs)))
	}
	if sensitivePattern.MatchString(combinedText) {
		findings = append(findings, newFinding("security", FindingMedium, "evidence may contain secrets or credentials and needs manual review", firstRef(refs)))
	}
	return findings
}

func buildRuleMetricScores(evaluationID string, metrics []Metric, findings []RuleCheckFinding, refs map[string]string) []MetricScore {
	scores := make([]MetricScore, 0, len(metrics))
	for _, metric := range metrics {
		penalty := 0
		for _, finding := range findings {
			penalty += findingPenalty(metric, finding)
		}
		if penalty > 80 {
			penalty = 80
		}
		score := metric.MaxScore - metric.MaxScore*penalty/100
		if score < 0 {
			score = 0
		}
		confidence := 7000
		if hasParsingRisk(findings) {
			confidence = 5000
		}
		scores = append(scores, MetricScore{
			ID:                 NewID("msc"),
			EvaluationResultID: evaluationID,
			MetricID:           metric.ID,
			MetricCode:         metric.Code,
			Source:             MetricScoreSourceRule,
			SuggestedScore:     score,
			MaxScore:           metric.MaxScore,
			ConfidenceBPS:      confidence,
			Rationale:          ruleRationale(penalty, findings),
			EvidenceRefs:       mustJSON(sortedRefs(refs)),
		})
	}
	return scores
}

func findingPenalty(metric Metric, finding RuleCheckFinding) int {
	base := map[FindingSeverity]int{FindingCritical: 50, FindingHigh: 30, FindingMedium: 15, FindingLow: 5, FindingInfo: 0}[finding.Severity]
	code := strings.ToLower(metric.Code + " " + metric.Name + " " + metric.Description)
	switch finding.Category {
	case "evidence", "parsing":
		if strings.Contains(code, "文档") || strings.Contains(code, "doc") || strings.Contains(code, "report") || strings.Contains(code, "evidence") {
			base += 5
		}
	case "steps", "completeness":
		if strings.Contains(code, "完整") || strings.Contains(code, "complete") || strings.Contains(code, "step") || strings.Contains(code, "功能") {
			base += 5
		}
	case "security", "logic":
		if strings.Contains(code, "质量") || strings.Contains(code, "quality") || strings.Contains(code, "安全") || strings.Contains(code, "logic") {
			base += 5
		}
	}
	return base
}

func summarizeFindings(findings []RuleCheckFinding) json.RawMessage {
	counts := map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0, "info": 0}
	categories := make(map[string]int)
	for _, finding := range findings {
		counts[string(finding.Severity)]++
		categories[finding.Category]++
	}
	return mustJSON(map[string]any{"total": len(findings), "severity_counts": counts, "category_counts": categories})
}

func newFinding(category string, severity FindingSeverity, message, evidenceRef string) RuleCheckFinding {
	return RuleCheckFinding{ID: NewID("vrf"), Category: category, Severity: severity, Message: message, EvidenceRef: evidenceRef}
}

func normalizeStringList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func combinedEvidenceText(artifacts []ArtifactWithExtraction) string {
	var builder strings.Builder
	for _, item := range artifacts {
		if item.Extraction.TextExcerpt == "" {
			continue
		}
		builder.WriteString("\n")
		builder.WriteString(item.Extraction.TextExcerpt)
	}
	return builder.String()
}

func sanitizeEvidenceText(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\x00", "")
	value = sensitivePattern.ReplaceAllString(value, "$1=[REDACTED]")
	if len([]rune(value)) <= maxLen {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxLen]) + "..."
}

func containsPromptInjection(text string) bool {
	markers := []string{"ignore previous instructions", "disregard previous instructions", "system prompt", "developer message", "请忽略", "忽略以上", "忘记之前", "越过系统", "BEGIN SYSTEM"}
	for _, marker := range markers {
		if strings.Contains(text, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

func requiresCodeEvidence(required []string) bool {
	for _, value := range required {
		value = normalizeCode(value)
		if value == string(ArtifactKindCodeArchive) || value == string(ArtifactKindGitLink) || strings.Contains(value, "code") || strings.Contains(value, "git") {
			return true
		}
	}
	return false
}

func firstRef(refs map[string]string) string {
	for _, ref := range sortedRefs(refs) {
		return ref
	}
	return ""
}

func sortedRefs(refs map[string]string) []string {
	out := make([]string, 0, len(refs))
	for ref := range refs {
		out = append(out, ref)
	}
	sort.Strings(out)
	return out
}

func hasParsingRisk(findings []RuleCheckFinding) bool {
	for _, finding := range findings {
		if finding.Category == "parsing" || finding.Category == "evidence" {
			return true
		}
	}
	return false
}

func ruleRationale(penalty int, findings []RuleCheckFinding) string {
	if len(findings) == 0 {
		return "Rule checks did not find material issues in available evidence."
	}
	return fmt.Sprintf("Rule checks applied a %d%% advisory penalty based on %d finding(s); teacher review is still required.", penalty, len(findings))
}
