export type ActorRole = 'admin' | 'teacher' | 'student';

export interface ActorProfile {
  id: string;
  roles: ActorRole[];
}

export interface Submission {
  id: string;
  experiment_id: string;
  student_id: string;
  status: string;
  attempt_no: number;
  submitted_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Artifact {
  id: string;
  submission_id: string;
  kind: string;
  original_name: string;
  content_type?: string;
  byte_size: number;
  status: string;
  source_url?: string;
  created_at: string;
}

export interface ExtractedContent {
  id: string;
  artifact_id: string;
  status: string;
  text_excerpt?: string;
  error?: string;
}

export interface ArtifactWithExtraction {
  artifact: Artifact;
  extraction: ExtractedContent;
  job_id?: string;
}

export interface SubmissionDetail {
  submission: Submission;
  artifacts: ArtifactWithExtraction[];
}

export interface EvaluationResult {
  id: string;
  submission_id: string;
  status: string;
  rule_status: string;
  llm_status: string;
  prompt_version: string;
  llm_summary?: string;
  needs_teacher_review: boolean;
  created_at: string;
}

export interface RuleCheckFinding {
  id: string;
  category: string;
  severity: 'info' | 'low' | 'medium' | 'high' | 'critical';
  message: string;
  evidence_ref?: string;
}

export interface MetricScore {
  id: string;
  metric_id: string;
  metric_code: string;
  source: 'rule' | 'llm';
  suggested_score: number;
  max_score: number;
  confidence_bps: number;
  rationale: string;
}

export interface EvaluationResultDetail {
  result: EvaluationResult;
  findings: RuleCheckFinding[];
  scores: MetricScore[];
}

export interface TeacherMetricScore {
  id: string;
  metric_id: string;
  metric_code: string;
  final_score: number;
  max_score: number;
  weight_bps: number;
  source: 'manual' | 'rule' | 'llm';
  source_metric_score_id?: string;
  comment?: string;
  adjustment_reason?: string;
}

export interface TeacherReview {
  id: string;
  submission_id: string;
  evaluation_result_id?: string;
  status: 'draft' | 'published';
  total_score_bps: number;
  teacher_comment?: string;
  published_at?: string;
}

export interface TeacherReviewDetail {
  review: TeacherReview;
  scores: TeacherMetricScore[];
  ai?: EvaluationResultDetail;
}

export interface SubmissionReport {
  submission: Submission;
  experiment: {
    id: string;
    title: string;
    rubric_version_id: string;
  };
  artifacts: ArtifactWithExtraction[];
  review: TeacherReviewDetail;
  evaluation?: EvaluationResultDetail;
  generated_at: string;
}

export interface MetricAverage {
  metric_code: string;
  average_score: number;
  average_percent_bps: number;
  max_score: number;
  count: number;
}

export interface FindingCount {
  category: string;
  severity: 'info' | 'low' | 'medium' | 'high' | 'critical';
  count: number;
}

export interface ExperimentReportSummary {
  experiment_id: string;
  submission_count: number;
  submitted_count: number;
  published_review_count: number;
  average_score_bps: number;
  min_score_bps: number;
  max_score_bps: number;
  score_buckets: Record<string, number>;
  submission_status_count: Record<string, number>;
  artifact_status_count: Record<string, number>;
  metric_averages: MetricAverage[];
  finding_counts: FindingCount[];
  generated_at: string;
}

export interface ReportExport {
  id: string;
  report_type: 'submission_report' | 'experiment_summary';
  scope_type: 'submission' | 'experiment';
  scope_id: string;
  format: 'html' | 'csv' | 'pdf';
  status: 'queued' | 'running' | 'succeeded' | 'failed';
  storage_key?: string;
  sha256_hex?: string;
  byte_size: number;
  error?: string;
  requested_by: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

export interface APIErrorBody {
  error?: {
    code?: string;
    message?: string;
  };
}
