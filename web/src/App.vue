<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import AccountSecurityPanel from './components/AccountSecurityPanel.vue';
import AdminSetupPanel from './components/AdminSetupPanel.vue';
import BootstrapPanel from './components/BootstrapPanel.vue';
import AdminUserPanel from './components/AdminUserPanel.vue';
import DeploymentAssistantPanel from './components/DeploymentAssistantPanel.vue';
import EvaluationPanel from './components/EvaluationPanel.vue';
import LoginPanel from './components/LoginPanel.vue';
import ReportPanel from './components/ReportPanel.vue';
import ReviewPanel from './components/ReviewPanel.vue';
import RuntimeConfigPanel from './components/RuntimeConfigPanel.vue';
import SubmissionDetailPanel from './components/SubmissionDetailPanel.vue';
import TeacherSetupPanel from './components/TeacherSetupPanel.vue';
import { api } from './lib/api';
import type { ActorProfile, ActorRole, AssistantConversationDetail, AssistantToolCall, BootstrapStatus, ClassRecord, CourseRecord, CourseReportSummary, EvaluationResultDetail, ExperimentRecord, ExperimentReportSummary, ReportExport, RubricTemplateRecord, RubricVersionRecord, RuntimeConfigSummary, Submission, SubmissionDetail, SubmissionReport, TeacherReviewDetail } from './lib/types';
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
const users = ref<ActorProfile[]>([]);
const classes = ref<ClassRecord[]>([]);
const courses = ref<CourseRecord[]>([]);
const templates = ref<RubricTemplateRecord[]>([]);
const versions = ref<RubricVersionRecord[]>([]);
const experiments = ref<ExperimentRecord[]>([]);
const studentExperiments = ref<ExperimentRecord[]>([]);
const studentSubmissions = ref<Submission[]>([]);

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
    await loadStudentSubmissions();
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
    if (!submissionID.value && response.items[0]) {
      submissionID.value = response.items[0].id;
    }
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

async function loadClasses() {
  const response = await api.listClasses(requestOptions.value);
  classes.value = response.items;
}

async function loadCourses() {
  const response = await api.listCourses(requestOptions.value);
  courses.value = response.items;
  if (!courseID.value && response.items[0]) {
    courseID.value = response.items[0].id;
  }
}

async function loadTeacherCourses() {
  const response = await api.listTeacherCourses(requestOptions.value);
  courses.value = response.items;
  if (!courseID.value && response.items[0]) {
    courseID.value = response.items[0].id;
  }
}

async function loadTemplates() {
  const response = await api.listRubricTemplates(requestOptions.value);
  templates.value = response.items;
  const versionResponses = await Promise.all(response.items.map((item) => api.listRubricVersions(item.id, requestOptions.value)));
  versions.value = versionResponses.flatMap((item) => item.items);
}

async function loadExperimentsForCourse() {
  if (!courseID.value) {
    experiments.value = [];
    return;
  }
  const response = await api.listTeacherExperiments(courseID.value, requestOptions.value);
  experiments.value = response.items;
  if (!experimentID.value && response.items[0]) {
    experimentID.value = response.items[0].id;
  }
}

async function loadStudentExperiments() {
  const response = await api.listStudentExperiments(requestOptions.value);
  studentExperiments.value = response.items;
  if (!experimentID.value && response.items[0]) {
    experimentID.value = response.items[0].id;
  }
}

