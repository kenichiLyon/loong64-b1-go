<script setup lang="ts">
import type { EvaluationResultDetail } from '../lib/types';

defineProps<{ evaluation: EvaluationResultDetail | null }>();
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
          <span v-for="ref in score.evidence_refs" :key="ref" class="chip">{{ ref }}</span>
        </div>
      </article>
    </div>

    <div v-if="evaluation?.findings.length" class="finding-list">
      <article v-for="finding in evaluation.findings" :key="finding.id" :class="['finding', finding.severity]">
        <strong>{{ finding.severity }} / {{ finding.category }}</strong>
        <p>{{ finding.message }}</p>
        <small v-if="finding.evidence_ref">{{ finding.evidence_ref }}</small>
      </article>
    </div>

    <p v-if="!evaluation" class="muted">教师可先运行规则核查或规则 + LLM 初评，再进入复核。</p>
  </section>
</template>
