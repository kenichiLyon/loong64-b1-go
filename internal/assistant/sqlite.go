package assistant

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kenichiLyon/loong64-b1-go/internal/database"
)

type SQLiteRepository struct {
	db *database.Pool
}

func NewSQLiteRepository(db *database.Pool) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) sqlDB() (*sql.DB, error) {
	if r == nil || r.db == nil || r.db.SQLDB() == nil {
		return nil, unavailableError("assistant sqlite repository is not configured", nil)
	}
	return r.db.SQLDB(), nil
}

func (r *SQLiteRepository) CreateConversation(ctx context.Context, conversation Conversation) (Conversation, error) {
	db, err := r.sqlDB()
	if err != nil {
		return Conversation{}, err
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO assistant_conversations (id, scope_type, actor_id, status, model, prompt_version, summary_text)
VALUES (?, ?, NULLIF(?, ''), ?, ?, ?, ?)`,
		conversation.ID, conversation.ScopeType, conversation.ActorID, conversation.Status, conversation.Model, conversation.PromptVersion, conversation.SummaryText); err != nil {
		return Conversation{}, sqliteMapError(err)
	}
	return r.GetConversation(ctx, conversation.ID)
}

func (r *SQLiteRepository) GetConversation(ctx context.Context, id string) (Conversation, error) {
	db, err := r.sqlDB()
	if err != nil {
		return Conversation{}, err
	}
	var conversation Conversation
	var scope, status string
	if err := db.QueryRowContext(ctx, `
SELECT id, scope_type, COALESCE(actor_id, ''), status, model, prompt_version, summary_text, created_at, updated_at, last_message_at
FROM assistant_conversations WHERE id = ?`, id).Scan(&conversation.ID, &scope, &conversation.ActorID, &status, &conversation.Model, &conversation.PromptVersion, &conversation.SummaryText, &conversation.CreatedAt, &conversation.UpdatedAt, &conversation.LastMessageAt); err != nil {
		return Conversation{}, sqliteMapError(err)
	}
	conversation.ScopeType = ScopeType(scope)
	conversation.Status = ConversationStatus(status)
	return conversation, nil
}

func (r *SQLiteRepository) UpdateConversation(ctx context.Context, conversation Conversation) (Conversation, error) {
	db, err := r.sqlDB()
	if err != nil {
		return Conversation{}, err
	}
	if _, err := db.ExecContext(ctx, `
UPDATE assistant_conversations
SET status = ?, model = ?, summary_text = ?, updated_at = CURRENT_TIMESTAMP, last_message_at = ?
WHERE id = ?`,
		conversation.Status, conversation.Model, conversation.SummaryText, conversation.LastMessageAt, conversation.ID); err != nil {
		return Conversation{}, sqliteMapError(err)
	}
	return r.GetConversation(ctx, conversation.ID)
}

func (r *SQLiteRepository) CreateMessage(ctx context.Context, message Message) (Message, error) {
	db, err := r.sqlDB()
	if err != nil {
		return Message{}, err
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO assistant_messages (id, conversation_id, role, content_text, context_snapshot_id, tool_call_id)
VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''))`,
		message.ID, message.ConversationID, message.Role, message.ContentText, message.ContextSnapshotID, message.ToolCallID); err != nil {
		return Message{}, sqliteMapError(err)
	}
	return r.getMessage(ctx, message.ID)
}

func (r *SQLiteRepository) getMessage(ctx context.Context, id string) (Message, error) {
	db, err := r.sqlDB()
	if err != nil {
		return Message{}, err
	}
	var message Message
	var role string
	if err := db.QueryRowContext(ctx, `
SELECT id, conversation_id, role, content_text, COALESCE(context_snapshot_id, ''), COALESCE(tool_call_id, ''), created_at
FROM assistant_messages WHERE id = ?`, id).Scan(&message.ID, &message.ConversationID, &role, &message.ContentText, &message.ContextSnapshotID, &message.ToolCallID, &message.CreatedAt); err != nil {
		return Message{}, sqliteMapError(err)
	}
	message.Role = MessageRole(role)
	return message, nil
}

