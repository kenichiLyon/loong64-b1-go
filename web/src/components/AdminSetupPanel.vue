<script setup lang="ts">
import { computed, reactive } from 'vue';
import type { ActorProfile, ClassRecord, CourseRecord } from '../lib/types';

const props = defineProps<{
  busy: boolean;
  users: ActorProfile[];
  classes: ClassRecord[];
  courses: CourseRecord[];
}>();

const emit = defineEmits<{
  createUser: [payload: { username: string; display_name: string; password?: string; email?: string; student_no?: string; employee_no?: string; roles: string[] }];
  createClass: [payload: { code: string; name: string; grade_year?: number; major?: string }];
  createCourse: [payload: { code: string; name: string; term: string }];
  addCourseClass: [payload: { courseID: string; classID: string }];
  assignTeacher: [payload: { courseID: string; teacherID: string }];
  enrollStudent: [payload: { courseID: string; studentID: string; classID?: string }];
}>();

const userForm = reactive({
  username: '',
  display_name: '',
  password: '',
  role: 'teacher',
  employee_no: '',
  student_no: '',
});

const classForm = reactive({
  code: '',
  name: '',
  grade_year: new Date().getFullYear(),
  major: '',
});

const courseForm = reactive({
  code: '',
  name: '',
  term: '2026-spring',
});

const bindForm = reactive({
  courseID: '',
  classID: '',
  teacherID: '',
  studentID: '',
});

const teachers = computed(() => props.users.filter((item) => item.roles.includes('teacher')));
const students = computed(() => props.users.filter((item) => item.roles.includes('student')));

function submitUser() {
  emit('createUser', {
    username: userForm.username,
    display_name: userForm.display_name,
    password: userForm.password || undefined,
    employee_no: userForm.role === 'teacher' ? userForm.employee_no || undefined : undefined,
    student_no: userForm.role === 'student' ? userForm.student_no || undefined : undefined,
    roles: [userForm.role],
  });
  userForm.username = '';
  userForm.display_name = '';
  userForm.password = '';
  userForm.employee_no = '';
  userForm.student_no = '';
}

function submitClass() {
  emit('createClass', {
    code: classForm.code,
    name: classForm.name,
    grade_year: classForm.grade_year || undefined,
    major: classForm.major || undefined,
  });
  classForm.code = '';
  classForm.name = '';
  classForm.major = '';
}

function submitCourse() {
  emit('createCourse', {
    code: courseForm.code,
    name: courseForm.name,
    term: courseForm.term,
  });
  courseForm.code = '';
  courseForm.name = '';
}
</script>

<template>
  <section class="card setup-panel">
    <div class="panel-heading split-heading">
      <div>
        <p class="eyebrow">管理搭建</p>
        <h2>创建用户、班级和课程</h2>
      </div>
      <strong class="score-badge">admin</strong>
    </div>

    <div class="setup-grid">
      <div class="setup-block">
        <h3>创建用户</h3>
        <label>用户名<input v-model="userForm.username" placeholder="teacher1" /></label>
        <label>显示名<input v-model="userForm.display_name" placeholder="Teacher One" /></label>
        <label>角色
          <select v-model="userForm.role">
            <option value="teacher">teacher</option>
            <option value="student">student</option>
            <option value="admin">admin</option>
          </select>
        </label>
        <label v-if="userForm.role === 'teacher'">工号<input v-model="userForm.employee_no" placeholder="T001" /></label>
        <label v-if="userForm.role === 'student'">学号<input v-model="userForm.student_no" placeholder="S001" /></label>
        <label>初始密码<input v-model="userForm.password" type="password" placeholder="可选" /></label>
        <button :disabled="busy || !userForm.username.trim() || !userForm.display_name.trim()" @click="submitUser">创建用户</button>
      </div>

      <div class="setup-block">
        <h3>创建班级</h3>
        <label>班级代码<input v-model="classForm.code" placeholder="SE2401" /></label>
        <label>班级名称<input v-model="classForm.name" placeholder="软件工程 2401" /></label>
        <label>入学年<input v-model.number="classForm.grade_year" type="number" min="2000" max="2100" /></label>
        <label>专业<input v-model="classForm.major" placeholder="软件工程" /></label>
        <button :disabled="busy || !classForm.code.trim() || !classForm.name.trim()" @click="submitClass">创建班级</button>
      </div>

      <div class="setup-block">
        <h3>创建课程</h3>
        <label>课程代码<input v-model="courseForm.code" placeholder="SE-LAB-01" /></label>
        <label>课程名称<input v-model="courseForm.name" placeholder="软件实训一" /></label>
        <label>学期<input v-model="courseForm.term" placeholder="2026-spring" /></label>
        <button :disabled="busy || !courseForm.code.trim() || !courseForm.name.trim() || !courseForm.term.trim()" @click="submitCourse">创建课程</button>
      </div>

      <div class="setup-block">
        <h3>绑定课程</h3>
        <label>课程
          <select v-model="bindForm.courseID">
            <option value="">选择课程</option>
            <option v-for="course in courses" :key="course.id" :value="course.id">{{ course.name }} · {{ course.id }}</option>
          </select>
        </label>
        <label>班级
          <select v-model="bindForm.classID">
            <option value="">选择班级</option>
            <option v-for="item in classes" :key="item.id" :value="item.id">{{ item.name }} · {{ item.id }}</option>
          </select>
        </label>
        <button :disabled="busy || !bindForm.courseID || !bindForm.classID" @click="emit('addCourseClass', { courseID: bindForm.courseID, classID: bindForm.classID })">关联课程班级</button>
        <label>教师
          <select v-model="bindForm.teacherID">
            <option value="">选择教师</option>
            <option v-for="teacher in teachers" :key="teacher.id" :value="teacher.id">{{ teacher.display_name || teacher.username || teacher.id }}</option>
          </select>
        </label>
        <button :disabled="busy || !bindForm.courseID || !bindForm.teacherID" @click="emit('assignTeacher', { courseID: bindForm.courseID, teacherID: bindForm.teacherID })">分配教师</button>
        <label>学生
          <select v-model="bindForm.studentID">
            <option value="">选择学生</option>
            <option v-for="student in students" :key="student.id" :value="student.id">{{ student.display_name || student.username || student.id }}</option>
          </select>
        </label>
        <button :disabled="busy || !bindForm.courseID || !bindForm.studentID" @click="emit('enrollStudent', { courseID: bindForm.courseID, studentID: bindForm.studentID, classID: bindForm.classID || undefined })">登记选课</button>
      </div>
    </div>
  </section>
</template>