async function loadStudentSubmissions() {
  const response = await api.listStudentSubmissions(experimentID.value, requestOptions.value);
  studentSubmissions.value = response.items;
  if (!submissionID.value && response.items[0]) {
    submissionID.value = response.items[0].id;
  }
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

async function exportSubmissionReport(format: 'html' | 'csv' | 'xlsx' | 'pdf') {
  await runAction(`导出个人报告 ${format}`, async () => {
    exportResult.value = await api.createSubmissionReportExport(submissionID.value, format, requestOptions.value);
  });
}

async function exportExperimentSummary(format: 'html' | 'csv' | 'xlsx' | 'pdf') {
  await runAction(`导出实验统计 ${format}`, async () => {
    exportResult.value = await api.createExperimentSummaryExport(experimentID.value, format, requestOptions.value);
  });
}

async function exportCourseSummary(format: 'html' | 'csv' | 'xlsx' | 'pdf') {
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

async function loadUsers() {
  await runAction('读取用户列表', async () => {
    const response = await api.listUsers(requestOptions.value);
    users.value = response.items;
  });
}

async function createUser(payload: { username: string; display_name: string; password?: string; email?: string; student_no?: string; employee_no?: string; roles: string[] }) {
  await runAction('创建用户', async () => {
    const created = await api.createUser(payload, requestOptions.value);
    users.value = [created, ...users.value];
  });
}

async function createClass(payload: { code: string; name: string; grade_year?: number; major?: string }) {
  await runAction('创建班级', async () => {
    const created = await api.createClass(payload, requestOptions.value);
    classes.value = [created, ...classes.value];
  });
}

async function createCourse(payload: { code: string; name: string; term: string }) {
  await runAction('创建课程', async () => {
    const created = await api.createCourse(payload, requestOptions.value);
    courses.value = [created, ...courses.value];
    courseID.value = created.id;
  });
}

async function addCourseClass(payload: { courseID: string; classID: string }) {
  await runAction('关联课程班级', async () => {
    await api.addCourseClass(payload.courseID, payload.classID, requestOptions.value);
    courseID.value = payload.courseID;
  });
}

async function assignTeacher(payload: { courseID: string; teacherID: string }) {
  await runAction('分配教师', async () => {
    await api.assignTeacher(payload.courseID, payload.teacherID, requestOptions.value);
    courseID.value = payload.courseID;
  });
}

async function enrollStudent(payload: { courseID: string; studentID: string; classID?: string }) {
  await runAction('登记选课', async () => {
    await api.enrollStudent(payload.courseID, { student_id: payload.studentID, class_id: payload.classID }, requestOptions.value);
    courseID.value = payload.courseID;
  });
}

async function createTemplate(payload: { name: string; description?: string }) {
  await runAction('创建模板', async () => {
    const created = await api.createRubricTemplate(payload, requestOptions.value);
    templates.value = [created, ...templates.value];
  });
}

async function createVersion(payload: { templateID: string; weight_mode: string; metrics: Array<{ code: string; name: string; description?: string; weight_bps: number; max_score: number; sort_order: number }> }) {
  await runAction('创建模板版本', async () => {
    const created = await api.createRubricVersion(payload.templateID, { weight_mode: payload.weight_mode, metrics: payload.metrics }, requestOptions.value);
    versions.value = [created.version, ...versions.value];
  });
}

async function publishVersion(versionID: string) {
  await runAction('发布模板版本', async () => {
    const published = await api.publishRubricVersion(versionID, requestOptions.value);
    versions.value = versions.value.map((item) => (item.id === published.id ? published : item));
  });
}

async function createExperimentFromSetup(payload: { courseID: string; title: string; description?: string; rubric_version_id: string; submission_spec?: Record<string, unknown> }) {
  await runAction('创建实验', async () => {
    const created = await api.createExperiment(payload.courseID, payload, requestOptions.value);
    experiments.value = [created, ...experiments.value];
    courseID.value = payload.courseID;
    experimentID.value = created.id;
  });
}

async function publishExperimentFromSetup(experimentIDValue: string) {
  await runAction('发布实验', async () => {
    const published = await api.publishExperiment(experimentIDValue, requestOptions.value);
    experiments.value = experiments.value.map((item) => (item.id === published.id ? published : item));
    experimentID.value = published.id;
  });
}

async function setUserPassword(payload: { userID: string; password: string }) {
  const resettingSelf = payload.userID === actorID.value;
  await runAction('设置用户密码', async () => {
    await api.setUserPassword(payload.userID, payload.password, requestOptions.value);
    if (resettingSelf) {
      resetAuthenticatedState();
      return;
    }
    await loadUsers();
  });
  if (resettingSelf && !loggedIn.value) {
    message.value = '当前账号密码已更新，请重新登录';
  }
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
    await loadWorkspace();
  });
}

