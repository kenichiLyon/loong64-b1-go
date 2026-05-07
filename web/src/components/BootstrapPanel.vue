<script setup lang="ts">
import { reactive } from 'vue';
import type { BootstrapStatus } from '../lib/types';

const props = defineProps<{
  status: BootstrapStatus | null;
  busy: boolean;
}>();

const emit = defineEmits<{
  createAdmin: [payload: { username: string; display_name: string; email?: string; employee_no?: string }];
}>();

const form = reactive({
  username: 'admin',
  display_name: 'Bootstrap Admin',
  email: '',
  employee_no: 'A001',
});

function submit() {
  emit('createAdmin', {
    username: form.username,
    display_name: form.display_name,
    email: form.email || undefined,
    employee_no: form.employee_no || undefined,
  });
}
</script>

<template>
  <section class="card bootstrap-card" v-if="status && !status.initialized">
    <div class="panel-heading split-heading">
      <div>
        <p class="eyebrow">首次启动</p>
        <h2>创建首个管理员</h2>
      </div>
      <strong class="score-badge">{{ status.runtime.db_driver }}</strong>
    </div>

    <p class="muted">{{ status.message }}</p>

    <div class="facts-grid compact-facts">
      <div>
        <span>当前数据库</span>
        <strong>{{ status.runtime.db_driver }}</strong>
      </div>
      <div>
        <span>SQLite 路径</span>
        <strong>{{ status.runtime.sqlite_path || 'n/a' }}</strong>
      </div>
      <div>
        <span>已保存配置</span>
        <strong>{{ status.stored ? status.stored.db_driver : 'none' }}</strong>
      </div>
      <div>
        <span>自动迁移</span>
        <strong>{{ status.runtime.auto_migrate ? 'on' : 'off' }}</strong>
      </div>
    </div>

    <label>
      用户名
      <input v-model="form.username" placeholder="admin" />
    </label>
    <label>
      显示名
      <input v-model="form.display_name" placeholder="Bootstrap Admin" />
    </label>
    <label>
      工号
      <input v-model="form.employee_no" placeholder="A001" />
    </label>
    <label>
      邮箱（可选）
      <input v-model="form.email" placeholder="admin@example.edu" />
    </label>

    <div class="button-row">
      <button :disabled="busy" @click="submit">创建首个管理员</button>
    </div>

    <p class="muted">数据库切换仍通过“运行配置”写入 `runtime.json`，保存后需重启。当前引导只负责当前数据库中的首个管理员初始化。</p>
  </section>
</template>
