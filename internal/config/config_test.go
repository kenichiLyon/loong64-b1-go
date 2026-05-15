package config

import (
	"testing"
	"time"
)

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("DB_MAX_CONNS", "")
	t.Setenv("DEV_AUTH_BYPASS", "")
	t.Setenv("MAX_UPLOAD_BYTES", "")
	t.Setenv("MAX_ARTIFACTS_PER_SUBMISSION", "")
	t.Setenv("READY_TIMEOUT", "")
	t.Setenv("LLM_TIMEOUT", "")
	t.Setenv("AI_GATEWAY_BASE_URL", "")
	t.Setenv("AI_GATEWAY_TIMEOUT", "")

	cfg := Load()
	if cfg.HTTPAddr != "127.0.0.1:8080" {
		t.Fatalf("unexpected HTTPAddr: %s", cfg.HTTPAddr)
	}
	if cfg.AppEnv != "development" {
		t.Fatalf("unexpected AppEnv: %s", cfg.AppEnv)
	}
	if cfg.DBMaxConns != 10 {
		t.Fatalf("unexpected DBMaxConns: %d", cfg.DBMaxConns)
	}
	if cfg.DevAuthBypass {
		t.Fatal("DevAuthBypass should be disabled by default")
	}
	if cfg.MaxUploadBytes != 50*1024*1024 {
		t.Fatalf("unexpected MaxUploadBytes: %d", cfg.MaxUploadBytes)
	}
	if cfg.MaxArtifactsPerSubmission != 20 {
		t.Fatalf("unexpected MaxArtifactsPerSubmission: %d", cfg.MaxArtifactsPerSubmission)
	}
	if cfg.ReadyTimeout != 2*time.Second {
		t.Fatalf("unexpected ReadyTimeout: %s", cfg.ReadyTimeout)
	}
	if cfg.LLMTimeout != 30*time.Second {
		t.Fatalf("unexpected LLMTimeout: %s", cfg.LLMTimeout)
	}
	if cfg.AIGatewayBaseURL != "" {
		t.Fatalf("unexpected AIGatewayBaseURL: %s", cfg.AIGatewayBaseURL)
	}
	if cfg.AIGatewayTimeout != 10*time.Second {
		t.Fatalf("unexpected AIGatewayTimeout: %s", cfg.AIGatewayTimeout)
	}
}

func TestLoadParsesOverrides(t *testing.T) {
	t.Setenv("HTTP_ADDR", "0.0.0.0:9000")
	t.Setenv("DB_MAX_CONNS", "17")
	t.Setenv("DEV_AUTH_BYPASS", "true")
	t.Setenv("MAX_UPLOAD_BYTES", "1024")
	t.Setenv("MAX_ARTIFACTS_PER_SUBMISSION", "3")
	t.Setenv("READY_TIMEOUT", "1500ms")
	t.Setenv("LLM_BASE_URL", "https://llm.example/v1")
	t.Setenv("LLM_MODEL", "qwen")
	t.Setenv("LLM_API_KEY", "test-key")
	t.Setenv("LLM_TIMEOUT", "45s")
	t.Setenv("AI_GATEWAY_BASE_URL", "http://127.0.0.1:8081")
	t.Setenv("AI_GATEWAY_TIMEOUT", "2500ms")

	cfg := Load()
	if cfg.HTTPAddr != "0.0.0.0:9000" {
		t.Fatalf("unexpected HTTPAddr: %s", cfg.HTTPAddr)
	}
	if cfg.DBMaxConns != 17 {
		t.Fatalf("unexpected DBMaxConns: %d", cfg.DBMaxConns)
	}
	if !cfg.DevAuthBypass {
		t.Fatal("DevAuthBypass should parse true override")
	}
	if cfg.MaxUploadBytes != 1024 {
		t.Fatalf("unexpected MaxUploadBytes: %d", cfg.MaxUploadBytes)
	}
	if cfg.MaxArtifactsPerSubmission != 3 {
		t.Fatalf("unexpected MaxArtifactsPerSubmission: %d", cfg.MaxArtifactsPerSubmission)
	}
	if cfg.ReadyTimeout != 1500*time.Millisecond {
		t.Fatalf("unexpected ReadyTimeout: %s", cfg.ReadyTimeout)
	}
	if cfg.LLMBaseURL != "https://llm.example/v1" || cfg.LLMModel != "qwen" || cfg.LLMAPIKey != "test-key" {
		t.Fatalf("unexpected LLM settings: %+v", cfg)
	}
	if cfg.LLMTimeout != 45*time.Second {
		t.Fatalf("unexpected LLMTimeout: %s", cfg.LLMTimeout)
	}
	if cfg.AIGatewayBaseURL != "http://127.0.0.1:8081" {
		t.Fatalf("unexpected AIGatewayBaseURL: %s", cfg.AIGatewayBaseURL)
	}
	if cfg.AIGatewayTimeout != 2500*time.Millisecond {
		t.Fatalf("unexpected AIGatewayTimeout: %s", cfg.AIGatewayTimeout)
	}
}

func TestLoadFallsBackForInvalidUploadLimitOverrides(t *testing.T) {
	tests := []struct {
		name      string
		maxBytes  string
		maxCount  string
		wantBytes int64
		wantCount int
	}{
		{name: "non numeric", maxBytes: "large", maxCount: "many", wantBytes: 50 * 1024 * 1024, wantCount: 20},
		{name: "negative", maxBytes: "-1", maxCount: "-1", wantBytes: 50 * 1024 * 1024, wantCount: 20},
		{name: "zero", maxBytes: "0", maxCount: "0", wantBytes: 50 * 1024 * 1024, wantCount: 20},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("MAX_UPLOAD_BYTES", tc.maxBytes)
			t.Setenv("MAX_ARTIFACTS_PER_SUBMISSION", tc.maxCount)
			cfg := Load()
			if cfg.MaxUploadBytes != tc.wantBytes {
				t.Fatalf("unexpected MaxUploadBytes: %d", cfg.MaxUploadBytes)
			}
			if cfg.MaxArtifactsPerSubmission != tc.wantCount {
				t.Fatalf("unexpected MaxArtifactsPerSubmission: %d", cfg.MaxArtifactsPerSubmission)
			}
		})
	}
}

func TestLoadPrefersAutoUpgradeAndKeepsAutoMigrateFallback(t *testing.T) {
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("AUTO_UPGRADE", "true")
	t.Setenv("AUTO_MIGRATE", "false")
	if cfg := Load(); !cfg.AutoMigrate {
		t.Fatal("AUTO_UPGRADE should enable automatic system upgrades")
	}

	t.Setenv("AUTO_UPGRADE", "")
	t.Setenv("AUTO_MIGRATE", "true")
	if cfg := Load(); !cfg.AutoMigrate {
		t.Fatal("AUTO_MIGRATE should remain a compatibility fallback")
	}
}
