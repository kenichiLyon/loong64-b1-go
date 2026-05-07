package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/database"
	"github.com/kenichiLyon/loong64-b1-go/internal/runtimecfg"
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
)

func allowedTools(scope ScopeType) []ToolName {
	switch scope {
	case ScopeBootstrap:
		return []ToolName{ToolInspectBootstrapStatus, ToolInspectRuntimeConfig, ToolBootstrapCreateAdmin}
	case ScopeDeploymentAdmin:
		return []ToolName{ToolInspectBootstrapStatus, ToolInspectRuntimeConfig, ToolTestSQLitePath, ToolTestPostgresConnection, ToolSaveRuntimeConfig}
	default:
		return nil
	}
}

func isAllowedTool(scope ScopeType, tool ToolName) bool {
	for _, candidate := range allowedTools(scope) {
		if candidate == tool {
			return true
		}
	}
	return false
}

func (s *Service) fallbackTool(scope ScopeType, content string) ToolName {
	text := strings.ToLower(strings.TrimSpace(content))
	switch scope {
	case ScopeBootstrap:
		if strings.Contains(text, "管理员") || strings.Contains(text, "admin") || strings.Contains(text, "初始化") {
			return ToolBootstrapCreateAdmin
		}
		if strings.Contains(text, "配置") || strings.Contains(text, "数据库") {
			return ToolInspectRuntimeConfig
		}
		if strings.Contains(text, "状态") || strings.Contains(text, "为什么") {
			return ToolInspectBootstrapStatus
		}
	case ScopeDeploymentAdmin:
		if strings.Contains(text, "postgres") && strings.Contains(text, "测试") {
			return ToolTestPostgresConnection
		}
		if strings.Contains(text, "sqlite") && strings.Contains(text, "测试") {
			return ToolTestSQLitePath
		}
		if strings.Contains(text, "保存") || strings.Contains(text, "切换") || strings.Contains(text, "改成") {
			return ToolSaveRuntimeConfig
		}
		if strings.Contains(text, "配置") || strings.Contains(text, "状态") {
			return ToolInspectRuntimeConfig
		}
	}
	return ""
}

func (s *Service) fallbackReply(scope ScopeType, snapshot json.RawMessage, content string) string {
	switch s.fallbackTool(scope, content) {
	case ToolBootstrapCreateAdmin:
		return "当前系统还没有任何用户。请先在 bootstrap 面板填写首个管理员信息，然后确认创建管理员工具调用。"
	case ToolInspectRuntimeConfig:
		return "我可以先读取当前运行配置和已保存的 runtime.json，帮你判断现在是 SQLite 还是 PostgreSQL，以及是否需要重启。"
	case ToolInspectBootstrapStatus:
		return "我可以先读取当前 bootstrap 状态，确认系统是否已初始化、当前活动数据库是什么。"
	case ToolTestPostgresConnection:
		return "我可以测试 PostgreSQL 连接。请先在运行配置面板填写 PostgreSQL URL，再确认执行。"
	case ToolTestSQLitePath:
		return "我可以测试 SQLite 路径是否可用。请先在运行配置面板填写 SQLite 路径，再确认执行。"
	case ToolSaveRuntimeConfig:
		return "我可以把当前运行配置面板中的数据库设置保存到 runtime.json，但保存后仍然需要你手动重启服务。"
	default:
		return "我可以帮助你检查 bootstrap 状态、运行配置、测试数据库连接，或保存 runtime.json。请直接描述你要初始化或切换数据库的目标。"
	}
}

func (s *Service) buildContextSnapshot(ctx context.Context, scope ScopeType) (ContextStage, json.RawMessage, error) {
	status, err := s.teaching.GetBootstrapStatus(ctx)
	if err != nil {
		return "", nil, err
	}
	storedCfg, exists, err := s.runtime.Load()
	if err != nil {
		return "", nil, unavailableError("runtime configuration is unavailable", err)
	}
	activeView := runtimecfg.ToView(s.runtime.Path(), runtimecfg.FileConfig{
		DBDriver:    s.config.DBDriver,
		SQLitePath:  s.config.SQLitePath,
		DatabaseURL: redactRuntimeURL(s.config.DatabaseURL),
		AutoMigrate: boolPtr(s.config.AutoMigrate),
	}, false)
	var storedView *runtimecfg.View
	if exists {
		view := runtimecfg.ToView(s.runtime.Path(), runtimecfg.FileConfig{
			DBDriver:    storedCfg.DBDriver,
			SQLitePath:  storedCfg.SQLitePath,
			DatabaseURL: redactRuntimeURL(storedCfg.DatabaseURL),
			AutoMigrate: storedCfg.AutoMigrate,
		}, false)
		storedView = &view
	}
	stage := ContextStageRuntimeConfig
	if !status.Initialized {
		stage = ContextStageBootstrapStatus
	}
	return stage, mustJSON(map[string]any{
		"bootstrap": map[string]any{
			"initialized": status.Initialized,
			"user_count":  status.UserCount,
		},
		"runtime":        activeView,
		"stored_runtime": storedView,
		"allowed_tools":  allowedTools(scope),
	}), nil
}

