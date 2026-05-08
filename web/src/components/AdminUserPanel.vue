<script setup lang="ts">
import { reactive } from 'vue';
import type { ActorProfile } from '../lib/types';

const props = defineProps<{
  users: ActorProfile[];
  busy: boolean;
}>();

const emit = defineEmits<{
  refresh: [];
  setPassword: [payload: { userID: string; password: string }];
}>();

const drafts = reactive<Record<string, string>>({});

function submit(userID: string) {
  emit('setPassword', { userID, password: drafts[userID] || '' });
  drafts[userID] = '';
}
</script>

<template>
  <section class="card user-admin-card">
    <div class="panel-heading split-heading">
      <div>
        <p class="eyebrow">用户管理</p>
        <h2>重置用户密码</h2>
      </div>
      <strong class="score-badge">{{ users.length }} users</strong>
    </div>

    <div class="button-row">
      <button :disabled="busy" @click="emit('refresh')">刷新用户列表</button>
    </div>

    <div class="assistant-thread">
      <article v-for="user in users" :key="user.id" class="assistant-message">
        <span>{{ user.roles.join(', ') }}</span>
        <p>{{ user.display_name || user.id }}</p>
        <label>
          新密码
          <input v-model="drafts[user.id]" type="password" placeholder="至少一位" />
        </label>
        <div class="button-row">
          <button :disabled="busy || !(drafts[user.id] || '').trim()" @click="submit(user.id)">设置密码</button>
        </div>
      </article>
    </div>
  </section>
</template>
