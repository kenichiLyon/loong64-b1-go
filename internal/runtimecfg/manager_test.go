package runtimecfg

import (
	"path/filepath"
	"testing"
)

func TestManagerSaveAndLoadSQLite(t *testing.T) {
	t.Parallel()
	manager := New(filepath.Join(t.TempDir(), "runtime.json"))
	cfg, err := manager.Save(UpdateInput{DBDriver: "sqlite", SQLitePath: "./data/test.db", AutoMigrate: boolPtr(true)})
	if err != nil {
		t.Fatalf("save runtime config: %v", err)
	}
	if cfg.DBDriver != "sqlite" || cfg.SQLitePath != "./data/test.db" || cfg.AutoMigrate == nil || !*cfg.AutoMigrate {
		t.Fatalf("unexpected saved config: %+v", cfg)
	}
	loaded, exists, err := manager.Load()
	if err != nil {
		t.Fatalf("load runtime config: %v", err)
	}
	if !exists || loaded.SQLitePath != "./data/test.db" {
		t.Fatalf("unexpected loaded config: exists=%v cfg=%+v", exists, loaded)
	}
}

func TestNormalizeUpdateRequiresPostgresURL(t *testing.T) {
	t.Parallel()
	if _, err := normalizeUpdate(UpdateInput{DBDriver: "postgres"}); err == nil {
		t.Fatal("expected postgres validation error")
	}
}
