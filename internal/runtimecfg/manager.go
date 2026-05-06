package runtimecfg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileConfig struct {
	DBDriver    string `json:"db_driver,omitempty"`
	SQLitePath  string `json:"sqlite_path,omitempty"`
	DatabaseURL string `json:"database_url,omitempty"`
	AutoMigrate *bool  `json:"auto_migrate,omitempty"`
}

type View struct {
	DBDriver          string `json:"db_driver"`
	SQLitePath        string `json:"sqlite_path,omitempty"`
	DatabaseURL       string `json:"database_url,omitempty"`
	DatabaseURLSet    bool   `json:"database_url_set"`
	AutoMigrate       bool   `json:"auto_migrate"`
	RequiresRestart   bool   `json:"requires_restart,omitempty"`
	RuntimeConfigPath string `json:"runtime_config_path,omitempty"`
}

type Summary struct {
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	Active  View   `json:"active"`
	Stored  *View  `json:"stored,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type UpdateInput struct {
	DBDriver    string `json:"db_driver"`
	SQLitePath  string `json:"sqlite_path,omitempty"`
	DatabaseURL string `json:"database_url,omitempty"`
	AutoMigrate *bool  `json:"auto_migrate,omitempty"`
}

type Manager struct {
	path string
}

func New(path string) *Manager {
	if strings.TrimSpace(path) == "" {
		path = "./config/runtime.json"
	}
	return &Manager{path: path}
}

func (m *Manager) Path() string {
	if m == nil {
		return ""
	}
	return m.path
}

func (m *Manager) Load() (FileConfig, bool, error) {
	if m == nil || strings.TrimSpace(m.path) == "" {
		return FileConfig{}, false, errors.New("runtime config path is not configured")
	}
	content, err := os.ReadFile(m.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return FileConfig{}, false, nil
		}
		return FileConfig{}, false, fmt.Errorf("read runtime config: %w", err)
	}
	var cfg FileConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return FileConfig{}, true, fmt.Errorf("decode runtime config: %w", err)
	}
	return cfg, true, nil
}

func (m *Manager) Save(input UpdateInput) (FileConfig, error) {
	cfg, err := normalizeUpdate(input)
	if err != nil {
		return FileConfig{}, err
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o750); err != nil {
		return FileConfig{}, fmt.Errorf("create runtime config directory: %w", err)
	}
	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return FileConfig{}, fmt.Errorf("encode runtime config: %w", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(m.path, payload, 0o640); err != nil {
		return FileConfig{}, fmt.Errorf("write runtime config: %w", err)
	}
	return cfg, nil
}

func normalizeUpdate(input UpdateInput) (FileConfig, error) {
	driver := strings.ToLower(strings.TrimSpace(input.DBDriver))
	if driver == "" {
		driver = "sqlite"
	}
	autoMigrate := true
	if input.AutoMigrate != nil {
		autoMigrate = *input.AutoMigrate
	} else if driver == "postgres" {
		autoMigrate = false
	}
	switch driver {
	case "sqlite":
		sqlitePath := strings.TrimSpace(input.SQLitePath)
		if sqlitePath == "" {
			sqlitePath = "./data/loong64-b1-go.db"
		}
		return FileConfig{DBDriver: driver, SQLitePath: sqlitePath, AutoMigrate: boolPtr(autoMigrate)}, nil
	case "postgres":
		databaseURL := strings.TrimSpace(input.DatabaseURL)
		if databaseURL == "" {
			return FileConfig{}, errors.New("database_url is required when db_driver=postgres")
		}
		return FileConfig{DBDriver: driver, DatabaseURL: databaseURL, AutoMigrate: boolPtr(autoMigrate)}, nil
	default:
		return FileConfig{}, fmt.Errorf("unsupported db_driver: %s", driver)
	}
}

func ToView(path string, cfg FileConfig, requiresRestart bool) View {
	autoMigrate := cfg.AutoMigrate != nil && *cfg.AutoMigrate
	return View{
		DBDriver:          cfg.DBDriver,
		SQLitePath:        cfg.SQLitePath,
		DatabaseURL:       cfg.DatabaseURL,
		DatabaseURLSet:    strings.TrimSpace(cfg.DatabaseURL) != "",
		AutoMigrate:       autoMigrate,
		RequiresRestart:   requiresRestart,
		RuntimeConfigPath: path,
	}
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}