func (r *SQLiteRepository) ListMessages(ctx context.Context, conversationID string, limit int) ([]Message, error) {
	db, err := r.sqlDB()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, conversation_id, role, content_text, COALESCE(context_snapshot_id, ''), COALESCE(tool_call_id, ''), created_at
FROM assistant_messages
WHERE conversation_id = ?
ORDER BY created_at DESC, id DESC
LIMIT ?`, conversationID, limit)
	if err != nil {
		return nil, sqliteMapError(err)
	}
	defer func() { _ = rows.Close() }()
	result := make([]Message, 0)
	for rows.Next() {
		var message Message
		var role string
		if err := rows.Scan(&message.ID, &message.ConversationID, &role, &message.ContentText, &message.ContextSnapshotID, &message.ToolCallID, &message.CreatedAt); err != nil {
			return nil, sqliteMapError(err)
		}
		message.Role = MessageRole(role)
		result = append(result, message)
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result, rows.Err()
}

func (r *SQLiteRepository) CreateContextSnapshot(ctx context.Context, snapshot ContextSnapshot) (ContextSnapshot, error) {
	db, err := r.sqlDB()
	if err != nil {
		return ContextSnapshot{}, err
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO assistant_context_snapshots (id, conversation_id, scope_stage, payload_json)
VALUES (?, ?, ?, ?)`, snapshot.ID, snapshot.ConversationID, snapshot.ScopeStage, string(defaultJSON(snapshot.PayloadJSON))); err != nil {
		return ContextSnapshot{}, sqliteMapError(err)
	}
	return r.getSnapshot(ctx, snapshot.ID)
}

func (r *SQLiteRepository) getSnapshot(ctx context.Context, id string) (ContextSnapshot, error) {
	db, err := r.sqlDB()
	if err != nil {
		return ContextSnapshot{}, err
	}
	var snapshot ContextSnapshot
	var stage string
	var payload string
	if err := db.QueryRowContext(ctx, `
SELECT id, conversation_id, scope_stage, payload_json, created_at
FROM assistant_context_snapshots WHERE id = ?`, id).Scan(&snapshot.ID, &snapshot.ConversationID, &stage, &payload, &snapshot.CreatedAt); err != nil {
		return ContextSnapshot{}, sqliteMapError(err)
	}
	snapshot.ScopeStage = ContextStage(stage)
	snapshot.PayloadJSON = json.RawMessage(payload)
	return snapshot, nil
}

func (r *SQLiteRepository) GetLatestContextSnapshot(ctx context.Context, conversationID string) (ContextSnapshot, error) {
	db, err := r.sqlDB()
	if err != nil {
		return ContextSnapshot{}, err
	}
	var snapshot ContextSnapshot
	var stage string
	var payload string
	if err := db.QueryRowContext(ctx, `
SELECT id, conversation_id, scope_stage, payload_json, created_at
FROM assistant_context_snapshots
WHERE conversation_id = ?
ORDER BY created_at DESC, id DESC
LIMIT 1`, conversationID).Scan(&snapshot.ID, &snapshot.ConversationID, &stage, &payload, &snapshot.CreatedAt); err != nil {
		return ContextSnapshot{}, sqliteMapError(err)
	}
	snapshot.ScopeStage = ContextStage(stage)
	snapshot.PayloadJSON = json.RawMessage(payload)
	return snapshot, nil
}

func (r *SQLiteRepository) CreateToolCall(ctx context.Context, call ToolCall) (ToolCall, error) {
	db, err := r.sqlDB()
	if err != nil {
		return ToolCall{}, err
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO assistant_tool_calls (id, conversation_id, tool_name, status, request_json, response_json, error, confirmed_by_actor)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		call.ID, call.ConversationID, call.ToolName, call.Status, string(defaultJSON(call.RequestJSON)), string(defaultJSON(call.ResponseJSON)), call.Error, call.ConfirmedByActor); err != nil {
		return ToolCall{}, sqliteMapError(err)
	}
	return r.GetToolCall(ctx, call.ID)
}

func (r *SQLiteRepository) UpdateToolCall(ctx context.Context, call ToolCall) (ToolCall, error) {
	db, err := r.sqlDB()
	if err != nil {
		return ToolCall{}, err
	}
	if _, err := db.ExecContext(ctx, `
UPDATE assistant_tool_calls
SET status = ?, request_json = ?, response_json = ?, error = ?, confirmed_by_actor = ?, completed_at = ?
WHERE id = ?`,
		call.Status, string(defaultJSON(call.RequestJSON)), string(defaultJSON(call.ResponseJSON)), call.Error, call.ConfirmedByActor, call.CompletedAt, call.ID); err != nil {
		return ToolCall{}, sqliteMapError(err)
	}
	return r.GetToolCall(ctx, call.ID)
}

