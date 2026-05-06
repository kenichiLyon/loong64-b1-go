<script setup lang="ts">
import { reactive, watch } from 'vue';
import type { RuntimeConfigSummary } from '../lib/types';

const props = defineProps<{
  summary: RuntimeConfigSummary | null;
  busy: boolean;
}>();

const emit = defineEmits<{
  load: [];
  save: [payload: { db_driver: 'sqlite' | 'postgres'; sqlite_path?: string; database_url?: string; auto_migrate?: boolean }];
}>();

const form = reactive({
  db_driver: 'sqlite' as 'sqlite' | 'postgres',
  sqlite_path: './data/loong64-b1-go.db',
  database_url: '',
  auto_migrate: true,
});

watch(
  () => props.summary,
  (summary) => {
    const source = summary?.stored ?? summary?.active;
    if (!source) {
      return;
    }
    form.db_driver = (source.db_driver === 'postgres' ? 'postgres' : 'sqlite');
    form.sqlite_path = source.sqlite_path || './data/loong64-b1-go.db';
    form.database_url = source.database_url || '';
    form.auto_migrate = source.auto_migrate;
  },
  { immediate: true },
);

function submit() {
  emit('save', {
    db_driver: form.db_driver,
    sqlite_path: form.db_driver === 'sqlite' ? form.sqlite_path : undefined,
    database_url: form.db_driver === 'postgres' ? form.database_url : undefined,
    auto_migrate: form.auto_migrate,
  });
}
</script>

<template>
  <section class="card runtime-card">
    <div class="panel-heading split-heading">
      <div>
        <p class="eyebrow">运行配置</p>
        <h2>数据库切换</h2>
      </div>
      <strong class="score-badge">需重启</strong>
    </div>

    <div class="facts-grid compact-facts" v-if="summary">
      <div>
        <span>配置文件</span>
        <strong>{{ summary.path }}</strong>
      </div>
      <div>
        <span>已落盘</span>
        <strong>{{ summary.exists ? 'yes' : 'no' }}</strong>
      </div>
      <div>
        <span>当前驱动</span>
        <strong>{{ summary.active.db_driver }}</strong>
      </div>
      <div>
        <span>自动迁移</span>
        <strong>{{ summary.active.auto_migrate ? 'on' : 'off' }}</strong>
      </div>
    </div>

    <label>
      数据库驱动
      <select v-model="form.db_driver">
        <option value="sqlite">sqlite</option>
        <option value="postgres">postgres</option>
      </select>
    </label>

    <label v-if="form.db_driver === 'sqlite'">
      SQLite 路径
      <input v-model="form.sqlite_path" placeholder="./data/loong64-b1-go.db" />
    </label>

    <label v-else>
      PostgreSQL URL
      <input v-model="form.database_url" placeholder="postgres://user:password@127.0.0.1:5432/db?sslmode=disable" />
    </label>

    <label class="runtime-toggle">
      <input v-model="form.auto_migrate" type="checkbox" />
      <span>启动时自动迁移</span>
    </label>

    <div class="button-row">
      <button :disabled="busy" @click="emit('load')">读取配置</button>
      <button :disabled="busy" @click="submit">保存并等待重启</button>
    </div>

    <p class="muted" v-if="summary?.message">{{ summary.message }}</p>
    <p class="muted" v-else>当前只保存运行配置，不做热切换。保存后重启服务生效。</p>
  </section>
</template>
