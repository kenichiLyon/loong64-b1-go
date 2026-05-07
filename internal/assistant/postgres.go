package assistant

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kenichiLyon/loong64-b1-go/internal/database"
)

type PostgresRepository struct {
	db *database.Pool
}

func NewPostgresRepository(db *database.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

type pgxPool interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (r *PostgresRepository) raw() (pgxPool, error) {
	if r == nil || r.db == nil || r.db.Raw() == nil {
		return nil, unavailableError("assistant postgres repository is not configured", nil)
	}
	return r.db.Raw(), nil
}

func (r *PostgresRepository) CreateConversation(ctx context.Context, conversation Conversation) (Conversation, error) {
	pool, err := r.raw()
	if err != nil {
		return Conversation{}, err
	}
	if err := scanConversation(pool.QueryRow(ctx, `
INSERT INTO assistant_conversations (id, scope_type, actor_id, status, model, prompt_version, summary_text)
VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7)
RETURNING id, scope_type, COALESCE(actor_id, ''), status, model, prompt_version, summary_text, created_at, updated_at, last_message_at`,
		conversation.ID, conversation.ScopeType, conversation.ActorID, conversation.Status, conversation.Model, conversation.PromptVersion, conversation.SummaryText), &conversation); err != nil {
		return Conversation{}, mapDBError(err)
	}
	return conversation, nil
}

func (r *PostgresRepository) GetConversation(ctx context.Context, id string) (Conversation, error) {
	pool, err := r.raw()
	if err != nil {
		return Conversation{}, err
	}
	var conversation Conversation
	if err := scanConversation(pool.QueryRow(ctx, `
SELECT id, scope_type, COALESCE(actor_id, ''), status, model, prompt_version, summary_text, created_at, updated_at, last_message_at
FROM assistant_conversations WHERE id = $1`, id), &conversation); err != nil {
		return Conversation{}, mapDBError(err)
	}
	return conversation, nil
}

func (r *PostgresRepository) UpdateConversation(ctx context.Context, conversation Conversation) (Conversation, error) {
	pool, err := r.raw()
	if err != nil {
		return Conversation{}, err
	}
	if err := scanConversation(pool.QueryRow(ctx, `
UPDATE assistant_conversations
SET status = $2, model = $3, summary_text = $4, updated_at = now(), last_message_at = $5
WHERE id = $1
RETURNING id, scope_type, COALESCE(actor_id, ''), status, model, prompt_version, summary_text, created_at, updated_at, last_message_at`,
		conversation.ID, conversation.Status, conversation.Model, conversation.SummaryText, conversation.LastMessageAt), &conversation); err != nil {
		return Conversation{}, mapDBError(err)
	}
	return conversation, nil
}

func (r *PostgresRepository) CreateMessage(ctx context.Context, message Message) (Message, error) {
	pool, err := r.raw()
	if err != nil {
		return Message{}, err
	}
	if err := scanMessage(pool.QueryRow(ctx, `
INSERT INTO assistant_messages (id, conversation_id, role, content_text, context_snapshot_id, tool_call_id)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''))
RETURNING id, conversation_id, role, content_text, COALESCE(context_snapshot_id, ''), COALESCE(tool_call_id, ''), created_at`,
		message.ID, message.ConversationID, message.Role, message.ContentText, message.ContextSnapshotID, message.ToolCallID), &message); err != nil {
		return Message{}, mapDBError(err)
	}
	return message, nil
}

func (r *PostgresRepository) ListMessages(ctx context.Context, conversationID string, limit int) ([]Message, error) {
	pool, err := r.raw()
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
SELECT id, conversation_id, role, content_text, COALESCE(context_snapshot_id, ''), COALESCE(tool_call_id, ''), created_at
FROM assistant_messages
WHERE conversation_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2`, conversationID, limit)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()
	ordered := make([]Message, 0)
	for rows.Next() {
		var message Message
		if err := scanMessage(rows, &message); err != nil {
			return nil, mapDBError(err)
		}
		ordered = append(ordered, message)
	}
	for i, j := 0, len(ordered)-1; i < j; i, j = i+1, j-1 {
		ordered[i], ordered[j] = ordered[j], ordered[i]
	}
	return ordered, rows.Err()
}

func (r *PostgresRepository) CreateContextSnapshot(ctx context.Context, snapshot ContextSnapshot) (ContextSnapshot, error) {
	pool, err := r.raw()
	if err != nil {
		return ContextSnapshot{}, err
	}
	if err := scanSnapshot(pool.QueryRow(ctx, `
INSERT INTO assistant_context_snapshots (id, conversation_id, scope_stage, payload_json)
VALUES ($1, $2, $3, $4)
RETURNING id, conversation_id, scope_stage, payload_json, created_at`,
		snapshot.ID, snapshot.ConversationID, snapshot.ScopeStage, snapshot.PayloadJSON), &snapshot); err != nil {
		return ContextSnapshot{}, mapDBError(err)
	}
	return snapshot, nil
}

func (r *PostgresRepository) GetLatestContextSnapshot(ctx context.Context, conversationID string) (ContextSnapshot, error) {
	pool, err := r.raw()
	if err != nil {
		return ContextSnapshot{}, err
	}
	var snapshot ContextSnapshot
	if err := scanSnapshot(pool.QueryRow(ctx, `
SELECT id, conversation_id, scope_stage, payload_json, created_at
FROM assistant_context_snapshots
WHERE conversation_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1`, conversationID), &snapshot); err != nil {
		return ContextSnapshot{}, mapDBError(err)
	}
	return snapshot, nil
}

func (r *PostgresRepository) CreateToolCall(ctx context.Context, call ToolCall) (ToolCall, error) {
	pool, err := r.raw()
	if err != nil {
		return ToolCall{}, err
	}
	if err := scanToolCall(pool.QueryRow(ctx, `
INSERT INTO assistant_tool_calls (id, conversation_id, tool_name, status, request_json, response_json, error, confirmed_by_actor)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, conversation_id, tool_name, status, request_json, response_json, error, confirmed_by_actor, created_at, completed_at`,
		call.ID, call.ConversationID, call.ToolName, call.Status, defaultJSON(call.RequestJSON), defaultJSON(call.ResponseJSON), call.Error, call.ConfirmedByActor), &call); err != nil {
		return ToolCall{}, mapDBError(err)
	}
	return call, nil
}

func (r *PostgresRepository) UpdateToolCall(ctx context.Context, call ToolCall) (ToolCall, error) {
	pool, err := r.raw()
	if err != nil {
		return ToolCall{}, err
	}
	if err := scanToolCall(pool.QueryRow(ctx, `
UPDATE assistant_tool_calls
SET status = $2, request_json = $3, response_json = $4, error = $5, confirmed_by_actor = $6, completed_at = $7
WHERE id = $1
RETURNING id, conversation_id, tool_name, status, request_json, response_json, error, confirmed_by_actor, created_at, completed_at`,
		call.ID, call.Status, defaultJSON(call.RequestJSON), defaultJSON(call.ResponseJSON), call.Error, call.ConfirmedByActor, call.CompletedAt), &call); err != nil {
		return ToolCall{}, mapDBError(err)
	}
	return call, nil
}

func (r *PostgresRepository) GetToolCall(ctx context.Context, id string) (ToolCall, error) {
	pool, err := r.raw()
	if err != nil {
		return ToolCall{}, err
	}
	var call ToolCall
	if err := scanToolCall(pool.QueryRow(ctx, `
SELECT id, conversation_id, tool_name, status, request_json, response_json, error, confirmed_by_actor, created_at, completed_at
FROM assistant_tool_calls WHERE id = $1`, id), &call); err != nil {
		return ToolCall{}, mapDBError(err)
	}
	return call, nil
}

func (r *PostgresRepository) GetLatestPendingToolCall(ctx context.Context, conversationID string) (ToolCall, error) {
	pool, err := r.raw()
	if err != nil {
		return ToolCall{}, err
	}
	var call ToolCall
	if err := scanToolCall(pool.QueryRow(ctx, `
SELECT id, conversation_id, tool_name, status, request_json, response_json, error, confirmed_by_actor, created_at, completed_at
FROM assistant_tool_calls
WHERE conversation_id = $1 AND status = 'pending_confirmation'
ORDER BY created_at DESC, id DESC
LIMIT 1`, conversationID), &call); err != nil {
		return ToolCall{}, mapDBError(err)
	}
	return call, nil
}

func (r *PostgresRepository) CreateLLMCall(ctx context.Context, call LLMCall) (LLMCall, error) {
	pool, err := r.raw()
	if err != nil {
		return LLMCall{}, err
	}
	if err := scanLLMCall(pool.QueryRow(ctx, `
INSERT INTO assistant_llm_calls (id, conversation_id, provider, model, prompt_version, input_hash, output, status, error, latency_ms, prompt_tokens, completion_tokens)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, conversation_id, provider, model, prompt_version, input_hash, output, status, error, latency_ms, prompt_tokens, completion_tokens, created_at`,
		call.ID, call.ConversationID, call.Provider, call.Model, call.PromptVersion, call.InputHash, defaultJSON(call.Output), call.Status, call.Error, call.LatencyMS, call.PromptTokens, call.CompletionTokens), &call); err != nil {
		return LLMCall{}, mapDBError(err)
	}
	return call, nil
}

func scanConversation(row pgx.Row, conversation *Conversation) error {
	var scope, status string
	if err := row.Scan(&conversation.ID, &scope, &conversation.ActorID, &status, &conversation.Model, &conversation.PromptVersion, &conversation.SummaryText, &conversation.CreatedAt, &conversation.UpdatedAt, &conversation.LastMessageAt); err != nil {
		return err
	}
	conversation.ScopeType = ScopeType(scope)
	conversation.Status = ConversationStatus(status)
	return nil
}

func scanMessage(row pgx.Row, message *Message) error {
	var role string
	if err := row.Scan(&message.ID, &message.ConversationID, &role, &message.ContentText, &message.ContextSnapshotID, &message.ToolCallID, &message.CreatedAt); err != nil {
		return err
	}
	message.Role = MessageRole(role)
	return nil
}

func scanSnapshot(row pgx.Row, snapshot *ContextSnapshot) error {
	var stage string
	if err := row.Scan(&snapshot.ID, &snapshot.ConversationID, &stage, &snapshot.PayloadJSON, &snapshot.CreatedAt); err != nil {
		return err
	}
	snapshot.ScopeStage = ContextStage(stage)
	return nil
}

func scanToolCall(row pgx.Row, call *ToolCall) error {
	var tool, status string
	var completedAt pgtype.Timestamptz
	if err := row.Scan(&call.ID, &call.ConversationID, &tool, &status, &call.RequestJSON, &call.ResponseJSON, &call.Error, &call.ConfirmedByActor, &call.CreatedAt, &completedAt); err != nil {
		return err
	}
	call.ToolName = ToolName(tool)
	call.Status = ToolCallStatus(status)
	if completedAt.Valid {
		v := completedAt.Time
		call.CompletedAt = &v
	} else {
		call.CompletedAt = nil
	}
	return nil
}

func scanLLMCall(row pgx.Row, call *LLMCall) error {
	return row.Scan(&call.ID, &call.ConversationID, &call.Provider, &call.Model, &call.PromptVersion, &call.InputHash, &call.Output, &call.Status, &call.Error, &call.LatencyMS, &call.PromptTokens, &call.CompletionTokens, &call.CreatedAt)
}

func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return notFoundError("assistant resource not found")
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return conflictError("assistant resource already exists")
		case "23503":
			return validationError("referenced resource does not exist")
		}
	}
	return fmt.Errorf("assistant postgres repository: %w", err)
}
