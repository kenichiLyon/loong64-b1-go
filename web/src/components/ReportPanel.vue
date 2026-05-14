<script setup lang="ts">
import type { CourseReportSummary, ExperimentReportSummary, ReportExport, SubmissionReport } from '../lib/types';

const props = defineProps<{
  report: SubmissionReport | null;
  summary: ExperimentReportSummary | null;
  courseSummary: CourseReportSummary | null;
  exportResult: ReportExport | null;
  busy: boolean;
  downloadUrl: string;
}>();

const emit = defineEmits<{
  loadReport: [];
  loadSummary: [];
  loadCourseSummary: [];
  exportSubmission: [format: 'html' | 'csv' | 'xlsx' | 'pdf'];
  exportSummary: [format: 'html' | 'csv' | 'xlsx' | 'pdf'];
  exportCourseSummary: [format: 'html' | 'csv' | 'xlsx' | 'pdf'];
}>();

function percent(value: number) {
  return `${(value / 100).toFixed(1)}%`;
}

function uniqueEvidenceRefs(report: SubmissionReport | null) {
  if (!report?.evaluation) {
    return [];
  }
  const refs = new Set<string>();
  for (const score of report.evaluation.scores) {
    for (const ref of score.evidence_refs ?? []) {
      if (ref.trim() !== '') {
        refs.add(ref);
      }
    }
  }
  for (const finding of report.evaluation.findings) {
    if (finding.evidence_ref && finding.evidence_ref.trim() !== '') {
      refs.add(finding.evidence_ref);
    }
  }
  return Array.from(refs);
}
</script>

<template>
  <section class="card report-panel">
    <div class="panel-heading split-heading">
      <div>
        <p class="eyebrow">统计报表</p>
        <h2>{{ report ? '个人报告就绪' : 'Stage 6 报表导出' }}</h2>
      </div>
      <strong class="score-badge">HTML / CSV / XLSX / PDF</strong>
    </div>

    <div class="button-row report-actions">
      <button :disabled="busy" @click="emit('loadReport')">预览个人报告</button>
      <button :disabled="busy" @click="emit('loadSummary')">实验统计</button>
      <button :disabled="busy" @click="emit('loadCourseSummary')">课程统计</button>
      <button :disabled="busy" @click="emit('exportSubmission', 'html')">个人 HTML</button>
      <button :disabled="busy" @click="emit('exportSubmission', 'csv')">个人 CSV</button>
      <button :disabled="busy" @click="emit('exportSubmission', 'xlsx')">个人 XLSX</button>
      <button :disabled="busy" @click="emit('exportSummary', 'csv')">统计 CSV</button>
      <button :disabled="busy" @click="emit('exportSummary', 'xlsx')">统计 XLSX</button>
      <button :disabled="busy" @click="emit('exportSummary', 'pdf')">统计 PDF</button>
      <button :disabled="busy" @click="emit('exportCourseSummary', 'csv')">课程 CSV</button>
      <button :disabled="busy" @click="emit('exportCourseSummary', 'xlsx')">课程 XLSX</button>
      <button :disabled="busy" @click="emit('exportCourseSummary', 'pdf')">课程 PDF</button>
      <button :disabled="busy" @click="emit('exportSubmission', 'pdf')">个人 PDF</button>
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

    <div v-if="report && uniqueEvidenceRefs(report).length" class="summary-block">
      <h3>AI 引用证据</h3>
      <div class="chip-list">
        <span v-for="ref in uniqueEvidenceRefs(report)" :key="ref" class="chip">{{ ref }}</span>
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
      <div v-if="summary.evidence_ref_counts.length" class="summary-block nested-block">
        <h4>AI 证据引用概览</h4>
        <div class="chip-list">
          <span v-for="item in summary.evidence_ref_counts" :key="item.reference" class="chip">{{ item.reference }} · {{ item.count }}</span>
        </div>
      </div>
    </div>

    <div v-if="courseSummary" class="summary-block">
      <h3>课程统计 {{ courseSummary.course_id }}</h3>
      <div class="facts-grid compact-facts">
        <div>
          <span>实验数</span>
          <strong>{{ courseSummary.experiment_count }}</strong>
        </div>
        <div>
          <span>提交数</span>
          <strong>{{ courseSummary.submission_count }}</strong>
        </div>
        <div>
          <span>已发布评价</span>
          <strong>{{ courseSummary.published_review_count }}</strong>
        </div>
        <div>
          <span>平均分</span>
          <strong>{{ percent(courseSummary.average_score_bps) }}</strong>
        </div>
      </div>
      <div class="bucket-row">
        <span v-for="(count, bucket) in courseSummary.score_buckets" :key="bucket">{{ bucket }} · {{ count }}</span>
      </div>
      <article v-for="experiment in courseSummary.experiments" :key="experiment.experiment_id" class="metric-card">
        <span>{{ experiment.experiment_id }}</span>
        <strong>{{ percent(experiment.average_score_bps) }} · {{ experiment.published_review_count }}/{{ experiment.submission_count }}</strong>
      </article>
      <div v-if="courseSummary.evidence_ref_counts.length" class="summary-block nested-block">
        <h4>课程级 AI 证据引用概览</h4>
        <div class="chip-list">
          <span v-for="item in courseSummary.evidence_ref_counts" :key="item.reference" class="chip">{{ item.reference }} · {{ item.count }}</span>
        </div>
      </div>
    </div>

    <div v-if="exportResult" :class="['export-result', exportResult.status]">
      <strong>{{ exportResult.report_type }} · {{ exportResult.format }} · {{ exportResult.status }}</strong>
      <p v-if="exportResult.error">{{ exportResult.error }}</p>
      <p v-else>SHA-256：{{ exportResult.sha256_hex }} · {{ exportResult.byte_size }} bytes</p>
      <a v-if="exportResult.status === 'succeeded'" :href="props.downloadUrl" target="_blank" rel="noreferrer">下载导出文件</a>
    </div>

    <p class="muted">HTML 是规范归档源；CSV 与 XLSX 面向表格分析；PDF 用于归档和打印。</p>
  </section>
</template>
