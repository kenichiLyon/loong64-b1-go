import type {
  ActorProfile,
  BootstrapCreateAdminResponse,
  BootstrapStatus,
  CourseReportSummary,
  ExperimentReportSummary,
  EvaluationResultDetail,
  ReportExport,
  RuntimeConfigSummary,
  Submission,
  SubmissionDetail,
  SubmissionReport,
  TeacherReviewDetail,
} from './types';

export interface RequestOptions extends RequestInit {
  actorID: string;
  roles: string[];
}

export class APIClient {
  constructor(private readonly baseURL = '') {}

  async me(options: RequestOptions): Promise<ActorProfile> {
    return this.request('/api/v1/me', options);
  }

  async getBootstrapStatus(): Promise<BootstrapStatus> {
    return this.request('/api/v1/bootstrap/status', { actorID: '', roles: [] });
  }

  async bootstrapCreateAdmin(payload: { username: string; display_name: string; email?: string; employee_no?: string }): Promise<BootstrapCreateAdminResponse> {
    return this.request('/api/v1/bootstrap/admin', {
      actorID: '',
      roles: [],
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async createSubmission(experimentID: string, options: RequestOptions): Promise<Submission> {
    return this.request(`/api/v1/student/experiments/${encodeURIComponent(experimentID)}/submissions`, {
      ...options,
      method: 'POST',
      body: JSON.stringify({}),
    });
  }

  async getSubmission(submissionID: string, role: 'teacher' | 'student', options: RequestOptions): Promise<SubmissionDetail> {
    return this.request(`/api/v1/${role}/submissions/${encodeURIComponent(submissionID)}`, options);
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

  async createEvaluation(submissionID: string, mode: 'rule_only' | 'rule_and_llm', options: RequestOptions): Promise<EvaluationResultDetail> {
    return this.request(`/api/v1/teacher/submissions/${encodeURIComponent(submissionID)}/evaluations/initial`, {
      ...options,
      method: 'POST',
      body: JSON.stringify({ mode }),
    });
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

  async createSubmissionReportExport(submissionID: string, format: 'html' | 'csv' | 'pdf', options: RequestOptions): Promise<ReportExport> {
    return this.request(`/api/v1/teacher/submissions/${encodeURIComponent(submissionID)}/report-exports`, {
      ...options,
      method: 'POST',
      body: JSON.stringify({ format }),
    });
  }

  async createExperimentSummaryExport(experimentID: string, format: 'html' | 'csv' | 'pdf', options: RequestOptions): Promise<ReportExport> {
    return this.request(`/api/v1/teacher/experiments/${encodeURIComponent(experimentID)}/report-exports`, {
      ...options,
      method: 'POST',
      body: JSON.stringify({ format }),
    });
  }

  async getCourseReportSummary(courseID: string, options: RequestOptions): Promise<CourseReportSummary> {
    return this.request(`/api/v1/teacher/courses/${encodeURIComponent(courseID)}/reports/summary`, options);
  }

  async createCourseSummaryExport(courseID: string, format: 'html' | 'csv' | 'pdf', options: RequestOptions): Promise<ReportExport> {
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

  reportExportDownloadURL(exportID: string): string {
    return `${this.baseURL}/api/v1/teacher/report-exports/${encodeURIComponent(exportID)}/download`;
  }

  private async request<T>(path: string, options: RequestOptions): Promise<T> {
    const headers = new Headers(options.headers);
    if (options.actorID) {
      headers.set('X-Actor-ID', options.actorID);
    }
    if (options.roles.length > 0) {
      headers.set('X-Actor-Roles', options.roles.join(','));
    }
    if (options.body && !(options.body instanceof FormData)) {
      headers.set('Content-Type', 'application/json');
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

export const api = new APIClient(import.meta.env.VITE_API_BASE_URL ?? '');
