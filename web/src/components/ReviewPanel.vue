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

watch(
  () => [props.evaluation, props.review] as const,
  ([evaluation, review]) => {
    if (review) {
      state.teacher_comment = review.review.teacher_comment ?? '';
      state.scores = review.scores.map((score: TeacherMetricScore) => ({
        metric_code: score.metric_code,
        final_score: score.final_score,
        max_score: score.max_score,
        source: score.source,
        source_metric_score_id: score.source_metric_score_id,
        adjustment_reason: score.adjustment_reason ?? '',
        comment: score.comment ?? '',
      }));
      return;
    }
    if (evaluation) {
      const byMetric = new Map<string, typeof evaluation.scores[number]>();
      for (const score of evaluation.scores) {
        if (!byMetric.has(score.metric_code) || score.source === 'llm') {
          byMetric.set(score.metric_code, score);
        }
      }
      state.teacher_comment = evaluation.result.llm_summary ?? '';
      state.scores = Array.from(byMetric.values()).map((score) => ({
        metric_code: score.metric_code,
        final_score: score.suggested_score,
        max_score: score.max_score,
        source: score.source,
        source_metric_score_id: score.id,
        adjustment_reason: '基于智能初评建议生成，教师已复核。',
        comment: score.rationale,
      }));
    }
  },
  { immediate: true },
);

function save() {
  emit('save', {
    evaluation_result_id: props.evaluation?.result.id,
    teacher_comment: state.teacher_comment,
    scores: state.scores.map((score) => ({
      metric_code: score.metric_code,
      final_score: Number(score.final_score),
      source: score.source,
      source_metric_score_id: score.source_metric_score_id,
      adjustment_reason: score.adjustment_reason,
      comment: score.comment,
    })),
  });
}
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

    <div class="button-row">
      <button :disabled="busy || published || !state.scores.length" @click="save">保存草稿</button>
      <button class="danger" :disabled="busy || published || !review" @click="emit('publish')">确认发布</button>
    </div>
  </section>
</template>
