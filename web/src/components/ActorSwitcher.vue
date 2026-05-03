<script setup lang="ts">
import type { ActorRole } from '../lib/types';

const actorId = defineModel<string>('actorId', { required: true });
const roles = defineModel<ActorRole[]>('roles', { required: true });

const roleOptions: ActorRole[] = ['student', 'teacher', 'admin'];
</script>

<template>
  <section class="card identity-card">
    <div>
      <p class="eyebrow">开发态身份</p>
      <h2>用请求头模拟登录</h2>
    </div>
    <label>
      Actor ID
      <input v-model="actorId" placeholder="teacher-1" />
    </label>
    <div class="role-row" aria-label="角色选择">
      <label v-for="role in roleOptions" :key="role" class="role-pill">
        <input v-model="roles" type="checkbox" :value="role" />
        {{ role }}
      </label>
    </div>
  </section>
</template>
