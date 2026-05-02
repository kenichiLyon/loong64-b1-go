package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStoreEnsureCreatesExpectedDirectories(t *testing.T) {
	root := t.TempDir()
	store := NewLocal(root)
	if err := store.Ensure(context.Background()); err != nil {
		t.Fatalf("Ensure failed: %v", err)
	}
	for _, dir := range []string{"artifacts", "reports", "tmp"} {
		info, err := os.Stat(filepath.Join(root, dir))
		if err != nil {
			t.Fatalf("missing directory %s: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", dir)
		}
	}
}

func TestResolveRejectsPathTraversal(t *testing.T) {
	store := NewLocal(t.TempDir())
	if _, err := store.Resolve("../secret"); err == nil {
		t.Fatal("expected traversal key to be rejected")
	}
	if _, err := store.Resolve("artifacts/report.pdf"); err != nil {
		t.Fatalf("safe key rejected: %v", err)
	}
}
