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
}

func TestLoadParsesOverrides(t *testing.T) {
	t.Setenv("HTTP_ADDR", "0.0.0.0:9000")
	t.Setenv("DB_MAX_CONNS", "17")
	t.Setenv("DEV_AUTH_BYPASS", "true")
	t.Setenv("MAX_UPLOAD_BYTES", "1024")
	t.Setenv("MAX_ARTIFACTS_PER_SUBMISSION", "3")
	t.Setenv("READY_TIMEOUT", "1500ms")

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
}
