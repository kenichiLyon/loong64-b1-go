<script setup lang="ts">
import { computed } from 'vue';
import { collectEvaluationEvidenceRefs, resolveEvidenceSnippets } from '../lib/evidence';
import type { EvaluationResultDetail, SubmissionDetail } from '../lib/types';
import type { EvidenceSnippet } from '../lib/evidence';

const props = defineProps<{
  evaluation: EvaluationResultDetail | null;
  detail: SubmissionDetail | null;
}>();

const evidenceRefs = computed(() => collectEvaluationEvidenceRefs(props.evaluation));
const evidenceSnippets = computed(() => resolveEvidenceSnippets(props.detail, evidenceRefs.value));
const scoreSnippetsByID = computed<Record<string, EvidenceSnippet[]>>(() => {
  const entries: Record<string, EvidenceSnippet[]> = {};
  for (const score of props.evaluation?.scores ?? []) {
    entries[score.id] = resolveEvidenceSnippets(props.detail, score.evidence_refs ?? []);
  }
  return entries;
});
const findingSnippetsByID = computed<Record<string, EvidenceSnippet[]>>(() => {
  const entries: Record<string, EvidenceSnippet[]> = {};
  for (const finding of props.evaluation?.findings ?? []) {
    entries[finding.id] = finding.evidence_ref ? resolveEvidenceSnippets(props.detail, [finding.evidence_ref]) : [];
  }
  return entries;
});
</script>

<template>
  <section class="card evaluation-panel">
    <div class="panel-heading">
      <p class="eyebrow">智能核查</p>
      <h2>{{ evaluation ? evaluation.result.status : '等待触发' }}</h2>
    </div>

    <div v-if="evaluation" class="evaluation-grid">
      <article class="metric-card" v-for="score in evaluation.scores" :key="score.id">
        <span>{{ score.source }}</span>
        <strong>{{ score.metric_code }}：{{ score.suggested_score }}/{{ score.max_score }}</strong>
        <p>{{ score.rationale }}</p>
        <div v-if="score.evidence_refs?.length" class="chip-list">
          <span v-for="ref in score.evidence_refs.filter((item) => item.trim() !== '')" :key="ref" class="chip">{{ ref }}</span>
        </div>
        <div v-if="scoreSnippetsByID[score.id]?.length" class="snippet-list inline-snippets">
          <article v-for="snippet in scoreSnippetsByID[score.id]" :key="snippet.ref" class="snippet-card">
            <strong>{{ snippet.title }}</strong>
            <small>{{ snippet.ref }}</small>
            <p>{{ snippet.text }}</p>
          </article>
        </div>
      </article>
    </div>

    <div v-if="evaluation?.findings.length" class="finding-list">
      <article v-for="finding in evaluation.findings" :key="finding.id" :class="['finding', finding.severity]">
        <strong>{{ finding.severity }} / {{ finding.category }}</strong>
        <p>{{ finding.message }}</p>
        <small v-if="finding.evidence_ref">{{ finding.evidence_ref }}</small>
        <div v-if="findingSnippetsByID[finding.id]?.length" class="snippet-list inline-snippets">
          <article v-for="snippet in findingSnippetsByID[finding.id]" :key="snippet.ref" class="snippet-card">
            <strong>{{ snippet.title }}</strong>
            <small>{{ snippet.ref }}</small>
            <p>{{ snippet.text }}</p>
          </article>
        </div>
      </article>
    </div>

    <section v-if="evidenceSnippets.length" class="evidence-card">
      <p class="eyebrow">证据片段</p>
      <div class="chip-list">
        <span v-for="snippet in evidenceSnippets" :key="snippet.ref" class="chip">{{ snippet.ref }}</span>
      </div>
      <div class="snippet-list">
        <article v-for="snippet in evidenceSnippets" :key="snippet.ref" class="snippet-card">
          <strong>{{ snippet.title }}</strong>
          <small>{{ snippet.ref }}</small>
          <p>{{ snippet.text }}</p>
        </article>
      </div>
    </section>

    <p v-if="!evaluation" class="muted">教师可先运行规则核查或规则 + LLM 初评，再进入复核。</p>
  </section>
</template>
