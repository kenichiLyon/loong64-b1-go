package assistant

import (
	"context"
	"encoding/json"
	"time"
)

const PromptVersion = "deployment-assistant-v1"

type ScopeType string

const (
	ScopeBootstrap       ScopeType = "bootstrap"
	ScopeDeploymentAdmin ScopeType = "deployment_admin"
)

type ConversationStatus string

const (
	ConversationActive ConversationStatus = "active"
	ConversationClosed ConversationStatus = "closed"
)

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

type ToolName string

const (
	ToolInspectBootstrapStatus ToolName = "inspect_bootstrap_status"
	ToolInspectRuntimeConfig   ToolName = "inspect_runtime_config"
	ToolTestSQLitePath         ToolName = "test_sqlite_path"
	ToolTestPostgresConnection ToolName = "test_postgres_connection"
	ToolSaveRuntimeConfig      ToolName = "save_runtime_config"
	ToolBootstrapCreateAdmin   ToolName = "bootstrap_create_admin"
)

type ToolCallStatus string

const (
	ToolCallPendingConfirmation ToolCallStatus = "pending_confirmation"
	ToolCallRunning             ToolCallStatus = "running"
	ToolCallSucceeded           ToolCallStatus = "succeeded"
	ToolCallFailed              ToolCallStatus = "failed"
	ToolCallCancelled           ToolCallStatus = "cancelled"
)

type ContextStage string

const (
	ContextStageBootstrapStatus ContextStage = "bootstrap_status"
	ContextStageRuntimeConfig   ContextStage = "runtime_config"
	ContextStageDBConnectivity  ContextStage = "db_connectivity"
	ContextStageAdminInit       ContextStage = "admin_init"
)

type Conversation struct {
	ID            string             `json:"id"`
	ScopeType     ScopeType          `json:"scope_type"`
	ActorID       string             `json:"actor_id,omitempty"`
	Status        ConversationStatus `json:"status"`
	Model         string             `json:"model,omitempty"`
	PromptVersion string             `json:"prompt_version"`
	SummaryText   string             `json:"summary_text,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	LastMessageAt time.Time          `json:"last_message_at"`
}

type Message struct {
	ID                string           `json:"id"`
	ConversationID    string           `json:"conversation_id"`
	Role              MessageRole      `json:"role"`
	ContentText       string           `json:"content_text"`
	ContextSnapshotID string           `json:"context_snapshot_id,omitempty"`
	ToolCallID        string           `json:"tool_call_id,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	ContextSnapshot   *ContextSnapshot `json:"context_snapshot,omitempty"`
}

type ContextSnapshot struct {
	ID             string          `json:"id"`
	ConversationID string          `json:"conversation_id"`
	ScopeStage     ContextStage    `json:"scope_stage"`
	PayloadJSON    json.RawMessage `json:"payload_json"`
	CreatedAt      time.Time       `json:"created_at"`
}

type ToolCall struct {
	ID               string          `json:"id"`
	ConversationID   string          `json:"conversation_id"`
	ToolName         ToolName        `json:"tool_name"`
	Status           ToolCallStatus  `json:"status"`
	RequestJSON      json.RawMessage `json:"request_json"`
	ResponseJSON     json.RawMessage `json:"response_json"`
	Error            string          `json:"error,omitempty"`
	ConfirmedByActor string          `json:"confirmed_by_actor,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
}

type LLMCall struct {
	ID               string          `json:"id"`
	ConversationID   string          `json:"conversation_id"`
	Provider         string          `json:"provider"`
	Model            string          `json:"model,omitempty"`
	PromptVersion    string          `json:"prompt_version"`
	InputHash        string          `json:"input_hash"`
	Output           json.RawMessage `json:"output"`
	Status           string          `json:"status"`
	Error            string          `json:"error,omitempty"`
	LatencyMS        int             `json:"latency_ms"`
	PromptTokens     int             `json:"prompt_tokens"`
	CompletionTokens int             `json:"completion_tokens"`
	CreatedAt        time.Time       `json:"created_at"`
}

type ConversationDetail struct {
	Conversation    Conversation     `json:"conversation"`
	Messages        []Message        `json:"messages"`
	PendingToolCall *ToolCall        `json:"pending_tool_call,omitempty"`
	LatestSnapshot  *ContextSnapshot `json:"latest_context_snapshot,omitempty"`
}

type CreateConversationInput struct {
	ScopeType ScopeType `json:"scope_type"`
}

type SendMessageInput struct {
	Content string `json:"content"`
}

type ConfirmToolCallInput struct {
	Inputs json.RawMessage `json:"inputs,omitempty"`
}

type SendMessageResult struct {
	Conversation         Conversation    `json:"conversation"`
	AssistantMessage     Message         `json:"assistant_message"`
	PendingToolCall      *ToolCall       `json:"pending_tool_call,omitempty"`
	ContextSnapshot      ContextSnapshot `json:"context_snapshot"`
	RequiresConfirmation bool            `json:"requires_confirmation"`
}

type ConfirmToolCallResult struct {
	Conversation     Conversation `json:"conversation"`
	ToolCall         ToolCall     `json:"tool_call"`
	ToolMessage      Message      `json:"tool_message"`
	AssistantMessage Message      `json:"assistant_message"`
}

type Repository interface {
	CreateConversation(context.Context, Conversation) (Conversation, error)
	GetConversation(context.Context, string) (Conversation, error)
	UpdateConversation(context.Context, Conversation) (Conversation, error)
	CreateMessage(context.Context, Message) (Message, error)
	ListMessages(context.Context, string, int) ([]Message, error)
	CreateContextSnapshot(context.Context, ContextSnapshot) (ContextSnapshot, error)
	GetLatestContextSnapshot(context.Context, string) (ContextSnapshot, error)
	CreateToolCall(context.Context, ToolCall) (ToolCall, error)
	UpdateToolCall(context.Context, ToolCall) (ToolCall, error)
	GetToolCall(context.Context, string) (ToolCall, error)
	GetLatestPendingToolCall(context.Context, string) (ToolCall, error)
	CreateLLMCall(context.Context, LLMCall) (LLMCall, error)
}
