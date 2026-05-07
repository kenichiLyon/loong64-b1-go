<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import BootstrapPanel from './components/BootstrapPanel.vue';
import DeploymentAssistantPanel from './components/DeploymentAssistantPanel.vue';
import EvaluationPanel from './components/EvaluationPanel.vue';
import LoginPanel from './components/LoginPanel.vue';
import ReportPanel from './components/ReportPanel.vue';
import ReviewPanel from './components/ReviewPanel.vue';
import RuntimeConfigPanel from './components/RuntimeConfigPanel.vue';
import SubmissionDetailPanel from './components/SubmissionDetailPanel.vue';
import { api } from './lib/api';
import type { ActorRole, AssistantConversationDetail, AssistantToolCall, BootstrapStatus, CourseReportSummary, EvaluationResultDetail, ExperimentReportSummary, ReportExport, RuntimeConfigSummary, Submission, SubmissionDetail, SubmissionReport, TeacherReviewDetail } from './lib/types';
import './styles.css';

const actorID = ref('teacher-1');
const roles = ref<ActorRole[]>(['teacher']);
const courseID = ref('');
const experimentID = ref('');
const submissionID = ref('');
const artifactKind = ref('report');
const gitURL = ref('');
const gitCommitSHA = ref('');
const gitNote = ref('');
const selectedFile = ref<File | null>(null);
const busy = ref(false);
const message = ref('准备就绪');

const submissions = ref<Submission[]>([]);
const detail = ref<SubmissionDetail | null>(null);
const evaluation = ref<EvaluationResultDetail | null>(null);
const review = ref<TeacherReviewDetail | null>(null);
const report = ref<SubmissionReport | null>(null);
const summary = ref<ExperimentReportSummary | null>(null);
const courseSummary = ref<CourseReportSummary | null>(null);
const exportResult = ref<ReportExport | null>(null);
const runtimeConfig = ref<RuntimeConfigSummary | null>(null);
const bootstrapStatus = ref<BootstrapStatus | null>(null);
const bootstrapAssistant = ref<AssistantConversationDetail | null>(null);
const deploymentAssistant = ref<AssistantConversationDetail | null>(null);
const loggedIn = ref(false);

const requestOptions = computed(() => ({}));
const exportDownloadURL = computed(() => (exportResult.value ? api.reportExportDownloadURL(exportResult.value.id) : ''));
const mode = reactive({ evaluation: 'rule_only' as 'rule_only' | 'rule_and_llm' });

async function runAction(label: string, action: () => Promise<void>) {
  busy.value = true;
  message.value = `${label}...`;
  try {
    await action();
    message.value = `${label}完成`;
  } catch (error) {
    message.value = error instanceof Error ? error.message : String(error);
  } finally {
    busy.value = false;
  }
}

async function createSubmission() {
  await runAction('创建提交', async () => {
    const submission = await api.createSubmission(experimentID.value, requestOptions.value);
    submissionID.value = submission.id;
    await loadStudentSubmission();
  });
}

async function loadBootstrapStatus() {
  try {
    bootstrapStatus.value = await api.getBootstrapStatus();
  } catch (error) {
    message.value = error instanceof Error ? error.message : String(error);
  }
}

async function loadCurrentUser() {
  try {
    const me = await api.me();
    actorID.value = me.id;
    roles.value = me.roles;
    loggedIn.value = true;
  } catch {
    loggedIn.value = false;
    actorID.value = '';
    roles.value = [];
  }
}

async function ensureBootstrapAssistantConversation() {
  if (bootstrapAssistant.value || bootstrapStatus.value?.initialized) {
    return;
  }
  const conversation = await api.createBootstrapAssistantConversation();
  bootstrapAssistant.value = await api.getBootstrapAssistantConversation(conversation.id);
}

async function ensureDeploymentAssistantConversation() {
  if (deploymentAssistant.value || !roles.value.includes('admin') || bootstrapStatus.value?.initialized === false) {
    return;
  }
  const conversation = await api.createDeploymentAssistantConversation(requestOptions.value);
  deploymentAssistant.value = await api.getDeploymentAssistantConversation(conversation.id, requestOptions.value);
}

async function sendBootstrapAssistantMessage(content: string) {
  await runAction('部署助手响应中', async () => {
    await ensureBootstrapAssistantConversation();
    if (!bootstrapAssistant.value) {
      return;
    }
    await api.sendBootstrapAssistantMessage(bootstrapAssistant.value.conversation.id, content);
    bootstrapAssistant.value = await api.getBootstrapAssistantConversation(bootstrapAssistant.value.conversation.id);
  });
}