func (s *Service) runTool(ctx context.Context, scope ScopeType, tool ToolName, rawInputs json.RawMessage) (string, json.RawMessage, error) {
	if !isAllowedTool(scope, tool) {
		return "", nil, forbiddenError("tool is not allowed for this assistant scope")
	}
	switch tool {
	case ToolInspectBootstrapStatus:
		status, err := s.teaching.GetBootstrapStatus(ctx)
		if err != nil {
			return "", nil, err
		}
		payload := mustJSON(status)
		return fmt.Sprintf("当前 bootstrap 状态：initialized=%t，user_count=%d。", status.Initialized, status.UserCount), payload, nil
	case ToolInspectRuntimeConfig:
		stage, snapshot, err := s.buildContextSnapshot(ctx, scope)
		if err != nil {
			return "", nil, err
		}
		return fmt.Sprintf("已读取当前运行配置，上下文阶段=%s。", stage), snapshot, nil
	case ToolTestSQLitePath:
		var input struct {
			SQLitePath string `json:"sqlite_path"`
		}
		if err := decodeInputs(rawInputs, &input); err != nil {
			return "", nil, err
		}
		cfg := config.Config{DBDriver: "sqlite", SQLitePath: strings.TrimSpace(input.SQLitePath)}
		if cfg.SQLitePath == "" {
			return "", nil, validationError("sqlite_path is required")
		}
		testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		pool, err := database.Open(testCtx, cfg)
		if err != nil {
			return "", nil, err
		}
		pool.Close()
		payload := mustJSON(map[string]any{"sqlite_path": input.SQLitePath, "ok": true})
		return "SQLite 路径测试通过。", payload, nil
	case ToolTestPostgresConnection:
		var input struct {
			DatabaseURL string `json:"database_url"`
		}
		if err := decodeInputs(rawInputs, &input); err != nil {
			return "", nil, err
		}
		if strings.TrimSpace(input.DatabaseURL) == "" {
			return "", nil, validationError("database_url is required")
		}
		cfg := config.Config{DBDriver: "postgres", DatabaseURL: strings.TrimSpace(input.DatabaseURL), DBMaxConns: 1}
		testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		pool, err := database.Open(testCtx, cfg)
		if err != nil {
			return "", nil, err
		}
		pool.Close()
		payload := mustJSON(map[string]any{"database_url": redactRuntimeURL(input.DatabaseURL), "ok": true})
		return "PostgreSQL 连接测试通过。", payload, nil
	case ToolSaveRuntimeConfig:
		var input runtimecfg.UpdateInput
		if err := decodeInputs(rawInputs, &input); err != nil {
			return "", nil, err
		}
		saved, err := s.runtime.Save(input)
		if err != nil {
			return "", nil, err
		}
		view := runtimecfg.ToView(s.runtime.Path(), runtimecfg.FileConfig{
			DBDriver:    saved.DBDriver,
			SQLitePath:  saved.SQLitePath,
			DatabaseURL: redactRuntimeURL(saved.DatabaseURL),
			AutoMigrate: saved.AutoMigrate,
		}, true)
		payload := mustJSON(view)
		return "运行配置已保存到 runtime.json；需要重启服务后才会生效。", payload, nil
	case ToolBootstrapCreateAdmin:
		var input teaching.BootstrapCreateAdminInput
		if err := decodeInputs(rawInputs, &input); err != nil {
			return "", nil, err
		}
		user, err := s.teaching.BootstrapCreateAdmin(ctx, input, teaching.AuditEntry{})
		if err != nil {
			return "", nil, err
		}
		payload := mustJSON(map[string]any{"user_id": user.ID, "username": user.Username, "display_name": user.DisplayName})
		return "首个管理员已创建。现在可以切换到该管理员身份继续配置系统。", payload, nil
	default:
		return "", nil, validationError("unknown assistant tool")
	}
}

func buildToolFollowUp(tool ToolName, status ToolCallStatus, summary string) string {
	if status == ToolCallSucceeded {
		return fmt.Sprintf("工具 `%s` 已执行成功。%s", tool, summary)
	}
	return fmt.Sprintf("工具 `%s` 执行失败。%s", tool, summary)
}

func decodeInputs(raw json.RawMessage, dst any) error {
	if len(raw) == 0 {
		return validationError("tool confirmation inputs are required")
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return validationError("tool confirmation inputs must be valid JSON")
	}
	return nil
}

func redactRuntimeURL(raw string) string {
	masked, ok := redactDSN(raw)
	if !ok {
		return ""
	}
	return masked
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}