func (r *SQLiteRepository) GetToolCall(ctx context.Context, id string) (ToolCall, error) {
	db, err := r.sqlDB()
	if err != nil {
		return ToolCall{}, err
	}
	var call ToolCall
	var tool, status string
	var requestJSON, responseJSON string
	var completedAt sql.NullTime
	if err := db.QueryRowContext(ctx, `
SELECT id, conversation_id, tool_name, status, request_json, response_json, error, confirmed_by_actor, created_at, completed_at
FROM assistant_tool_calls WHERE id = ?`, id).Scan(&call.ID, &call.ConversationID, &tool, &status, &requestJSON, &responseJSON, &call.Error, &call.ConfirmedByActor, &call.CreatedAt, &completedAt); err != nil {
		return ToolCall{}, sqliteMapError(err)
	}
	call.ToolName = ToolName(tool)
	call.Status = ToolCallStatus(status)
	call.RequestJSON = json.RawMessage(requestJSON)
	call.ResponseJSON = json.RawMessage(responseJSON)
	if completedAt.Valid {
		v := completedAt.Time
		call.CompletedAt = &v
	}
	return call, nil
}

func (r *SQLiteRepository) GetLatestPendingToolCall(ctx context.Context, conversationID string) (ToolCall, error) {
	db, err := r.sqlDB()
	if err != nil {
		return ToolCall{}, err
	}
	var call ToolCall
	var tool, status string
	var requestJSON, responseJSON string
	var completedAt sql.NullTime
	if err := db.QueryRowContext(ctx, `
SELECT id, conversation_id, tool_name, status, request_json, response_json, error, confirmed_by_actor, created_at, completed_at
FROM assistant_tool_calls
WHERE conversation_id = ? AND status = 'pending_confirmation'
ORDER BY created_at DESC, id DESC
LIMIT 1`, conversationID).Scan(&call.ID, &call.ConversationID, &tool, &status, &requestJSON, &responseJSON, &call.Error, &call.ConfirmedByActor, &call.CreatedAt, &completedAt); err != nil {
		return ToolCall{}, sqliteMapError(err)
	}
	call.ToolName = ToolName(tool)
	call.Status = ToolCallStatus(status)
	call.RequestJSON = json.RawMessage(requestJSON)
	call.ResponseJSON = json.RawMessage(responseJSON)
	if completedAt.Valid {
		v := completedAt.Time
		call.CompletedAt = &v
	}
	return call, nil
}

func (r *SQLiteRepository) CreateLLMCall(ctx context.Context, call LLMCall) (LLMCall, error) {
	db, err := r.sqlDB()
	if err != nil {
		return LLMCall{}, err
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO assistant_llm_calls (id, conversation_id, provider, model, prompt_version, input_hash, output, status, error, latency_ms, prompt_tokens, completion_tokens)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		call.ID, call.ConversationID, call.Provider, call.Model, call.PromptVersion, call.InputHash, string(defaultJSON(call.Output)), call.Status, call.Error, call.LatencyMS, call.PromptTokens, call.CompletionTokens); err != nil {
		return LLMCall{}, sqliteMapError(err)
	}
	var created LLMCall
	var output string
	if err := db.QueryRowContext(ctx, `
SELECT id, conversation_id, provider, model, prompt_version, input_hash, output, status, error, latency_ms, prompt_tokens, completion_tokens, created_at
FROM assistant_llm_calls WHERE id = ?`, call.ID).Scan(&created.ID, &created.ConversationID, &created.Provider, &created.Model, &created.PromptVersion, &created.InputHash, &output, &created.Status, &created.Error, &created.LatencyMS, &created.PromptTokens, &created.CompletionTokens, &created.CreatedAt); err != nil {
		return LLMCall{}, sqliteMapError(err)
	}
	created.Output = json.RawMessage(output)
	return created, nil
}

func sqliteMapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return notFoundError("assistant resource not found")
	}
	text := strings.ToLower(err.Error())
	switch {
	case strings.Contains(text, "unique constraint failed"):
		return conflictError("assistant resource already exists")
	case strings.Contains(text, "foreign key constraint failed"):
		return validationError("referenced resource does not exist")
	}
	return fmt.Errorf("assistant sqlite repository: %w", err)
}