async function confirmBootstrapAssistantTool(toolCall: AssistantToolCall, inputs: Record<string, unknown>) {
  await runAction(`执行 ${toolCall.tool_name}`, async () => {
    const result = await api.confirmBootstrapAssistantToolCall(toolCall.id, inputs);
    if (bootstrapAssistant.value) {
      bootstrapAssistant.value = await api.getBootstrapAssistantConversation(bootstrapAssistant.value.conversation.id);
    }
    await loadBootstrapStatus();
    if (toolCall.tool_name === 'bootstrap_create_admin') {
      await loadCurrentUser();
    }
  });
}

async function uploadArtifact() {
  if (!selectedFile.value || !submissionID.value) {
    message.value = '请选择文件并填写提交 ID';
    return;
  }
  await runAction('上传成果', async () => {
    await api.uploadArtifact(submissionID.value, selectedFile.value as File, artifactKind.value, requestOptions.value);
    await loadStudentSubmission();
  });
}

async function createGitLink() {
  await runAction('登记 Git 链接', async () => {
    await api.createGitLink(submissionID.value, gitURL.value, gitCommitSHA.value, gitNote.value, requestOptions.value);
    await loadStudentSubmission();
  });
}

async function listSubmissions() {
  await runAction('读取提交列表', async () => {
    const response = await api.listExperimentSubmissions(experimentID.value, requestOptions.value);
    submissions.value = response.items;
  });
}

async function loadTeacherSubmission(id = submissionID.value) {
  submissionID.value = id;
  await runAction('读取教师提交详情', async () => {
    detail.value = await api.getSubmission(submissionID.value, 'teacher', requestOptions.value);
  });
}

async function loadStudentSubmission() {
  await runAction('读取学生提交详情', async () => {
    detail.value = await api.getSubmission(submissionID.value, 'student', requestOptions.value);
  });
}

async function runEvaluation() {
  await runAction('运行智能核查', async () => {
    evaluation.value = await api.createEvaluation(submissionID.value, mode.evaluation, requestOptions.value);
  });
}

async function loadEvaluation() {
  await runAction('读取最新初评', async () => {
    evaluation.value = await api.getLatestEvaluation(submissionID.value, requestOptions.value);
  });
}

async function saveReview(payload: unknown) {
  await runAction('保存复核草稿', async () => {
    review.value = await api.saveTeacherReview(submissionID.value, payload, requestOptions.value);
  });
}

async function publishReview() {
  await runAction('发布最终评价', async () => {
    review.value = await api.publishTeacherReview(submissionID.value, requestOptions.value);
  });
}

async function loadReview(role: 'teacher' | 'student') {
  await runAction(role === 'teacher' ? '读取教师复核' : '读取已发布评价', async () => {
    review.value = await api.getTeacherReview(submissionID.value, role, requestOptions.value);
  });
}

async function loadSubmissionReport() {
  const role = roles.value.includes('teacher') || roles.value.includes('admin') ? 'teacher' : 'student';
  await runAction('读取个人评价报告', async () => {
    report.value = await api.getSubmissionReport(submissionID.value, role, requestOptions.value);
    review.value = report.value.review;
    evaluation.value = report.value.evaluation ?? evaluation.value;
  });
}

async function loadExperimentSummary() {
  await runAction('读取实验统计', async () => {
    summary.value = await api.getExperimentReportSummary(experimentID.value, requestOptions.value);
  });
}

async function loadCourseSummary() {
  await runAction('读取课程统计', async () => {
    courseSummary.value = await api.getCourseReportSummary(courseID.value, requestOptions.value);
  });
}

async function exportSubmissionReport(format: 'html' | 'csv' | 'pdf') {
  await runAction(`导出个人报告 ${format}`, async () => {
    exportResult.value = await api.createSubmissionReportExport(submissionID.value, format, requestOptions.value);
  });
}

async function exportExperimentSummary(format: 'html' | 'csv' | 'pdf') {
  await runAction(`导出实验统计 ${format}`, async () => {
    exportResult.value = await api.createExperimentSummaryExport(experimentID.value, format, requestOptions.value);
  });
}

async function exportCourseSummary(format: 'html' | 'csv' | 'pdf') {
  await runAction(`导出课程统计 ${format}`, async () => {
    exportResult.value = await api.createCourseSummaryExport(courseID.value, format, requestOptions.value);
  });
}

async function loadRuntimeConfig() {
  await runAction('读取运行配置', async () => {
    runtimeConfig.value = await api.getRuntimeConfig(requestOptions.value);
  });
}

