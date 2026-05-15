import type {
  ActorProfile,
  AssistantConfirmToolResult,
  AssistantConversation,
  AssistantConversationDetail,
  AssistantSendMessageResult,
  AssistantToolCall,
  BootstrapCreateAdminResponse,
  ClassRecord,
  CourseRecord,
  BootstrapStatus,
  CourseReportSummary,
  ExperimentReportSummary,
  ExperimentRecord,
  EvaluationResultDetail,
  EvaluationJob,
  MetricRecord,
  ReportExport,
  RubricTemplateRecord,
  RubricVersionRecord,
  RubricVersionWithMetrics,
  RuntimeConfigSummary,
  Submission,
  SubmissionDetail,
  SubmissionReport,
  TeacherReviewDetail,
} from './types';

export interface RequestOptions extends RequestInit {
  actorID?: string;
  roles?: string[];
}

const csrfCookieName = import.meta.env.VITE_CSRF_COOKIE_NAME ?? 'loong64_b1_csrf';

export class APIClient {
  constructor(private readonly baseURL = '') {}

  async me(options: RequestOptions = {}): Promise<ActorProfile> {
    return this.request('/api/v1/me', options);
  }

  async getBootstrapStatus(): Promise<BootstrapStatus> {
    return this.request('/api/v1/bootstrap/status', { actorID: '', roles: [] });
  }

