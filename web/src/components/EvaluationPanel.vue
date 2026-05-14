<script setup lang="ts">
import { computed } from 'vue';
import { collectEvaluationEvidenceRefs, resolveEvidenceSnippets } from '../lib/evidence';
import type { EvaluationResultDetail, SubmissionDetail } from '../lib/types';

const props = defineProps<{
  evaluation: EvaluationResultDetail | null;
  detail: SubmissionDetail | null;
}>();

const evidenceRefs = computed(() => collectEvaluationEvidenceRefs(props.evaluation));
const evidenceSnippets = computed(() => resolveEvidenceSnippets(props.detail, evidenceRefs.value));

function resolveScoreSnippets(refs: string[] | undefined) {
  return resolveEvidenceSnippets(props.detail, refs ?? []);
}

function resolveFindingSnippets(ref: string | undefined) {
  return ref ? resolveEvidenceSnippets(props.detail, [ref]) : [];
}
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
        <div v-if="resolveScoreSnippets(score.evidence_refs).length" class="snippet-list inline-snippets">
          <article v-for="snippet in resolveScoreSnippets(score.evidence_refs)" :key="snippet.ref" class="snippet-card">
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
        <div v-if="resolveFindingSnippets(finding.evidence_ref).length" class="snippet-list inline-snippets">
          <article v-for="snippet in resolveFindingSnippets(finding.evidence_ref)" :key="snippet.ref" class="snippet-card">
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
