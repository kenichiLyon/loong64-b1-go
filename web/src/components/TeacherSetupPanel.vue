<script setup lang="ts">
import { reactive } from 'vue';
import type { CourseRecord, ExperimentRecord, MetricRecord, RubricTemplateRecord, RubricVersionRecord } from '../lib/types';

const props = defineProps<{
  busy: boolean;
  courses: CourseRecord[];
  templates: RubricTemplateRecord[];
  versions: RubricVersionRecord[];
  experiments: ExperimentRecord[];
  activeCourseId: string;
}>();

const emit = defineEmits<{
  createTemplate: [payload: { name: string; description?: string }];
  createVersion: [payload: { templateID: string; weight_mode: string; metrics: Array<{ code: string; name: string; description?: string; weight_bps: number; max_score: number; sort_order: number }> }];
  publishVersion: [versionID: string];
  createExperiment: [payload: { courseID: string; title: string; description?: string; rubric_version_id: string; submission_spec?: Record<string, unknown> }];
  publishExperiment: [experimentID: string];
  selectCourse: [courseID: string];
  selectExperiment: [experimentID: string];
}>();

const templateForm = reactive({
  name: '软件实训评价模板',
  description: '代码质量、文档规范性、功能实现度',
});

const versionForm = reactive({
  templateID: '',
  metrics: [
    { code: 'quality', name: '代码质量', description: '结构、可读性、规范', weight_bps: 4000, max_score: 40, sort_order: 1 },
    { code: 'docs', name: '文档规范性', description: '报告完整性、截图说明', weight_bps: 3000, max_score: 30, sort_order: 2 },
    { code: 'feature', name: '功能实现度', description: '功能完成情况与运行结果', weight_bps: 3000, max_score: 30, sort_order: 3 },
  ] as Array<{ code: string; name: string; description?: string; weight_bps: number; max_score: number; sort_order: number }>,
});

const experimentForm = reactive({
  courseID: props.activeCourseId || '',
  rubricVersionID: '',
  title: 'LoongArch 部署与验收实训',
  description: '上传报告、截图和代码压缩包，完成部署验证与结果总结。',
  submissionSpecText: JSON.stringify({
    required_artifacts: ['report', 'screenshot', 'code_archive'],
    required_steps: ['环境准备', '部署执行', '结果验证'],
  }, null, 2),
});

function submitTemplate() {
  emit('createTemplate', { name: templateForm.name, description: templateForm.description || undefined });
}

function submitVersion() {
  emit('createVersion', {
    templateID: versionForm.templateID,
    weight_mode: 'strict_100',
    metrics: versionForm.metrics,
  });
}

function submitExperiment() {
  let submissionSpec: Record<string, unknown> | undefined;
  try {
    submissionSpec = JSON.parse(experimentForm.submissionSpecText) as Record<string, unknown>;
  } catch {
    submissionSpec = undefined;
  }
  emit('createExperiment', {
    courseID: experimentForm.courseID,
    title: experimentForm.title,
    description: experimentForm.description || undefined,
    rubric_version_id: experimentForm.rubricVersionID,
    submission_spec: submissionSpec,
  });
}

function useCourse(courseID: string) {
  experimentForm.courseID = courseID;
  emit('selectCourse', courseID);
}

function useExperiment(experimentID: string) {
  emit('selectExperiment', experimentID);
}
</script>

<template>
  <section class="card setup-panel">
    <div class="panel-heading split-heading">
      <div>
        <p class="eyebrow">教师搭建</p>
        <h2>创建模板版本与实验任务</h2>
      </div>
      <strong class="score-badge">teacher</strong>
    </div>

    <div class="setup-grid teacher-setup-grid">
      <div class="setup-block">
        <h3>创建模板</h3>
        <label>模板名称<input v-model="templateForm.name" /></label>
        <label>描述<textarea v-model="templateForm.description" /></label>
        <button :disabled="busy || !templateForm.name.trim()" @click="submitTemplate">创建模板</button>
        <div class="inline-list">
          <button v-for="template in templates" :key="template.id" type="button" @click="versionForm.templateID = template.id">{{ template.name }} · {{ template.id }}</button>
        </div>
      </div>

      <div class="setup-block">
        <h3>创建版本</h3>
        <label>模板 ID
          <select v-model="versionForm.templateID">
            <option value="">选择模板</option>
            <option v-for="template in templates" :key="template.id" :value="template.id">{{ template.name }} · {{ template.id }}</option>
          </select>
        </label>
        <article v-for="metric in versionForm.metrics" :key="metric.sort_order" class="metric-inline">
          <input v-model="metric.code" placeholder="code" />
          <input v-model="metric.name" placeholder="名称" />
          <input v-model.number="metric.weight_bps" type="number" min="0" max="10000" />
          <input v-model.number="metric.max_score" type="number" min="1" />
        </article>
        <button :disabled="busy || !versionForm.templateID" @click="submitVersion">创建版本</button>
        <div class="inline-list">
          <button v-for="version in versions" :key="version.id" type="button" @click="emit('publishVersion', version.id)">{{ version.id }} · {{ version.status }}</button>
        </div>
      </div>

      <div class="setup-block">
        <h3>创建实验</h3>
        <label>课程
          <select v-model="experimentForm.courseID">
            <option value="">选择课程</option>
            <option v-for="course in courses" :key="course.id" :value="course.id">{{ course.name }} · {{ course.id }}</option>
          </select>
        </label>
        <label>已选课程快捷使用
          <div class="inline-list">
            <button v-for="course in courses" :key="course.id" type="button" @click="useCourse(course.id)">{{ course.name }}</button>
          </div>
        </label>
        <label>已发布版本
          <select v-model="experimentForm.rubricVersionID">
            <option value="">选择版本</option>
            <option v-for="version in versions" :key="version.id" :value="version.id">{{ version.id }} · {{ version.status }}</option>
          </select>
        </label>
        <label>实验标题<input v-model="experimentForm.title" /></label>
        <label>实验描述<textarea v-model="experimentForm.description" /></label>
        <label>提交规范 JSON<textarea v-model="experimentForm.submissionSpecText" /></label>
        <button :disabled="busy || !experimentForm.courseID || !experimentForm.rubricVersionID || !experimentForm.title.trim()" @click="submitExperiment">创建实验</button>
        <div class="inline-list">
          <button v-for="experiment in experiments" :key="experiment.id" type="button" @click="emit('publishExperiment', experiment.id)">{{ experiment.title }} · {{ experiment.status }}</button>
          <button v-for="experiment in experiments" :key="experiment.id + '-use'" type="button" @click="useExperiment(experiment.id)">使用实验 ID</button>
        </div>
      </div>
    </div>
  </section>
</template>
