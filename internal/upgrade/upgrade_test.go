package upgrade

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDirOrdersSQLUpgradeStepsAndComputesMetadata(t *testing.T) {
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
	steps, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir failed: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 upgrade steps, got %d", len(steps))
	}
	if steps[0].Scope != ScopeDatabase || steps[0].Version != "000001" || steps[0].Name != "first" {
		t.Fatalf("unexpected first upgrade step: %#v", steps[0])
	}
	if steps[0].Checksum == "" {
		t.Fatal("checksum should be populated")
	}
}
