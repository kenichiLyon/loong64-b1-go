<script setup lang="ts">
import { reactive } from 'vue';

const props = defineProps<{
  busy: boolean;
}>();

const emit = defineEmits<{
  login: [payload: { username: string; password: string }];
}>();

const form = reactive({
  username: 'admin1',
  password: '',
});

function submit() {
  emit('login', { username: form.username, password: form.password });
  form.password = '';
}
</script>

<template>
  <section class="card login-card">
    <div class="panel-heading split-heading">
      <div>
        <p class="eyebrow">登录</p>
        <h2>使用会话进入系统</h2>
      </div>
      <strong class="score-badge">session</strong>
    </div>
    <label>
      用户名
      <input v-model="form.username" placeholder="admin1" />
    </label>
    <label>
      密码
      <input v-model="form.password" type="password" placeholder="password" />
    </label>
    <div class="button-row">
      <button :disabled="busy || !form.username.trim() || !form.password.trim()" @click="submit">登录</button>
    </div>
    <p class="muted">默认主链路现在优先使用 httpOnly session cookie。`X-Actor-*` 只保留给开发态 bypass。</p>
  </section>
</template>
