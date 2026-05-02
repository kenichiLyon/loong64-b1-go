package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type LocalStore struct {
	root string
}

func NewLocal(root string) *LocalStore {
	if root == "" {
		root = "./storage"
	}
	return &LocalStore{root: root}
}

func (s *LocalStore) Root() string {
	return s.root
}

func (s *LocalStore) Ensure(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, dir := range []string{"artifacts", "reports", "tmp"} {
		if err := os.MkdirAll(filepath.Join(s.root, dir), 0o750); err != nil {
			return fmt.Errorf("create storage directory %s: %w", dir, err)
		}
	}
	return nil
}

func (s *LocalStore) Resolve(key string) (string, error) {
	if key == "" {
		return "", errors.New("storage key is empty")
	}
	cleaned := filepath.Clean(filepath.FromSlash(key))
	if filepath.IsAbs(cleaned) || cleaned == "." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) || cleaned == ".." {
		return "", fmt.Errorf("unsafe storage key: %s", key)
	}
	return filepath.Join(s.root, cleaned), nil
}

type Checker struct {
	Store *LocalStore
}

func (c Checker) Name() string { return "storage" }

func (c Checker) Check(ctx context.Context) error {
	if c.Store == nil {
		return errors.New("storage is not configured")
	}
	if err := c.Store.Ensure(ctx); err != nil {
		return err
	}
	probe := filepath.Join(c.Store.root, ".healthcheck")
	if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
		return fmt.Errorf("write storage probe: %w", err)
	}
	if err := os.Remove(probe); err != nil {
		return fmt.Errorf("remove storage probe: %w", err)
	}
	return nil
}
