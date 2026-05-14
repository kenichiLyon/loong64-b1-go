import type { ArtifactWithExtraction, EvaluationResultDetail, SubmissionDetail } from './types';

export interface EvidenceSnippet {
  ref: string;
  title: string;
  text: string;
}

export interface ArtifactEvidenceOutline {
  artifactID: string;
  artifactName: string;
  sections: EvidenceSnippet[];
  evidence: EvidenceSnippet[];
}

type MetadataEntry = Record<string, unknown>;

export function resolveEvidenceSnippets(detail: SubmissionDetail | null, refs: string[]): EvidenceSnippet[] {
  if (!detail) {
    return [];
  }
  const snippets: EvidenceSnippet[] = [];
  for (const ref of refs) {
    const trimmed = ref.trim();
    if (trimmed === '') {
      continue;
    }
    const snippet = resolveEvidenceSnippet(detail, trimmed);
    if (snippet) {
      snippets.push(snippet);
    }
  }
  return snippets;
}

export function resolveArtifactEvidenceOutlines(detail: SubmissionDetail | null): ArtifactEvidenceOutline[] {
  if (!detail) {
    return [];
  }
  return detail.artifacts.map((item) => ({
    artifactID: item.artifact.id,
    artifactName: item.artifact.original_name,
    sections: resolveArtifactEntries(item, 'section'),
    evidence: resolveArtifactEntries(item, 'evidence'),
  })).filter((item) => item.sections.length > 0 || item.evidence.length > 0);
}

export function collectEvaluationEvidenceRefs(evaluation: EvaluationResultDetail | null): string[] {
  if (!evaluation) {
    return [];
  }
  const refs = new Set<string>();
  for (const score of evaluation.scores) {
    for (const ref of score.evidence_refs ?? []) {
      const trimmed = ref.trim();
      if (trimmed !== '') {
        refs.add(trimmed);
      }
    }
  }
  for (const finding of evaluation.findings) {
    const trimmed = finding.evidence_ref?.trim();
    if (trimmed) {
      refs.add(trimmed);
    }
  }
  return Array.from(refs);
}

function resolveEvidenceSnippet(detail: SubmissionDetail, ref: string): EvidenceSnippet | null {
  const [artifactPart, fragment] = ref.split('#', 2);
  if (!artifactPart.startsWith('artifact:')) {
    return null;
  }
  const artifactID = artifactPart.slice('artifact:'.length);
  const item = detail.artifacts.find((candidate) => candidate.artifact.id === artifactID);
  if (!item) {
    return null;
  }
  if (!fragment) {
    const text = firstNonEmpty(item.extraction.text_excerpt, item.extraction.error);
    if (text === '') {
      return null;
    }
    return {
      ref,
      title: item.artifact.original_name,
      text,
    };
  }
  const [kind, rawIndex] = fragment.split(':', 2);
  const index = Number.parseInt(rawIndex ?? '', 10);
  if (!Number.isInteger(index) || index < 1) {
    return null;
  }
  const metadata = item.artifact.metadata;
  const entries = asEntryList(kind === 'section' ? metadata?.sections : kind === 'evidence' ? metadata?.evidence : undefined);
  const entry = entries[index - 1];
  if (!entry) {
    return null;
  }
  const title = firstNonEmpty(
    asText(entry.title),
    asText(entry.heading),
    asText(entry.name),
    asText(entry.label),
    `${item.artifact.original_name} · ${kind} ${index}`,
  );
  const text = firstNonEmpty(
    asText(entry.content),
    asText(entry.text),
    asText(entry.excerpt),
    asText(entry.body),
    asText(entry.summary),
  );
  if (text === '') {
    return null;
  }
  return { ref, title, text };
}

function resolveArtifactEntries(item: ArtifactWithExtraction, kind: 'section' | 'evidence'): EvidenceSnippet[] {
  const entries = asEntryList(kind === 'section' ? item.artifact.metadata?.sections : item.artifact.metadata?.evidence);
  return entries.map((entry, index) => {
    const title = firstNonEmpty(
      asText(entry.title),
      asText(entry.heading),
      asText(entry.name),
      asText(entry.label),
      `${item.artifact.original_name} · ${kind} ${index + 1}`,
    );
    const text = firstNonEmpty(
      asText(entry.content),
      asText(entry.text),
      asText(entry.excerpt),
      asText(entry.body),
      asText(entry.summary),
    );
    return text === ''
      ? null
      : {
          ref: `artifact:${item.artifact.id}#${kind}:${index + 1}`,
          title,
          text,
        };
  }).filter((entry): entry is EvidenceSnippet => entry !== null);
}

function asEntryList(value: unknown): MetadataEntry[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((entry): entry is MetadataEntry => typeof entry === 'object' && entry !== null);
}

function asText(value: unknown): string {
  return typeof value === 'string' ? value.trim() : '';
}

function firstNonEmpty(...values: Array<string | undefined>): string {
  for (const value of values) {
    if (value && value.trim() !== '') {
      return value.trim();
    }
  }
  return '';
}
