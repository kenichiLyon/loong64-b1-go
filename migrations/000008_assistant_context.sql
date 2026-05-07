CREATE TABLE IF NOT EXISTS assistant_conversations (
  id text PRIMARY KEY,
  scope_type text NOT NULL CHECK (scope_type IN ('bootstrap', 'deployment_admin')),
  actor_id text,
  status text NOT NULL CHECK (status IN ('active', 'closed')),
  model text NOT NULL DEFAULT '',
  prompt_version text NOT NULL,
  summary_text text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  last_message_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_assistant_conversations_scope_updated ON assistant_conversations (scope_type, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_assistant_conversations_actor_updated ON assistant_conversations (actor_id, updated_at DESC) WHERE actor_id IS NOT NULL AND actor_id <> '';

CREATE TABLE IF NOT EXISTS assistant_context_snapshots (
  id text PRIMARY KEY,
  conversation_id text NOT NULL REFERENCES assistant_conversations(id) ON DELETE CASCADE,
  scope_stage text NOT NULL CHECK (scope_stage IN ('bootstrap_status', 'runtime_config', 'db_connectivity', 'admin_init')),
  payload_json jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_assistant_context_snapshots_conversation_created ON assistant_context_snapshots (conversation_id, created_at DESC);

CREATE TABLE IF NOT EXISTS assistant_tool_calls (
  id text PRIMARY KEY,
  conversation_id text NOT NULL REFERENCES assistant_conversations(id) ON DELETE CASCADE,
  tool_name text NOT NULL CHECK (tool_name IN ('inspect_bootstrap_status', 'inspect_runtime_config', 'test_sqlite_path', 'test_postgres_connection', 'save_runtime_config', 'bootstrap_create_admin')),
  status text NOT NULL CHECK (status IN ('pending_confirmation', 'running', 'succeeded', 'failed', 'cancelled')),
  request_json jsonb NOT NULL DEFAULT '{}'::jsonb,
  response_json jsonb NOT NULL DEFAULT '{}'::jsonb,
  error text NOT NULL DEFAULT '',
  confirmed_by_actor text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_assistant_tool_calls_conversation_created ON assistant_tool_calls (conversation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_assistant_tool_calls_status_created ON assistant_tool_calls (status, created_at DESC);

CREATE TABLE IF NOT EXISTS assistant_messages (
  id text PRIMARY KEY,
  conversation_id text NOT NULL REFERENCES assistant_conversations(id) ON DELETE CASCADE,
  role text NOT NULL CHECK (role IN ('user', 'assistant', 'tool')),
  content_text text NOT NULL DEFAULT '',
  context_snapshot_id text REFERENCES assistant_context_snapshots(id) ON DELETE SET NULL,
  tool_call_id text REFERENCES assistant_tool_calls(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_assistant_messages_conversation_created ON assistant_messages (conversation_id, created_at DESC);

CREATE TABLE IF NOT EXISTS assistant_llm_calls (
  id text PRIMARY KEY,
  conversation_id text NOT NULL REFERENCES assistant_conversations(id) ON DELETE CASCADE,
  provider text NOT NULL DEFAULT 'openai-compatible',
  model text NOT NULL DEFAULT '',
  prompt_version text NOT NULL,
  input_hash text NOT NULL,
  output jsonb NOT NULL DEFAULT '{}'::jsonb,
  status text NOT NULL CHECK (status IN ('succeeded', 'failed', 'skipped')),
  error text NOT NULL DEFAULT '',
  latency_ms integer NOT NULL DEFAULT 0 CHECK (latency_ms >= 0),
  prompt_tokens integer NOT NULL DEFAULT 0 CHECK (prompt_tokens >= 0),
  completion_tokens integer NOT NULL DEFAULT 0 CHECK (completion_tokens >= 0),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_assistant_llm_calls_conversation_created ON assistant_llm_calls (conversation_id, created_at DESC);
