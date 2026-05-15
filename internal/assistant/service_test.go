package assistant

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/database"
	"github.com/kenichiLyon/loong64-b1-go/internal/runtimecfg"
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
	"github.com/kenichiLyon/loong64-b1-go/internal/upgrade"
)

func TestBootstrapAssistantCreatesFirstAdmin(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DBDriver:          "sqlite",
		SQLitePath:        filepath.Join(t.TempDir(), "assistant.db"),
		UpgradeDir:        "../../migrations",
		RuntimeConfigPath: filepath.Join(t.TempDir(), "runtime.json"),
		AutoMigrate:       true,
	}
	pool, err := database.Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer pool.Close()
	if _, err := upgrade.NewRunner(pool, cfg.UpgradeDir).Up(context.Background()); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	teachingService := teaching.NewService(teaching.NewSQLiteRepository(pool))
	service := NewService(NewSQLiteRepository(pool), teachingService, runtimecfg.New(cfg.RuntimeConfigPath), cfg, nil, pool, nil)

	conversation, err := service.CreateConversation(context.Background(), ScopeBootstrap, "")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	result, err := service.SendMessage(context.Background(), ScopeBootstrap, "", conversation.ID, SendMessageInput{Content: "请帮我创建首个管理员"})
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	if result.PendingToolCall == nil || result.PendingToolCall.ToolName != ToolBootstrapCreateAdmin {
		t.Fatalf("expected bootstrap tool call, got %+v", result.PendingToolCall)
	}
	confirm, err := service.ConfirmToolCall(context.Background(), ScopeBootstrap, "", result.PendingToolCall.ID, ConfirmToolCallInput{
		Inputs: mustJSON(map[string]any{
			"username":     "admin1",
			"display_name": "Admin One",
			"employee_no":  "A001",
			"password":     "test-pass",
		}),
	})
	if err != nil {
		t.Fatalf("confirm tool: %v", err)
	}
	if confirm.ToolCall.Status != ToolCallSucceeded {
		t.Fatalf("expected success, got %+v", confirm.ToolCall)
	}
	status, err := teachingService.GetBootstrapStatus(context.Background())
	if err != nil {
		t.Fatalf("bootstrap status: %v", err)
	}
	if !status.Initialized || status.UserCount != 1 {
		t.Fatalf("unexpected bootstrap status: %+v", status)
	}
}

func TestDeploymentAssistantSavesRuntimeConfig(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DBDriver:          "sqlite",
		SQLitePath:        filepath.Join(t.TempDir(), "assistant.db"),
		UpgradeDir:        "../../migrations",
		RuntimeConfigPath: filepath.Join(t.TempDir(), "runtime.json"),
		AutoMigrate:       true,
	}
	pool, err := database.Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer pool.Close()
	if _, err := upgrade.NewRunner(pool, cfg.UpgradeDir).Up(context.Background()); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	teachingService := teaching.NewService(teaching.NewSQLiteRepository(pool))
	service := NewService(NewSQLiteRepository(pool), teachingService, runtimecfg.New(cfg.RuntimeConfigPath), cfg, nil, pool, nil)
	if _, err := teachingService.BootstrapCreateAdmin(context.Background(), teaching.BootstrapCreateAdminInput{
		Username:    "admin1",
		DisplayName: "Admin One",
		EmployeeNo:  "A001",
		Password:    "test-pass",
	}, teaching.AuditEntry{}); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}

	conversation, err := service.CreateConversation(context.Background(), ScopeDeploymentAdmin, "admin-1")
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	result, err := service.SendMessage(context.Background(), ScopeDeploymentAdmin, "admin-1", conversation.ID, SendMessageInput{Content: "请把数据库切换到 postgres 并保存配置"})
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	if result.PendingToolCall == nil || result.PendingToolCall.ToolName != ToolSaveRuntimeConfig {
		t.Fatalf("expected save runtime config tool call, got %+v", result.PendingToolCall)
	}
	confirm, err := service.ConfirmToolCall(context.Background(), ScopeDeploymentAdmin, "admin-1", result.PendingToolCall.ID, ConfirmToolCallInput{
		Inputs: mustJSON(map[string]any{
			"db_driver":    "postgres",
			"database_url": "postgres://demo:secret@127.0.0.1:5432/demo?sslmode=disable",
			"auto_migrate": false,
		}),
	})
	if err != nil {
		t.Fatalf("confirm tool: %v", err)
	}
	if confirm.ToolCall.Status != ToolCallSucceeded {
		t.Fatalf("expected success, got %+v", confirm.ToolCall)
	}
	stored, exists, err := runtimecfg.New(cfg.RuntimeConfigPath).Load()
	if err != nil {
		t.Fatalf("load runtime config: %v", err)
	}
	if !exists || stored.DBDriver != "postgres" || stored.DatabaseURL == "" {
		t.Fatalf("unexpected runtime config: exists=%v cfg=%+v", exists, stored)
	}
	if string(confirm.ToolCall.RequestJSON) == "" || string(confirm.ToolCall.RequestJSON) == string(mustJSON(map[string]any{"database_url": "postgres://demo:secret@127.0.0.1:5432/demo?sslmode=disable"})) {
		t.Fatalf("expected redacted request json, got %s", string(confirm.ToolCall.RequestJSON))
	}
}
