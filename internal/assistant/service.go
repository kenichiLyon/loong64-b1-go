package assistant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/database"
	"github.com/kenichiLyon/loong64-b1-go/internal/llm"
	"github.com/kenichiLyon/loong64-b1-go/internal/runtimecfg"
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
)

type LLMCompleter interface {
	CompleteJSON(context.Context, llm.CompletionRequest) (llm.CompletionResponse, error)
}

type Service struct {
	repo      Repository
	teaching  *teaching.Service
	runtime   *runtimecfg.Manager
	config    config.Config
	logger    *slog.Logger
	llmClient LLMCompleter
	db        *database.Pool
}

func NewService(repo Repository, teachingService *teaching.Service, runtimeManager *runtimecfg.Manager, cfg config.Config, logger *slog.Logger, db *database.Pool, llmClient LLMCompleter) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	if runtimeManager == nil {
		runtimeManager = runtimecfg.New(cfg.RuntimeConfigPath)
	}
	return &Service{repo: repo, teaching: teachingService, runtime: runtimeManager, config: cfg, logger: logger, llmClient: llmClient, db: db}
}

func (s *Service) CreateConversation(ctx context.Context, scope ScopeType, actorID string) (Conversation, error) {
	if err := s.ready(); err != nil {
		return Conversation{}, err
	}
	scope = normalizeScope(scope)
	if scope == "" {
		return Conversation{}, validationError("invalid assistant scope")
	}
	conversation := Conversation{
		ID:            teaching.NewID("asc"),
		ScopeType:     scope,
		ActorID:       strings.TrimSpace(actorID),
		Status:        ConversationActive,
		PromptVersion: PromptVersion,
		Model:         s.config.LLMModel,
	}
	return s.repo.CreateConversation(ctx, conversation)
}

func (s *Service) GetConversationDetail(ctx context.Context, scope ScopeType, actorID, conversationID string) (ConversationDetail, error) {
	if err := s.ready(); err != nil {
		return ConversationDetail{}, err
	}
	conversation, err := s.repo.GetConversation(ctx, strings.TrimSpace(conversationID))
	if err != nil {
		return ConversationDetail{}, err
	}
	if err := ensureConversationAccess(scope, actorID, conversation); err != nil {
		return ConversationDetail{}, err
	}
	messages, err := s.repo.ListMessages(ctx, conversation.ID, 50)
	if err != nil {
		return ConversationDetail{}, err
	}
	var latestSnapshot *ContextSnapshot
	if snapshot, err := s.repo.GetLatestContextSnapshot(ctx, conversation.ID); err == nil {
		latestSnapshot = &snapshot
	}
	var pending *ToolCall
	if toolCall, err := s.repo.GetLatestPendingToolCall(ctx, conversation.ID); err == nil {
		pending = &toolCall
	}
	return ConversationDetail{Conversation: conversation, Messages: messages, PendingToolCall: pending, LatestSnapshot: latestSnapshot}, nil
}

func (s *Service) SendMessage(ctx context.Context, scope ScopeType, actorID, conversationID string, input SendMessageInput) (SendMessageResult, error) {
	if err := s.ready(); err != nil {
		return SendMessageResult{}, err
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return SendMessageResult{}, validationError("message content is required")
	}
	scope = normalizeScope(scope)
	if scope == "" {
		return SendMessageResult{}, validationError("invalid assistant scope")
	}
	conversation, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil {
		return SendMessageResult{}, err
	}
	if err := ensureConversationAccess(scope, actorID, conversation); err != nil {
		return SendMessageResult{}, err
	}
	stage, snapshotPayload, err := s.buildContextSnapshot(ctx, scope)
	if err != nil {
		return SendMessageResult{}, err
	}
	snapshot, err := s.repo.CreateContextSnapshot(ctx, ContextSnapshot{
		ID:             teaching.NewID("acs"),
		ConversationID: conversation.ID,
		ScopeStage:     stage,
		PayloadJSON:    redactJSON(snapshotPayload),
	})
	if err != nil {
		return SendMessageResult{}, err
	}
	if _, err := s.repo.CreateMessage(ctx, Message{
		ID:                teaching.NewID("asm"),
		ConversationID:    conversation.ID,
		Role:              RoleUser,
		ContentText:       redactText(content),
		ContextSnapshotID: snapshot.ID,
	}); err != nil {
		return SendMessageResult{}, err
	}
	history, err := s.repo.ListMessages(ctx, conversation.ID, 6)
	if err != nil {
		return SendMessageResult{}, err
	}
	responseText, proposedTool, llmCall, err := s.planReply(ctx, conversation, history, snapshot.PayloadJSON, content)
	if llmCall.ID != "" {
		_, _ = s.repo.CreateLLMCall(ctx, llmCall)
	}
	if err != nil {
		return SendMessageResult{}, err
	}
	var pending *ToolCall
	var toolCallID string
	requiresConfirmation := false
	if proposedTool != "" {
		call, callErr := s.repo.CreateToolCall(ctx, ToolCall{
			ID:             teaching.NewID("atc"),
			ConversationID: conversation.ID,
			ToolName:       proposedTool,
			Status:         ToolCallPendingConfirmation,
			RequestJSON:    redactJSON(mustJSON(map[string]any{"tool_name": proposedTool, "inputs_mode": "provide_on_confirm"})),
			ResponseJSON:   mustJSON(map[string]any{}),
		})
		if callErr != nil {
			return SendMessageResult{}, callErr
		}
		pending = &call
		toolCallID = call.ID
		requiresConfirmation = true
	}
	assistantMessage, err := s.repo.CreateMessage(ctx, Message{
		ID:                teaching.NewID("asm"),
		ConversationID:    conversation.ID,
		Role:              RoleAssistant,
		ContentText:       responseText,
		ContextSnapshotID: snapshot.ID,
		ToolCallID:        toolCallID,
	})
	if err != nil {
		return SendMessageResult{}, err
	}
	conversation.SummaryText = updateSummary(conversation.SummaryText, content, responseText)
	conversation.LastMessageAt = time.Now().UTC()
	conversation.UpdatedAt = conversation.LastMessageAt
	updatedConversation, err := s.repo.UpdateConversation(ctx, conversation)
	if err != nil {
		return SendMessageResult{}, err
	}
	return SendMessageResult{
		Conversation:         updatedConversation,
		AssistantMessage:     assistantMessage,
		PendingToolCall:      pending,
		ContextSnapshot:      snapshot,
		RequiresConfirmation: requiresConfirmation,
	}, nil
}

