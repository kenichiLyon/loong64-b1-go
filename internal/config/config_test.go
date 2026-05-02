package config

import (
	"testing"
	"time"
)

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("DB_MAX_CONNS", "")
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
	if cfg.ReadyTimeout != 2*time.Second {
		t.Fatalf("unexpected ReadyTimeout: %s", cfg.ReadyTimeout)
	}
}

func TestLoadParsesOverrides(t *testing.T) {
	t.Setenv("HTTP_ADDR", "0.0.0.0:9000")
	t.Setenv("DB_MAX_CONNS", "17")
	t.Setenv("READY_TIMEOUT", "1500ms")

	cfg := Load()
	if cfg.HTTPAddr != "0.0.0.0:9000" {
		t.Fatalf("unexpected HTTPAddr: %s", cfg.HTTPAddr)
	}
	if cfg.DBMaxConns != 17 {
		t.Fatalf("unexpected DBMaxConns: %d", cfg.DBMaxConns)
	}
	if cfg.ReadyTimeout != 1500*time.Millisecond {
		t.Fatalf("unexpected ReadyTimeout: %s", cfg.ReadyTimeout)
	}
}
