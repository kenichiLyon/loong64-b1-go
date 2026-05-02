package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDirOrdersSQLMigrationsAndComputesMetadata(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "000002_second.sql"), []byte("select 2;"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "000001_first.sql"), []byte("select 1;"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("ignore"), 0o600); err != nil {
		t.Fatal(err)
	}
	migrations, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir failed: %v", err)
	}
	if len(migrations) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(migrations))
	}
	if migrations[0].Version != "000001" || migrations[0].Name != "first" {
		t.Fatalf("unexpected first migration: %#v", migrations[0])
	}
	if migrations[0].Checksum == "" {
		t.Fatal("checksum should be populated")
	}
}