func (s *Service) ConfirmToolCall(ctx context.Context, scope ScopeType, actorID, toolCallID string, input ConfirmToolCallInput) (ConfirmToolCallResult, error) {
	if err := s.ready(); err != nil {
		return ConfirmToolCallResult{}, err
	}
	call, err := s.repo.GetToolCall(ctx, strings.TrimSpace(toolCallID))
	if err != nil {
		return ConfirmToolCallResult{}, err
	}
	conversation, err := s.repo.GetConversation(ctx, call.ConversationID)
	if err != nil {
		return ConfirmToolCallResult{}, err
	}
	if err := ensureConversationAccess(scope, actorID, conversation); err != nil {
		return ConfirmToolCallResult{}, err
	}
	if call.Status != ToolCallPendingConfirmation {
		return ConfirmToolCallResult{}, conflictError("tool call is not pending confirmation")
	}
	call.Status = ToolCallRunning
	call.ConfirmedByActor = strings.TrimSpace(actorID)
	call, err = s.repo.UpdateToolCall(ctx, call)
	if err != nil {
		return ConfirmToolCallResult{}, err
	}
	resultSummary, responseJSON, execErr := s.runTool(ctx, conversation.ScopeType, call.ToolName, input.Inputs)
	if execErr != nil {
		call.Status = ToolCallFailed
		call.Error = redactText(execErr.Error())
		call.ResponseJSON = mustJSON(map[string]any{})
	} else {
		call.Status = ToolCallSucceeded
		call.Error = ""
		call.ResponseJSON = redactJSON(responseJSON)
	}
	now := time.Now().UTC()
	call.CompletedAt = &now
	call, err = s.repo.UpdateToolCall(ctx, call)
	if err != nil {
		return ConfirmToolCallResult{}, err
	}
	toolMessage, err := s.repo.CreateMessage(ctx, Message{
		ID:             teaching.NewID("asm"),
		ConversationID: conversation.ID,
		Role:           RoleTool,
		ContentText:    resultSummary,
		ToolCallID:     call.ID,
	})
	if err != nil {
		return ConfirmToolCallResult{}, err
	}
	assistantText := buildToolFollowUp(call.ToolName, call.Status, resultSummary)
	assistantMessage, err := s.repo.CreateMessage(ctx, Message{
		ID:             teaching.NewID("asm"),
		ConversationID: conversation.ID,
		Role:           RoleAssistant,
		ContentText:    assistantText,
		ToolCallID:     call.ID,
	})
	if err != nil {
		return ConfirmToolCallResult{}, err
	}
	conversation.SummaryText = updateSummary(conversation.SummaryText, "tool:"+string(call.ToolName), assistantText)
	conversation.LastMessageAt = now
	conversation.UpdatedAt = now
	conversation, err = s.repo.UpdateConversation(ctx, conversation)
	if err != nil {
		return ConfirmToolCallResult{}, err
	}
	return ConfirmToolCallResult{Conversation: conversation, ToolCall: call, ToolMessage: toolMessage, AssistantMessage: assistantMessage}, nil
}

func (s *Service) ready() error {
	if s == nil || s.repo == nil || s.teaching == nil {
		return unavailableError("assistant service is not configured", nil)
	}
	return nil
}

