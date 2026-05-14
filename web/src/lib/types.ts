export type ActorRole = 'admin' | 'teacher' | 'student';

...unchanged prefix...

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

...unchanged remainder...