async function logout() {
  await runAction('退出登录', async () => {
    await api.logout();
    resetAuthenticatedState();
  });
}

async function changeOwnPassword(payload: { current_password: string; new_password: string }) {
  await runAction('修改当前密码', async () => {
    await api.changeOwnPassword(payload);
    resetAuthenticatedState();
  });
  if (!loggedIn.value) {
    message.value = '密码已更新，请重新登录';
  }
}

function onFileChange(event: Event) {
  selectedFile.value = (event.target as HTMLInputElement).files?.[0] ?? null;
}

function resetAuthenticatedState() {
  loggedIn.value = false;
  actorID.value = '';
  roles.value = [];
  runtimeConfig.value = null;
  deploymentAssistant.value = null;
  users.value = [];
  classes.value = [];
  courses.value = [];
  templates.value = [];
  versions.value = [];
  experiments.value = [];
  studentExperiments.value = [];
  studentSubmissions.value = [];
  detail.value = null;
  evaluation.value = null;
  review.value = null;
  report.value = null;
  summary.value = null;
  courseSummary.value = null;
  exportResult.value = null;
  courseID.value = '';
  experimentID.value = '';
  submissionID.value = '';
}

async function loadWorkspace() {
  if (!loggedIn.value) {
    return;
  }
  if (roles.value.includes('admin')) {
    await Promise.all([loadUsers(), loadClasses(), loadCourses(), loadRuntimeConfig(), loadTemplates()]);
    await loadExperimentsForCourse();
    return;
  }
  if (roles.value.includes('teacher')) {
    await Promise.all([loadTeacherCourses(), loadTemplates()]);
    await loadExperimentsForCourse();
    return;
  }
  if (roles.value.includes('student')) {
    await loadStudentExperiments();
    await loadStudentSubmissions();
  }
}

async function selectCourse(value: string) {
  courseID.value = value;
  experimentID.value = '';
  submissionID.value = '';
  submissions.value = [];
  studentSubmissions.value = [];
  detail.value = null;
  evaluation.value = null;
  review.value = null;
  report.value = null;
  summary.value = null;
  courseSummary.value = null;
  await runAction('加载课程实验', async () => {
    await loadExperimentsForCourse();
  });
}

async function selectExperiment(value: string) {
  experimentID.value = value;
  submissionID.value = '';
  detail.value = null;
  evaluation.value = null;
  review.value = null;
  report.value = null;
  summary.value = null;
  if (roles.value.includes('student')) {
    await runAction('加载学生提交列表', async () => {
      await loadStudentSubmissions();
    });
    return;
  }
  await listSubmissions();
}