func normalizeScope(scope ScopeType) ScopeType {
	switch scope {
	case ScopeBootstrap, ScopeDeploymentAdmin:
		return scope
	default:
		return ""
	}
}

func ensureConversationAccess(scope ScopeType, actorID string, conversation Conversation) error {
	if scope != conversation.ScopeType {
		return forbiddenError("conversation scope does not match request scope")
	}
	if scope == ScopeDeploymentAdmin && strings.TrimSpace(conversation.ActorID) != strings.TrimSpace(actorID) {
		return forbiddenError("admin can only access own deployment assistant conversations")
	}
	return nil
}

func updateSummary(existing, userText, assistantText string) string {
	parts := []string{}
	if strings.TrimSpace(existing) != "" {
		parts = append(parts, existing)
	}
	parts = append(parts, "USER: "+strings.TrimSpace(redactText(userText)))
	parts = append(parts, "ASSISTANT: "+strings.TrimSpace(redactText(assistantText)))
	summary := strings.Join(parts, "\n")
	if len(summary) > 2400 {
		summary = summary[len(summary)-2400:]
	}
	return summary
}

func (s *Service) planReply(ctx context.Context, conversation Conversation, history []Message, snapshot json.RawMessage, rawContent string) (string, ToolName, LLMCall, error) {
	if s.llmClient == nil {
		return s.fallbackReply(conversation.ScopeType, snapshot, rawContent), s.fallbackTool(conversation.ScopeType, rawContent), LLMCall{}, nil
	}
	request := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: deploymentAssistantSystemPrompt()},
			{Role: "user", Content: string(buildAssistantPrompt(conversation, history, snapshot, rawContent))},
		},
		Temperature: 0.1,
		MaxTokens:   900,
	}
	inputHash := sha256Hex(buildAssistantPrompt(conversation, history, snapshot, rawContent))
	log := LLMCall{
		ID:             teaching.NewID("alc"),
		ConversationID: conversation.ID,
		Provider:       "openai-compatible",
		PromptVersion:  PromptVersion,
		InputHash:      inputHash,
		Output:         mustJSON(map[string]any{}),
		Status:         "skipped",
	}
	response, err := s.llmClient.CompleteJSON(ctx, request)
	if response.Model != "" {
		log.Model = response.Model
	}
	log.LatencyMS = int(response.Latency.Milliseconds())
	log.PromptTokens = response.PromptTokens
	log.CompletionTokens = response.CompletionTokens
	if err != nil {
		log.Status = "failed"
		log.Error = redactText(err.Error())
		return s.fallbackReply(conversation.ScopeType, snapshot, rawContent), s.fallbackTool(conversation.ScopeType, rawContent), log, nil
	}
	log.Status = "succeeded"
	if json.Valid([]byte(response.Content)) {
		log.Output = json.RawMessage(response.Content)
	}
	var decoded struct {
		ResponseText         string `json:"response_text"`
		ProposedToolName     string `json:"proposed_tool_name"`
		RequiresConfirmation bool   `json:"requires_confirmation"`
	}
	if err := json.Unmarshal([]byte(response.Content), &decoded); err != nil {
		log.Status = "failed"
		log.Error = redactText(err.Error())
		return s.fallbackReply(conversation.ScopeType, snapshot, rawContent), s.fallbackTool(conversation.ScopeType, rawContent), log, nil
	}
	tool := ToolName(strings.TrimSpace(decoded.ProposedToolName))
	if tool != "" && !isAllowedTool(conversation.ScopeType, tool) {
		tool = ""
	}
	text := strings.TrimSpace(decoded.ResponseText)
	if text == "" {
		text = s.fallbackReply(conversation.ScopeType, snapshot, rawContent)
	}
	return text, tool, log, nil
}

func buildAssistantPrompt(conversation Conversation, history []Message, snapshot json.RawMessage, rawContent string) json.RawMessage {
	items := make([]map[string]string, 0, len(history))
	for _, msg := range history {
		items = append(items, map[string]string{"role": string(msg.Role), "content": msg.ContentText})
	}
	return mustJSON(map[string]any{
		"prompt_version":  PromptVersion,
		"scope":           conversation.ScopeType,
		"summary_text":    conversation.SummaryText,
		"recent_messages": items,
		"latest_context":  json.RawMessage(snapshot),
		"user_message":    redactText(rawContent),
		"allowed_tools":   allowedTools(conversation.ScopeType),
		"output_schema": map[string]string{
			"response_text":         "string",
			"proposed_tool_name":    "allowed tool name or empty string",
			"requires_confirmation": "boolean",
		},
	})
}

func deploymentAssistantSystemPrompt() string {
	return "You are a deployment assistant for a software training evaluation system. Never reveal secrets, never ask to hot-restart the service, and never execute tools directly. Only propose one allowed tool at a time when user intent is clear. If configuration secrets may be required, tell the user to provide them at confirmation time through the UI, not in the stored conversation."
}

func sha256Hex(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
