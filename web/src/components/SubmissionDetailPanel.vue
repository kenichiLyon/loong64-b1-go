<script setup lang="ts">
import { computed } from 'vue';
import { collectEvaluationEvidenceRefs, resolveArtifactEvidenceOutlines } from '../lib/evidence';
import type { EvaluationResultDetail, SubmissionDetail, TeacherReviewDetail } from '../lib/types';

const props = defineProps<{
  detail: SubmissionDetail | null;
  review?: TeacherReviewDetail | null;
  evaluation?: EvaluationResultDetail | null;
}>();

const evidenceOutlines = computed(() => resolveArtifactEvidenceOutlines(props.detail));
const aiEvidenceRefs = computed(() => new Set(collectEvaluationEvidenceRefs(props.evaluation ?? null)));
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

    <div v-if="evidenceOutlines.length" class="artifact-list">
      <article v-for="outline in evidenceOutlines" :key="outline.artifactID" class="artifact-item">
        <div>
          <strong>{{ outline.artifactName }}</strong>
          <span>{{ outline.sections.length }} sections / {{ outline.evidence.length }} evidence</span>
        </div>
        <div v-if="outline.sections.length" class="snippet-list">
          <article v-for="snippet in outline.sections" :key="snippet.ref" :class="['snippet-card', { highlight: aiEvidenceRefs.has(snippet.ref) }]">
            <strong>{{ snippet.title }}</strong>
            <small>{{ snippet.ref }}</small>
            <span v-if="aiEvidenceRefs.has(snippet.ref)" class="chip">AI 引用</span>
            <p>{{ snippet.text }}</p>
          </article>
        </div>
        <div v-if="outline.evidence.length" class="snippet-list">
          <article v-for="snippet in outline.evidence" :key="snippet.ref" :class="['snippet-card', { highlight: aiEvidenceRefs.has(snippet.ref) }]">
            <strong>{{ snippet.title }}</strong>
            <small>{{ snippet.ref }}</small>
            <span v-if="aiEvidenceRefs.has(snippet.ref)" class="chip">AI 引用</span>
            <p>{{ snippet.text }}</p>
          </article>
        </div>
      </article>
    </div>

    <p v-else class="muted">输入提交 ID 后可查看附件、解析摘要和已发布评价。</p>
  </section>
</template>
