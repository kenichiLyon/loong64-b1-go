package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/kenichiLyon/loong64-b1-go/internal/config"
)

func TestRuntimeConfigHandlerSaveAndLoad(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		AppEnv:            "development",
		RuntimeConfigPath: filepath.Join(t.TempDir(), "runtime.json"),
		DBDriver:          "sqlite",
		SQLitePath:        "./data/active.db",
		AutoMigrate:       true,
	}
	handler := newRuntimeConfigHandler(cfg, nil, false)

	body := bytes.NewBufferString(`{"db_driver":"postgres","database_url":"postgres://user:pass@127.0.0.1:5432/db?sslmode=disable","auto_migrate":false}`)
	saveReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/runtime-config", body)
	saveReq.Header.Set("X-Actor-ID", "admin-1")
	saveReq.Header.Set("X-Actor-Roles", "admin")
	saveRec := httptest.NewRecorder()
	handler.put(saveRec, saveReq)
	if saveRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", saveRec.Code, saveRec.Body.String())
	}
	var saved map[string]any
	if err := json.Unmarshal(saveRec.Body.Bytes(), &saved); err != nil {
		t.Fatalf("decode save response: %v", err)
	}
	if saved["message"] == "" {
		t.Fatalf("expected restart message, got %+v", saved)
	}

	loadReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/runtime-config", nil)
	loadReq.Header.Set("X-Actor-ID", "admin-1")
	loadReq.Header.Set("X-Actor-Roles", "admin")
	loadRec := httptest.NewRecorder()
	handler.get(loadRec, loadReq)
	if loadRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", loadRec.Code, loadRec.Body.String())
	}
	var summary struct {
		Exists bool `json:"exists"`
		Stored *struct {
			DBDriver       string `json:"db_driver"`
			DatabaseURLSet bool   `json:"database_url_set"`
		} `json:"stored"`
	}
	if err := json.Unmarshal(loadRec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode load response: %v", err)
	}
	if !summary.Exists || summary.Stored == nil || summary.Stored.DBDriver != "postgres" || !summary.Stored.DatabaseURLSet {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}
