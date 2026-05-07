<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import type { AssistantConversationDetail, AssistantToolCall, BootstrapStatus, RuntimeConfigSummary } from '../lib/types';

const props = defineProps<{
  scope: 'bootstrap' | 'deployment_admin';
  detail: AssistantConversationDetail | null;
  busy: boolean;
  bootstrapStatus: BootstrapStatus | null;
  runtimeConfig: RuntimeConfigSummary | null;
}>();

const emit = defineEmits<{
  ensureConversation: [];
  send: [content: string];
  confirm: [toolCall: AssistantToolCall, inputs: Record<string, unknown>];
}>();

const draftMessage = ref('');
const draftTool = reactive({
  username: 'admin',
  display_name: 'Bootstrap Admin',
  employee_no: 'A001',
  email: '',
  db_driver: 'sqlite' as 'sqlite' | 'postgres',
  sqlite_path: './data/loong64-b1-go.db',
  database_url: '',
  auto_migrate: true,
});

const pendingTool = computed(() => props.detail?.pending_tool_call ?? null);

onMounted(() => {
  emit('ensureConversation');
});

function sendMessage() {
  if (!draftMessage.value.trim()) {
    return;
  }
  emit('send', draftMessage.value);
  draftMessage.value = '';
}

function confirmTool() {
  if (!pendingTool.value) {
    return;
  }
  let inputs: Record<string, unknown> = {};
  switch (pendingTool.value.tool_name) {
    case 'bootstrap_create_admin':
      inputs = {
        username: draftTool.username,
        display_name: draftTool.display_name,
        employee_no: draftTool.employee_no || undefined,
        email: draftTool.email || undefined,
      };
      break;
    case 'test_sqlite_path':
      inputs = { sqlite_path: draftTool.sqlite_path };
      break;
    case 'test_postgres_connection':
      inputs = { database_url: draftTool.database_url };
      break;
    case 'save_runtime_config':
      inputs = {
        db_driver: draftTool.db_driver,
        sqlite_path: draftTool.db_driver === 'sqlite' ? draftTool.sqlite_path : undefined,
        database_url: draftTool.db_driver === 'postgres' ? draftTool.database_url : undefined,
        auto_migrate: draftTool.auto_migrate,
      };
      break;
    default:
      inputs = {};
  }
  emit('confirm', pendingTool.value, inputs);
}
</script>

<template>
  <section class="card assistant-card">
    <div class="panel-heading split-heading">
      <div>
        <p class="eyebrow">部署助手</p>
        <h2>{{ scope === 'bootstrap' ? '首次启动对话' : '运行配置对话' }}</h2>
      </div>
      <strong class="score-badge">{{ detail?.conversation.scope_type || scope }}</strong>
    </div>

    <div class="assistant-meta" v-if="bootstrapStatus && scope === 'bootstrap'">
      <span>initialized={{ bootstrapStatus.initialized }}</span>
      <span>users={{ bootstrapStatus.user_count }}</span>
    </div>
    <div class="assistant-meta" v-if="runtimeConfig && scope === 'deployment_admin'">
      <span>driver={{ runtimeConfig.active.db_driver }}</span>
      <span>auto_migrate={{ runtimeConfig.active.auto_migrate ? 'on' : 'off' }}</span>
    </div>

    <div class="assistant-thread">
      <article v-for="message in detail?.messages || []" :key="message.id" :class="['assistant-message', message.role]">
        <span>{{ message.role }}</span>
        <p>{{ message.content_text }}</p>
      </article>
    </div>

    <div v-if="pendingTool" class="assistant-tool">
      <strong>{{ pendingTool.tool_name }}</strong>
      <template v-if="pendingTool.tool_name === 'bootstrap_create_admin'">
        <label>
          用户名
          <input v-model="draftTool.username" />
        </label>
        <label>
          显示名
          <input v-model="draftTool.display_name" />
        </label>
        <label>
          工号
          <input v-model="draftTool.employee_no" />
        </label>
        <label>
          邮箱
          <input v-model="draftTool.email" />
        </label>
      </template>
      <template v-else-if="pendingTool.tool_name === 'test_sqlite_path'">
        <label>
          SQLite 路径
          <input v-model="draftTool.sqlite_path" />
        </label>
      </template>
      <template v-else-if="pendingTool.tool_name === 'test_postgres_connection'">
        <label>
          PostgreSQL URL
          <input v-model="draftTool.database_url" />
        </label>
      </template>
      <template v-else-if="pendingTool.tool_name === 'save_runtime_config'">
        <label>
          数据库驱动
          <select v-model="draftTool.db_driver">
            <option value="sqlite">sqlite</option>
            <option value="postgres">postgres</option>
          </select>
        </label>
        <label v-if="draftTool.db_driver === 'sqlite'">
          SQLite 路径
          <input v-model="draftTool.sqlite_path" />
        </label>
        <label v-else>
          PostgreSQL URL
          <input v-model="draftTool.database_url" />
        </label>
        <label class="runtime-toggle">
          <input v-model="draftTool.auto_migrate" type="checkbox" />
          <span>启动时自动迁移</span>
        </label>
      </template>
      <div class="button-row">
        <button :disabled="busy" @click="confirmTool">确认执行工具</button>
      </div>
    </div>

    <label>
      发送给部署助手
      <textarea v-model="draftMessage" placeholder="例如：帮我看一下当前为什么还不能初始化，或帮我把数据库切到 postgres 并测试连接"></textarea>
    </label>
    <div class="button-row">
      <button :disabled="busy || !draftMessage.trim()" @click="sendMessage">发送</button>
    </div>
  </section>
</template>