onMounted(() => {
  void (async () => {
    await loadBootstrapStatus();
    if (bootstrapStatus.value?.initialized) {
      await loadCurrentUser();
      if (loggedIn.value) {
        await loadWorkspace();
      }
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

    <AccountSecurityPanel
      v-if="bootstrapStatus?.initialized && loggedIn"
      :busy="busy"
      @change-password="changeOwnPassword"
    />

    <AdminSetupPanel
      v-if="bootstrapStatus?.initialized && loggedIn && roles.includes('admin')"
      :busy="busy"
      :users="users"
      :classes="classes"
      :courses="courses"
      @create-user="createUser"
      @create-class="createClass"
      @create-course="createCourse"
      @add-course-class="addCourseClass"
      @assign-teacher="assignTeacher"
      @enroll-student="enrollStudent"
    />

    <TeacherSetupPanel
      v-if="bootstrapStatus?.initialized && loggedIn && (roles.includes('teacher') || roles.includes('admin'))"
      :busy="busy"
      :courses="courses"
      :templates="templates"
      :versions="versions"
      :experiments="experiments"
      :active-course-id="courseID"
      @create-template="createTemplate"
      @create-version="createVersion"
      @publish-version="publishVersion"
      @create-experiment="createExperimentFromSetup"
      @publish-experiment="publishExperimentFromSetup"
      @select-course="selectCourse"
      @select-experiment="selectExperiment"
    />

    <RuntimeConfigPanel
      v-if="bootstrapStatus?.initialized && loggedIn && roles.includes('admin')"
      :busy="busy"
      :summary="runtimeConfig"
      @load="loadRuntimeConfig"
      @save="saveRuntimeConfig"
    />

    <AdminUserPanel
      v-if="bootstrapStatus?.initialized && loggedIn && roles.includes('admin')"
      :busy="busy"
      :users="users"
      @refresh="loadUsers"
      @set-password="setUserPassword"
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
        <h2>选择任务、提交成果并查看结果</h2>
        <label>
          可提交实验
          <select v-model="experimentID" @change="selectExperiment(($event.target as HTMLSelectElement).value)">
            <option value="">选择实验</option>
            <option v-for="item in studentExperiments" :key="item.id" :value="item.id">
              {{ item.title }} · {{ item.status }}
            </option>
          </select>
        </label>
        <label>
          我的提交
          <select v-model="submissionID">
            <option value="">选择提交</option>
            <option v-for="item in studentSubmissions" :key="item.id" :value="item.id">
              {{ item.id }} · {{ item.status }} · attempt {{ item.attempt_no }}
            </option>
          </select>
        </label>
        <div class="button-row">
          <button :disabled="busy || !experimentID" @click="createSubmission">创建提交</button>
          <button :disabled="busy || !submissionID" @click="loadStudentSubmission">读取提交</button>
          <button :disabled="busy || !submissionID" @click="loadReview('student')">读取已发布评价</button>
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
        <h2>选择课程、实验和提交后复核发布</h2>
        <label>
          课程
          <select v-model="courseID" @change="selectCourse(($event.target as HTMLSelectElement).value)">
            <option value="">选择课程</option>
            <option v-for="course in courses" :key="course.id" :value="course.id">
              {{ course.name }} · {{ course.term }}
            </option>
          </select>
        </label>
        <label>
          实验
          <select v-model="experimentID" @change="selectExperiment(($event.target as HTMLSelectElement).value)">
            <option value="">选择实验</option>
            <option v-for="item in experiments" :key="item.id" :value="item.id">
              {{ item.title }} · {{ item.status }}
            </option>
          </select>
        </label>
        <button :disabled="busy || !experimentID" @click="listSubmissions">查看提交列表</button>
        <div class="submission-list">
          <button v-for="item in submissions" :key="item.id" @click="loadTeacherSubmission(item.id)">
            {{ item.id }} · {{ item.student_id }} · {{ item.status }}
          </button>
        </div>
        <label>
          当前提交
          <select v-model="submissionID">
            <option value="">选择提交</option>
            <option v-for="item in submissions" :key="item.id" :value="item.id">
              {{ item.id }} · {{ item.student_id }} · {{ item.status }}
            </option>
          </select>
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
      <SubmissionDetailPanel :detail="detail" :evaluation="evaluation" :review="review" />
      <EvaluationPanel :detail="detail" :evaluation="evaluation" />
      <ReviewPanel :busy="busy" :detail="detail" :evaluation="evaluation" :review="review" @save="saveReview" @publish="publishReview" />
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
      <p>学生现在可以直接从实验和提交下拉框中选择自己的记录，不再依赖手工填写 ID。</p>
      <button :disabled="busy || !submissionID" @click="loadReview('student')">读取已发布评价</button>
    </section>
  </main>
</template>