  async bootstrapCreateAdmin(payload: { username: string; display_name: string; email?: string; employee_no?: string; password: string }): Promise<BootstrapCreateAdminResponse> {
    return this.request('/api/v1/bootstrap/admin', {
      actorID: '',
      roles: [],
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async login(payload: { username: string; password: string }): Promise<ActorProfile> {
    return this.request('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async logout(): Promise<void> {
    return this.request('/api/v1/auth/logout', {
      method: 'POST',
    });
  }

  async changeOwnPassword(payload: { current_password: string; new_password: string }): Promise<void> {
    return this.request('/api/v1/auth/password', {
      method: 'PUT',
      body: JSON.stringify(payload),
    });
  }

  async createBootstrapAssistantConversation(): Promise<AssistantConversation> {
    return this.request('/api/v1/bootstrap/assistant/conversations', { actorID: '', roles: [], method: 'POST' });
  }

  async getBootstrapAssistantConversation(conversationID: string): Promise<AssistantConversationDetail> {
    return this.request(`/api/v1/bootstrap/assistant/conversations/${encodeURIComponent(conversationID)}`, { actorID: '', roles: [] });
  }

  async sendBootstrapAssistantMessage(conversationID: string, content: string): Promise<AssistantSendMessageResult> {
    return this.request(`/api/v1/bootstrap/assistant/conversations/${encodeURIComponent(conversationID)}/messages`, {
      actorID: '',
      roles: [],
      method: 'POST',
      body: JSON.stringify({ content }),
    });
  }

  async confirmBootstrapAssistantToolCall(toolCallID: string, inputs: Record<string, unknown>): Promise<AssistantConfirmToolResult> {
    return this.request(`/api/v1/bootstrap/assistant/tool-calls/${encodeURIComponent(toolCallID)}/confirm`, {
      actorID: '',
      roles: [],
      method: 'POST',
      body: JSON.stringify({ inputs }),
    });
  }

  async createSubmission(experimentID: string, options: RequestOptions): Promise<Submission> {
    return this.request(`/api/v1/student/experiments/${encodeURIComponent(experimentID)}/submissions`, {
      ...options,
      method: 'POST',
      body: JSON.stringify({}),
    });
  }

  async listUsers(options: RequestOptions): Promise<{ items: ActorProfile[] }> {
    return this.request('/api/v1/admin/users', options);
  }

  async listClasses(options: RequestOptions): Promise<{ items: ClassRecord[] }> {
    return this.request('/api/v1/admin/classes', options);
  }

  async listCourses(options: RequestOptions): Promise<{ items: CourseRecord[] }> {
    return this.request('/api/v1/admin/courses', options);
  }

  async createUser(payload: { username: string; display_name: string; email?: string; student_no?: string; employee_no?: string; password?: string; roles: string[] }, options: RequestOptions): Promise<ActorProfile> {
    return this.request('/api/v1/admin/users', {
      ...options,
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async setUserPassword(userID: string, password: string, options: RequestOptions): Promise<void> {
    return this.request(`/api/v1/admin/users/${encodeURIComponent(userID)}/password`, {
      ...options,
      method: 'PUT',
      body: JSON.stringify({ password }),
    });
  }

  async createClass(payload: { code: string; name: string; grade_year?: number; major?: string }, options: RequestOptions): Promise<ClassRecord> {
    return this.request('/api/v1/admin/classes', {
      ...options,
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async createCourse(payload: { code: string; name: string; term: string }, options: RequestOptions): Promise<CourseRecord> {
    return this.request('/api/v1/admin/courses', {
      ...options,
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async addCourseClass(courseID: string, classID: string, options: RequestOptions): Promise<void> {
    return this.request(`/api/v1/admin/courses/${encodeURIComponent(courseID)}/classes`, {
      ...options,
      method: 'PUT',
      body: JSON.stringify({ class_id: classID }),
    });
  }

  async assignTeacher(courseID: string, teacherID: string, options: RequestOptions): Promise<void> {
    return this.request(`/api/v1/admin/courses/${encodeURIComponent(courseID)}/teachers`, {
      ...options,
      method: 'PUT',
      body: JSON.stringify({ teacher_id: teacherID }),
    });
  }

  async enrollStudent(courseID: string, payload: { student_id: string; class_id?: string }, options: RequestOptions): Promise<void> {
    return this.request(`/api/v1/admin/courses/${encodeURIComponent(courseID)}/enrollments`, {
      ...options,
      method: 'PUT',
      body: JSON.stringify(payload),
    });
  }

  async createRubricTemplate(payload: { name: string; description?: string }, options: RequestOptions): Promise<RubricTemplateRecord> {
    return this.request('/api/v1/teacher/rubric-templates', {
      ...options,
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async listTeacherCourses(options: RequestOptions): Promise<{ items: CourseRecord[] }> {
    return this.request('/api/v1/teacher/courses', options);
  }

  async listRubricTemplates(options: RequestOptions): Promise<{ items: RubricTemplateRecord[] }> {
    return this.request('/api/v1/teacher/rubric-templates', options);
  }

  async listRubricVersions(templateID: string, options: RequestOptions): Promise<{ items: RubricVersionRecord[] }> {
    return this.request(`/api/v1/teacher/rubric-templates/${encodeURIComponent(templateID)}/versions`, options);
  }

  async createRubricVersion(templateID: string, payload: { weight_mode: string; metrics: Array<{ code: string; name: string; description?: string; weight_bps: number; max_score: number; sort_order: number }> }, options: RequestOptions): Promise<RubricVersionWithMetrics> {
    return this.request(`/api/v1/teacher/rubric-templates/${encodeURIComponent(templateID)}/versions`, {
      ...options,
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async publishRubricVersion(versionID: string, options: RequestOptions): Promise<RubricVersionRecord> {
    return this.request(`/api/v1/teacher/rubric-template-versions/${encodeURIComponent(versionID)}/publish`, {
      ...options,
      method: 'POST',
      body: JSON.stringify({}),
    });
  }

  async createExperiment(courseID: string, payload: { title: string; description?: string; rubric_version_id: string; submission_spec?: Record<string, unknown> }, options: RequestOptions): Promise<ExperimentRecord> {
    return this.request(`/api/v1/teacher/courses/${encodeURIComponent(courseID)}/experiments`, {
      ...options,
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async publishExperiment(experimentID: string, options: RequestOptions): Promise<ExperimentRecord> {
    return this.request(`/api/v1/teacher/experiments/${encodeURIComponent(experimentID)}/publish`, {
      ...options,
      method: 'POST',
      body: JSON.stringify({}),
    });
  }

  async getSubmission(submissionID: string, role: 'teacher' | 'student', options: RequestOptions): Promise<SubmissionDetail> {
    return this.request(`/api/v1/${role}/submissions/${encodeURIComponent(submissionID)}`, options);
  }

  async listTeacherExperiments(courseID: string, options: RequestOptions): Promise<{ items: ExperimentRecord[] }> {
    return this.request(`/api/v1/teacher/courses/${encodeURIComponent(courseID)}/experiments`, options);
  }

  async listStudentExperiments(options: RequestOptions): Promise<{ items: ExperimentRecord[] }> {
    return this.request('/api/v1/student/experiments', options);
  }

  async listStudentSubmissions(experimentID: string, options: RequestOptions): Promise<{ items: Submission[] }> {
    const suffix = experimentID ? `?experiment_id=${encodeURIComponent(experimentID)}` : '';
    return this.request(`/api/v1/student/submissions${suffix}`, options);
  }

  async uploadArtifact(submissionID: string, file: File, artifactKind: string, options: RequestOptions): Promise<unknown> {
    const form = new FormData();
    form.append('file', file);
    form.append('artifact_kind', artifactKind);
    return this.request(`/api/v1/student/submissions/${encodeURIComponent(submissionID)}/artifacts`, {
      ...options,
      method: 'POST',
      body: form,
    });
  }

  async createGitLink(submissionID: string, url: string, commitSHA: string, note: string, options: RequestOptions): Promise<unknown> {
    return this.request(`/api/v1/student/submissions/${encodeURIComponent(submissionID)}/artifact-links`, {
      ...options,
      method: 'POST',
      body: JSON.stringify({
        url,
        commit_sha: commitSHA || undefined,
        note: note || undefined,
      }),
    });
  }

  async listExperimentSubmissions(experimentID: string, options: RequestOptions): Promise<{ items: Submission[] }> {
    return this.request(`/api/v1/teacher/experiments/${encodeURIComponent(experimentID)}/submissions`, options);
  }

  async createEvaluation(submissionID: string, mode: 'rule_only' | 'rule_and_llm', options: RequestOptions): Promise<EvaluationJob> {
    return this.request(`/api/v1/teacher/submissions/${encodeURIComponent(submissionID)}/evaluations/initial`, {
      ...options,
      method: 'POST',
      body: JSON.stringify({ mode }),
    });
  }

  async getEvaluationJob(jobID: string, options: RequestOptions): Promise<EvaluationJob> {
    return this.request(`/api/v1/teacher/evaluations/jobs/${encodeURIComponent(jobID)}`, options);
  }

  async getLatestEvaluation(submissionID: string, options: RequestOptions): Promise<EvaluationResultDetail> {
    return this.request(`/api/v1/teacher/submissions/${encodeURIComponent(submissionID)}/evaluations/latest`, options);
  }

  async saveTeacherReview(submissionID: string, payload: unknown, options: RequestOptions): Promise<TeacherReviewDetail> {
    return this.request(`/api/v1/teacher/submissions/${encodeURIComponent(submissionID)}/review`, {
      ...options,
      method: 'PUT',
      body: JSON.stringify(payload),
    });
  }

  async publishTeacherReview(submissionID: string, options: RequestOptions): Promise<TeacherReviewDetail> {
    return this.request(`/api/v1/teacher/submissions/${encodeURIComponent(submissionID)}/review/publish`, {
      ...options,
      method: 'POST',
      body: JSON.stringify({ confirm: true }),
    });
  }

  async getTeacherReview(submissionID: string, role: 'teacher' | 'student', options: RequestOptions): Promise<TeacherReviewDetail> {
    return this.request(`/api/v1/${role}/submissions/${encodeURIComponent(submissionID)}/review`, options);
  }

  async getSubmissionReport(submissionID: string, role: 'teacher' | 'student', options: RequestOptions): Promise<SubmissionReport> {
    return this.request(`/api/v1/${role}/submissions/${encodeURIComponent(submissionID)}/report`, options);
  }

  async getExperimentReportSummary(experimentID: string, options: RequestOptions): Promise<ExperimentReportSummary> {
    return this.request(`/api/v1/teacher/experiments/${encodeURIComponent(experimentID)}/reports/summary`, options);
  }

  async createSubmissionReportExport(submissionID: string, format: 'html' | 'csv' | 'xlsx' | 'pdf', options: RequestOptions): Promise<ReportExport> {
    return this.request(`/api/v1/teacher/submissions/${encodeURIComponent(submissionID)}/report-exports`, {
      ...options,
      method: 'POST',
      body: JSON.stringify({ format }),
    });
  }

  async createExperimentSummaryExport(experimentID: string, format: 'html' | 'csv' | 'xlsx' | 'pdf', options: RequestOptions): Promise<ReportExport> {
    return this.request(`/api/v1/teacher/experiments/${encodeURIComponent(experimentID)}/report-exports`, {
      ...options,
      method: 'POST',
      body: JSON.stringify({ format }),
    });
  }

  async getCourseReportSummary(courseID: string, options: RequestOptions): Promise<CourseReportSummary> {
    return this.request(`/api/v1/teacher/courses/${encodeURIComponent(courseID)}/reports/summary`, options);
  }

  async createCourseSummaryExport(courseID: string, format: 'html' | 'csv' | 'xlsx' | 'pdf', options: RequestOptions): Promise<ReportExport> {
    return this.request(`/api/v1/teacher/courses/${encodeURIComponent(courseID)}/report-exports`, {
      ...options,
      method: 'POST',
      body: JSON.stringify({ format }),
    });
  }

  async getRuntimeConfig(options: RequestOptions): Promise<RuntimeConfigSummary> {
    return this.request('/api/v1/admin/runtime-config', options);
  }

  async updateRuntimeConfig(payload: { db_driver: 'sqlite' | 'postgres'; sqlite_path?: string; database_url?: string; auto_migrate?: boolean }, options: RequestOptions): Promise<RuntimeConfigSummary> {
    return this.request('/api/v1/admin/runtime-config', {
      ...options,
      method: 'PUT',
      body: JSON.stringify(payload),
    });
  }

  async createDeploymentAssistantConversation(options: RequestOptions): Promise<AssistantConversation> {
    return this.request('/api/v1/admin/deployment-assistant/conversations', {
      ...options,
      method: 'POST',
    });
  }

  async getDeploymentAssistantConversation(conversationID: string, options: RequestOptions): Promise<AssistantConversationDetail> {
    return this.request(`/api/v1/admin/deployment-assistant/conversations/${encodeURIComponent(conversationID)}`, options);
  }

  async sendDeploymentAssistantMessage(conversationID: string, content: string, options: RequestOptions): Promise<AssistantSendMessageResult> {
    return this.request(`/api/v1/admin/deployment-assistant/conversations/${encodeURIComponent(conversationID)}/messages`, {
      ...options,
      method: 'POST',
      body: JSON.stringify({ content }),
    });
  }

  async confirmDeploymentAssistantToolCall(toolCallID: string, inputs: Record<string, unknown>, options: RequestOptions): Promise<AssistantConfirmToolResult> {
    return this.request(`/api/v1/admin/deployment-assistant/tool-calls/${encodeURIComponent(toolCallID)}/confirm`, {
      ...options,
      method: 'POST',
      body: JSON.stringify({ inputs }),
    });
  }

  reportExportDownloadURL(exportID: string): string {
    return `${this.baseURL}/api/v1/teacher/report-exports/${encodeURIComponent(exportID)}/download`;
  }

  private async request<T>(path: string, options: RequestOptions): Promise<T> {
    const headers = new Headers(options.headers);
    const method = (options.method ?? 'GET').toUpperCase();
    if (options.actorID) {
      headers.set('X-Actor-ID', options.actorID);
    }
    if ((options.roles ?? []).length > 0) {
      headers.set('X-Actor-Roles', (options.roles ?? []).join(','));
    }
    if (options.body && !(options.body instanceof FormData)) {
      headers.set('Content-Type', 'application/json');
    }
    if (csrfProtectedMethod(method)) {
      const csrfToken = readCookie(csrfCookieName);
      if (csrfToken !== '') {
        headers.set('X-CSRF-Token', csrfToken);
      }
    }
    const response = await fetch(`${this.baseURL}${path}`, { ...options, headers });
    if (!response.ok) {
      const body = (await response.json().catch(() => ({}))) as { error?: { message?: string } };
      throw new Error(body.error?.message ?? `${response.status} ${response.statusText}`);
    }
    if (response.status === 204) {
      return undefined as T;
    }
    return (await response.json()) as T;
  }
}

function csrfProtectedMethod(method: string): boolean {
  return method === 'POST' || method === 'PUT' || method === 'PATCH' || method === 'DELETE';
}

function readCookie(name: string): string {
  if (typeof document === 'undefined' || document.cookie === '') {
    return '';
  }
  const escaped = name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const match = document.cookie.match(new RegExp(`(?:^|; )${escaped}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : '';
}

export const api = new APIClient(import.meta.env.VITE_API_BASE_URL ?? '');
