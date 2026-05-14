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

...omitted unchanged prefix...

export interface EvaluationResultDetail {
  result: EvaluationResult;
  log?: LLMCallLog;
  findings: RuleCheckFinding[];
  scores: MetricScore[];
}

export interface LLMCallLog {
  id: string;
  provider: string;
  model?: string;
  prompt_version: string;
  status: string;
  output: Record<string, unknown>;
}

...omitted unchanged remainder...
