export type ActorRole = 'admin' | 'teacher' | 'student';

export interface ActorProfile {
  id: string;
  roles: ActorRole[];
  username?: string;
  display_name?: string;
  email?: string;
  student_no?: string;
  employee_no?: string;
  status?: string;
}

export interface ClassRecord {
  id: string;
  code: string;
  name: string;
  grade_year?: number;
  major?: string;
  status: string;
}

export interface CourseRecord {
  id: string;
  code: string;
  name: string;
  term: string;
  status: string;
  created_by?: string;
}

export interface MetricRecord {
  id: string;
  version_id?: string;
  code: string;
  name: string;
  description?: string;
  weight_bps: number;
  max_score: number;
  sort_order: number;
}

export interface RubricTemplateRecord {
  id: string;
  name: string;
  description?: string;
  owner_id: string;
  scope: string;
  status: string;
}

export interface RubricVersionRecord {
  id: string;
  template_id: string;
  version_no: number;
  status: string;
  weight_mode: string;
  total_weight_bps: number;
  published_at?: string;
}

export interface RubricVersionWithMetrics {
  version: RubricVersionRecord;
  metrics: MetricRecord[];
}

export interface ExperimentRecord {
  id: string;
  course_id: string;
  title: string;
  description?: string;
  rubric_version_id: string;
  status: string;
  start_at?: string;
  due_at?: string;
  published_at?: string;
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

export interface CourseReportSummary {
  course_id: string;
  experiment_count: number;
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
  experiments: ExperimentReportSummary[];
  generated_at: string;
}

export interface ReportExport {
  id: string;
  report_type: 'submission_report' | 'experiment_summary' | 'course_summary';
  scope_type: 'submission' | 'experiment' | 'course';
  scope_id: string;
  format: 'html' | 'csv' | 'xlsx' | 'pdf';
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

export interface RuntimeConfigView {
  db_driver: 'sqlite' | 'postgres' | string;
  sqlite_path?: string;
  database_url?: string;
  database_url_set: boolean;
  auto_migrate: boolean;
  requires_restart?: boolean;
  runtime_config_path?: string;
}

export interface RuntimeConfigSummary {
  path: string;
  exists: boolean;
  active: RuntimeConfigView;
  stored?: RuntimeConfigView;
  message?: string;
  error?: string;
}

export interface BootstrapStatus {
  initialized: boolean;
  user_count: number;
  runtime: RuntimeConfigView;
  stored?: RuntimeConfigView;
  message?: string;
}

export interface BootstrapCreateAdminResponse {
  user: ActorProfile & {
    username?: string;
    display_name?: string;
  };
  message: string;
}

export interface AssistantConversation {
  id: string;
  scope_type: 'bootstrap' | 'deployment_admin';
  actor_id?: string;
  status: 'active' | 'closed';
  model?: string;
  prompt_version: string;
  summary_text?: string;
  created_at: string;
  updated_at: string;
  last_message_at: string;
}

export interface AssistantContextSnapshot {
  id: string;
  conversation_id: string;
  scope_stage: 'bootstrap_status' | 'runtime_config' | 'db_connectivity' | 'admin_init';
  payload_json: Record<string, unknown>;
  created_at: string;
}

export interface AssistantToolCall {
  id: string;
  conversation_id: string;
  tool_name:
    | 'inspect_bootstrap_status'
    | 'inspect_runtime_config'
    | 'test_sqlite_path'
    | 'test_postgres_connection'
    | 'save_runtime_config'
    | 'bootstrap_create_admin';
  status: 'pending_confirmation' | 'running' | 'succeeded' | 'failed' | 'cancelled';
  request_json: Record<string, unknown>;
  response_json: Record<string, unknown>;
  error?: string;
  confirmed_by_actor?: string;
  created_at: string;
  completed_at?: string;
}

export interface AssistantMessage {
  id: string;
  conversation_id: string;
  role: 'user' | 'assistant' | 'tool';
  content_text: string;
  context_snapshot_id?: string;
  tool_call_id?: string;
  created_at: string;
}

export interface AssistantConversationDetail {
  conversation: AssistantConversation;
  messages: AssistantMessage[];
  pending_tool_call?: AssistantToolCall;
  latest_context_snapshot?: AssistantContextSnapshot;
}

export interface AssistantSendMessageResult {
  conversation: AssistantConversation;
  assistant_message: AssistantMessage;
  pending_tool_call?: AssistantToolCall;
  context_snapshot: AssistantContextSnapshot;
  requires_confirmation: boolean;
}

export interface AssistantConfirmToolResult {
  conversation: AssistantConversation;
  tool_call: AssistantToolCall;
  tool_message: AssistantMessage;
  assistant_message: AssistantMessage;
}

export interface APIErrorBody {
  error?: {
    code?: string;
    message?: string;
  };
}
