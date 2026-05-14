<script setup lang="ts">
import { computed, reactive, watch } from 'vue';
import type { EvaluationResultDetail, TeacherMetricScore, TeacherReviewDetail } from '../lib/types';

const props = defineProps<{
  evaluation: EvaluationResultDetail | null;
  review: TeacherReviewDetail | null;
  busy: boolean;
}>();

const emit = defineEmits<{
  save: [payload: unknown];
  publish: [];
}>();

const state = reactive({
  teacher_comment: '',
  scores: [] as Array<{ metric_code: string; final_score: number; max_score: number; source: string; source_metric_score_id?: string; adjustment_reason: string; comment: string }>,
});

const published = computed(() => props.review?.review.status === 'published');
const retrievalContext = computed(() => {
  const output = props.evaluation?.log?.output as Record<string, unknown> | undefined;
  const context = output?.retrieval_context;
  return context && typeof context === 'object' ? (context as Record<string, unknown>) : null;
});

...omitted unchanged watch/save logic...
</script>

<template>
  <section class="card review-panel">
    <div class="panel-heading split-heading">
      <div>
        <p class="eyebrow">教师复核</p>
        <h2>{{ published ? '已发布' : '复核草稿' }}</h2>
      </div>
      <strong class="score-badge">{{ review ? (review.review.total_score_bps / 100).toFixed(1) + '%' : '未保存' }}</strong>
    </div>

    <div v-if="state.scores.length" class="review-table">
      <label v-for="score in state.scores" :key="score.metric_code">
        <span>{{ score.metric_code }}</span>
        <input v-model.number="score.final_score" :disabled="published" type="number" min="0" :max="score.max_score" />
        <small>/ {{ score.max_score }} · {{ score.source }}</small>
        <textarea v-model="score.adjustment_reason" :disabled="published" placeholder="改分理由或复核说明" />
      </label>
    </div>
    <p v-else class="muted">先运行初评或读取已有复核结果。</p>

    <label class="comment-box">
      教师总评
      <textarea v-model="state.teacher_comment" :disabled="published" placeholder="给学生的综合反馈" />
    </label>

    <section v-if="retrievalContext" class="evidence-card">
      <p class="eyebrow">AI 检索证据</p>
      <p class="muted">命中数：{{ retrievalContext.hit_count ?? 0 }}</p>
      <div v-if="Array.isArray(retrievalContext.queries) && retrievalContext.queries.length" class="chip-list">
        <span v-for="query in retrievalContext.queries" :key="String(query)" class="chip">{{ query }}</span>
      </div>
      <ul v-if="Array.isArray(retrievalContext.citations) && retrievalContext.citations.length" class="citation-list">
        <li v-for="citation in retrievalContext.citations" :key="String((citation as Record<string, unknown>).chunk_id ?? (citation as Record<string, unknown>).evidence_ref)">
          {{ (citation as Record<string, unknown>).evidence_ref ?? 'unknown evidence' }}
        </li>
      </ul>
    </section>

    <div class="button-row">
      <button :disabled="busy || published || !state.scores.length" @click="save">保存草稿</button>
      <button class="danger" :disabled="busy || published || !review" @click="emit('publish')">确认发布</button>
    </div>
  </section>
</template>