async function saveRuntimeConfig(payload: { db_driver: 'sqlite' | 'postgres'; sqlite_path?: string; database_url?: string; auto_migrate?: boolean }) {
  await runAction('保存运行配置', async () => {
    runtimeConfig.value = await api.updateRuntimeConfig(payload, requestOptions.value);
  });
}

async function sendDeploymentAssistantMessage(content: string) {
  await runAction('部署助手响应中', async () => {
    await ensureDeploymentAssistantConversation();
    if (!deploymentAssistant.value) {
      return;
    }
    await api.sendDeploymentAssistantMessage(deploymentAssistant.value.conversation.id, content, requestOptions.value);
    deploymentAssistant.value = await api.getDeploymentAssistantConversation(deploymentAssistant.value.conversation.id, requestOptions.value);
  });
}

async function confirmDeploymentAssistantTool(toolCall: AssistantToolCall, inputs: Record<string, unknown>) {
  await runAction(`执行 ${toolCall.tool_name}`, async () => {
    await api.confirmDeploymentAssistantToolCall(toolCall.id, inputs, requestOptions.value);
    if (deploymentAssistant.value) {
      deploymentAssistant.value = await api.getDeploymentAssistantConversation(deploymentAssistant.value.conversation.id, requestOptions.value);
    }
    await loadRuntimeConfig();
  });
}

async function bootstrapCreateAdmin(payload: { username: string; display_name: string; email?: string; employee_no?: string; password: string }) {
  await runAction('初始化管理员', async () => {
    const response = await api.bootstrapCreateAdmin(payload);
    actorID.value = response.user.id;
    roles.value = ['admin'];
    loggedIn.value = true;
    await loadBootstrapStatus();
    await loadRuntimeConfig();
  });
}

async function login(payload: { username: string; password: string }) {
  await runAction('登录', async () => {
    const me = await api.login(payload);
    actorID.value = me.id;
    roles.value = me.roles;
    loggedIn.value = true;
    await loadRuntimeConfig();
  });
}

async function logout() {
  await runAction('退出登录', async () => {
    await api.logout();
    loggedIn.value = false;
    actorID.value = '';
    roles.value = [];
    runtimeConfig.value = null;
    deploymentAssistant.value = null;
  });
}

function onFileChange(event: Event) {
  selectedFile.value = (event.target as HTMLInputElement).files?.[0] ?? null;
}

onMounted(() => {
  void (async () => {
    await loadBootstrapStatus();
    if (bootstrapStatus.value?.initialized) {
      await loadCurrentUser();
    }
  })();
});
</script>

