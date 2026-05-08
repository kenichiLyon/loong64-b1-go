<script setup lang="ts">
import { reactive } from 'vue';

const props = defineProps<{
  busy: boolean;
}>();

const emit = defineEmits<{
  changePassword: [payload: { current_password: string; new_password: string }];
}>();

const form = reactive({
  currentPassword: '',
  newPassword: '',
  confirmPassword: '',
});

function submit() {
  emit('changePassword', {
    current_password: form.currentPassword,
    new_password: form.newPassword,
  });
  form.currentPassword = '';
  form.newPassword = '';
  form.confirmPassword = '';
}
</script>

<template>
  <section class="card account-security-card">
    <div class="panel-heading split-heading">
      <div>
        <p class="eyebrow">账号安全</p>
        <h2>修改当前密码</h2>
      </div>
      <strong class="score-badge">re-login</strong>
    </div>
    <label>
      当前密码
      <input v-model="form.currentPassword" type="password" placeholder="输入当前密码" />
    </label>
    <label>
      新密码
      <input v-model="form.newPassword" type="password" placeholder="输入新密码" />
    </label>
    <label>
      确认新密码
      <input v-model="form.confirmPassword" type="password" placeholder="再次输入新密码" />
    </label>
    <div class="button-row">
      <button
        :disabled="
          busy ||
          !form.currentPassword.trim() ||
          !form.newPassword.trim() ||
          form.newPassword !== form.confirmPassword
        "
        @click="submit"
      >
        修改密码
      </button>
    </div>
    <p class="muted">修改成功后，当前和其他已登录会话都会失效，需要重新登录。</p>
  </section>
</template>
