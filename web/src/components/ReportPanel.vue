<script setup lang="ts">
import type { ExperimentReportSummary, ReportExport, SubmissionReport } from '../lib/types';

const props = defineProps<{
  report: SubmissionReport | null;
  summary: ExperimentReportSummary | null;
  exportResult: ReportExport | null;
  busy: boolean;
  downloadUrl: string;
}>();

const emit = defineEmits<{
  loadReport: [];
  loadSummary: [];
  exportSubmission: [format: 'html' | 'csv' | 'pdf'];
  exportSummary: [format: 'html' | 'csv' | 'pdf'];
}>();

function percent(value: number) {
  return `${(value / 100).toFixed(1)}%`;
}
</script>

<template>
  <section class="card report-panel">
    <div class="panel-heading split-heading">
      <div>
        <p class="eyebrow">统计报表</p>
        <h2>{{ report ? '个人报告就绪' : 'Stage 6 报表导出' }}</h2>
      </div>
      <strong class="score-badge">HTML / CSV</strong>
    </div>

    <div class="button-row report-actions">
      <button :disabled="busy" @click="emit('loadReport')">预览个人报告</button>
      <button :disabled="busy" @click="emit('loadSummary')">实验统计</button>
      <button :disabled="busy" @click="emit('exportSubmission', 'html')">个人 HTML</button>
      <button :disabled="busy" @click="emit('exportSubmission', 'csv')">个人 CSV</button>
      <button :disabled="busy" @click="emit('exportSummary', 'csv')">统计 CSV</button>
      <button :disabled="busy" @click="emit('exportSubmission', 'pdf')">PDF 降级记录</button>
    </div>

    <div v-if="report" class="facts-grid compact-facts">
      <div>
        <span>实验</span>
        <strong>{{ report.experiment.title }}</strong>
      </div>
      <div>
        <span>最终分</span>
        <strong>{{ percent(report.review.review.total_score_bps) }}</strong>
      </div>
      <div>
        <span>指标数</span>
        <strong>{{ report.review.scores.length }}</strong>
      </div>
      <div>
        <span>证据数</span>
        <strong>{{ report.artifacts.length }}</strong>
      </div>
    </div>

    <div v-if="summary" class="summary-block">
      <h3>实验统计 {{ summary.experiment_id }}</h3>
      <div class="facts-grid compact-facts">
        <div>
          <span>提交数</span>
          <strong>{{ summary.submission_count }}</strong>
        </div>
        <div>
          <span>已发布评价</span>
          <strong>{{ summary.published_review_count }}</strong>
        </div>
        <div>
          <span>平均分</span>
          <strong>{{ percent(summary.average_score_bps) }}</strong>
        </div>
        <div>
          <span>最高分</span>
          <strong>{{ percent(summary.max_score_bps) }}</strong>
        </div>
      </div>
      <div class="bucket-row">
        <span v-for="(count, bucket) in summary.score_buckets" :key="bucket">{{ bucket }} · {{ count }}</span>
      </div>
      <article v-for="metric in summary.metric_averages" :key="metric.metric_code" class="metric-card">
        <span>{{ metric.metric_code }}</span>
        <strong>{{ metric.average_score }}/{{ metric.max_score }} · {{ percent(metric.average_percent_bps) }}</strong>
      </article>
    </div>

    <div v-if="exportResult" :class="['export-result', exportResult.status]">
      <strong>{{ exportResult.report_type }} · {{ exportResult.format }} · {{ exportResult.status }}</strong>
      <p v-if="exportResult.error">{{ exportResult.error }}</p>
      <p v-else>SHA-256：{{ exportResult.sha256_hex }} · {{ exportResult.byte_size }} bytes</p>
      <a v-if="exportResult.status === 'succeeded'" :href="props.downloadUrl" target="_blank" rel="noreferrer">下载导出文件</a>
    </div>

    <p class="muted">PDF 当前按 LoongArch 风险策略记录为失败/待配置，HTML 是规范归档源，CSV 可由 WPS/Excel/LibreOffice 打开。</p>
  </section>
</template>
