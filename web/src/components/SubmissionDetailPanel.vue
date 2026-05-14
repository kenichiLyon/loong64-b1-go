<script setup lang="ts">
import type { EvaluationResultDetail, SubmissionDetail, TeacherReviewDetail } from '../lib/types';

defineProps<{
  detail: SubmissionDetail | null;
  review?: TeacherReviewDetail | null;
  evaluation?: EvaluationResultDetail | null;
}>();
</script>

<template>
  <section class="card detail-panel">
    <div class="panel-heading">
      <p class="eyebrow">提交详情</p>
      <h2>{{ detail?.submission.id ?? '尚未选择提交' }}</h2>
    </div>

    <div v-if="detail" class="facts-grid">
      <div>
        <span>状态</span>
        <strong>{{ detail.submission.status }}</strong>
      </div>
      <div>
        <span>实验</span>
        <strong>{{ detail.submission.experiment_id }}</strong>
      </div>
      <div>
        <span>附件数</span>
        <strong>{{ detail.artifacts.length }}</strong>
      </div>
      <div>
        <span>最终分</span>
        <strong>{{ review ? (review.review.total_score_bps / 100).toFixed(1) + '%' : '未发布' }}</strong>
      </div>
    </div>

    <div v-if="detail" class="artifact-list">
      <article v-for="item in detail.artifacts" :key="item.artifact.id" class="artifact-item">
        <div>
          <strong>{{ item.artifact.original_name }}</strong>
          <span>{{ item.artifact.kind }} / {{ item.extraction.status }}</span>
        </div>
        <p>{{ item.extraction.text_excerpt || item.extraction.error || '暂无摘要' }}</p>
      </article>
    </div>

    <section v-if="evaluation?.log" class="artifact-list">
      <article class="artifact-item">
        <div>
          <strong>AI 调用摘要</strong>
          <span>{{ evaluation.log.provider }} / {{ evaluation.log.model || 'unknown model' }}</span>
        </div>
        <p>{{ evaluation.result.llm_summary || '暂无模型总结' }}</p>
      </article>
    </section>

    <p v-else class="muted">输入提交 ID 后可查看附件、解析摘要和已发布评价。</p>
  </section>
</template>
