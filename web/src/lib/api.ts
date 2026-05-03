import type {
  ActorProfile,
  EvaluationResultDetail,
  Submission,
  SubmissionDetail,
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

  private async request<T>(path: string, options: RequestOptions): Promise<T> {
    const headers = new Headers(options.headers);
    headers.set('X-Actor-ID', options.actorID);
    headers.set('X-Actor-Roles', options.roles.join(','));
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