<template>
  <main class="shell">
    <section class="hero">
      <div>
        <p class="eyebrow">LoongArch 实训评价系统</p>
        <h1>把上传、核查、初评、复核和发布串成一条可演示链路。</h1>
        <p>这是 PC Web MVP，优先服务学生提交和教师复核主流程。开发态通过请求头模拟身份。</p>
      </div>
      <div class="status-orb">
        <span>{{ busy ? '运行中' : '在线' }}</span>
        <strong>{{ message }}</strong>
      </div>
    </section>

    <BootstrapPanel
      v-if="bootstrapStatus && !bootstrapStatus.initialized"
      :busy="busy"
      :status="bootstrapStatus"
      @create-admin="bootstrapCreateAdmin"
    />

    <DeploymentAssistantPanel
      v-if="bootstrapStatus && !bootstrapStatus.initialized"
      :bootstrap-status="bootstrapStatus"
      :busy="busy"
      :detail="bootstrapAssistant"
      :runtime-config="runtimeConfig"
      scope="bootstrap"
      @confirm="confirmBootstrapAssistantTool"
      @ensure-conversation="ensureBootstrapAssistantConversation"
      @send="sendBootstrapAssistantMessage"
    />

    <LoginPanel v-else-if="bootstrapStatus?.initialized && !loggedIn" :busy="busy" @login="login" />

    <section v-else class="card identity-card">
      <div>
        <p class="eyebrow">当前会话</p>
        <h2>{{ actorID }}</h2>
      </div>
      <label>
        角色
        <input :value="roles.join(', ')" readonly />
      </label>
      <div class="button-row">
        <button :disabled="busy" @click="logout">退出登录</button>
      </div>
    </section>

    <RuntimeConfigPanel
      v-if="bootstrapStatus?.initialized && loggedIn && roles.includes('admin')"
      :busy="busy"
      :summary="runtimeConfig"
      @load="loadRuntimeConfig"
      @save="saveRuntimeConfig"
    />

    <DeploymentAssistantPanel
      v-if="bootstrapStatus?.initialized && loggedIn && roles.includes('admin')"
      :bootstrap-status="bootstrapStatus"
      :busy="busy"
      :detail="deploymentAssistant"
      :runtime-config="runtimeConfig"
      scope="deployment_admin"
      @confirm="confirmDeploymentAssistantTool"
      @ensure-conversation="ensureDeploymentAssistantConversation"
      @send="sendDeploymentAssistantMessage"
    />

    <section v-if="bootstrapStatus?.initialized !== false && loggedIn" class="workspace-grid">
      <section class="card flow-card student-flow">
        <p class="eyebrow">学生流程</p>
        <h2>创建提交与上传成果</h2>
        <label>
          课程 ID
          <input v-model="courseID" placeholder="crs_xxx" />
        </label>
        <label>
          实验 ID
          <input v-model="experimentID" placeholder="exp_xxx" />
        </label>
        <label>
          提交 ID
          <input v-model="submissionID" placeholder="sub_xxx" />
        </label>
        <div class="button-row">
          <button :disabled="busy || !experimentID" @click="createSubmission">创建提交</button>
          <button :disabled="busy || !submissionID" @click="loadStudentSubmission">读取提交</button>
        </div>
        <label>
          成果类型
          <select v-model="artifactKind">
            <option value="report">report</option>
            <option value="document">document</option>
            <option value="screenshot">screenshot</option>
            <option value="code_archive">code_archive</option>
          </select>
        </label>
        <input type="file" @change="onFileChange" />
        <button :disabled="busy || !selectedFile || !submissionID" @click="uploadArtifact">上传文件</button>
        <label>
          Git 链接
          <input v-model="gitURL" placeholder="https://example.edu/repo.git" />
        </label>
        <label>
          Commit SHA（可选）
          <input v-model="gitCommitSHA" placeholder="7-64 位提交哈希" />
        </label>
        <label>
          链接说明（可选）
          <input v-model="gitNote" placeholder="分支、目录或提交说明" />
        </label>
        <button :disabled="busy || !gitURL || !submissionID" @click="createGitLink">登记 Git 链接</button>
      </section>

      <section class="card flow-card teacher-flow">
        <p class="eyebrow">教师流程</p>
        <h2>核查、初评与发布</h2>
        <label>
          实验 ID
          <input v-model="experimentID" placeholder="exp_xxx" />
        </label>
        <button :disabled="busy || !experimentID" @click="listSubmissions">查看提交列表</button>
        <div class="submission-list">
          <button v-for="item in submissions" :key="item.id" @click="loadTeacherSubmission(item.id)">
            {{ item.id }} · {{ item.student_id }} · {{ item.status }}
          </button>
        </div>
        <label>
          提交 ID
          <input v-model="submissionID" placeholder="sub_xxx" />
        </label>
        <div class="button-row">
          <button :disabled="busy || !submissionID" @click="loadTeacherSubmission()">读取详情</button>
          <button :disabled="busy || !submissionID" @click="loadReview('teacher')">读取复核</button>
        </div>
        <label>
          初评模式
          <select v-model="mode.evaluation">
            <option value="rule_only">rule_only</option>
            <option value="rule_and_llm">rule_and_llm</option>
          </select>
        </label>
        <div class="button-row">
          <button :disabled="busy || !submissionID" @click="runEvaluation">运行初评</button>
          <button :disabled="busy || !submissionID" @click="loadEvaluation">读取初评</button>
        </div>
      </section>
    </section>

    <section v-if="bootstrapStatus?.initialized !== false && loggedIn" class="dashboard-grid">
      <SubmissionDetailPanel :detail="detail" :review="review" />
      <EvaluationPanel :evaluation="evaluation" />
      <ReviewPanel :busy="busy" :evaluation="evaluation" :review="review" @save="saveReview" @publish="publishReview" />
    </section>

    <ReportPanel
      v-if="bootstrapStatus?.initialized !== false && loggedIn"
      :busy="busy"
      :course-summary="courseSummary"
      :download-url="exportDownloadURL"
      :export-result="exportResult"
      :report="report"
      :summary="summary"
      @export-course-summary="exportCourseSummary"
      @export-submission="exportSubmissionReport"
      @export-summary="exportExperimentSummary"
      @load-course-summary="loadCourseSummary"
      @load-report="loadSubmissionReport"
      @load-summary="loadExperimentSummary"
    />

    <section v-if="bootstrapStatus?.initialized !== false && loggedIn" class="card published-card">
      <p class="eyebrow">学生查看发布结果</p>
      <h2>发布后反馈</h2>
      <p>切换为学生角色并输入自己的提交 ID 后，可读取教师发布的最终评价。</p>
      <button :disabled="busy || !submissionID" @click="loadReview('student')">读取已发布评价</button>
    </section>
  </main>
</template>
