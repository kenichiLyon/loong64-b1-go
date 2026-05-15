package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/kenichiLyon/loong64-b1-go/internal/authn"
	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/database"
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
	"github.com/kenichiLyon/loong64-b1-go/internal/upgrade"
)

func TestBootstrapStatusAndCreateAdmin(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DBDriver:          "sqlite",
		SQLitePath:        filepath.Join(t.TempDir(), "bootstrap.db"),
		UpgradeDir:        "../../migrations",
		RuntimeConfigPath: filepath.Join(t.TempDir(), "runtime.json"),
		AutoMigrate:       true,
		SessionCookieName: "test_session",
		SessionTTL:        24 * time.Hour,
	}
	pool, err := database.Open(t.Context(), cfg)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer pool.Close()
	if _, err := upgrade.NewRunner(pool, cfg.UpgradeDir).Up(t.Context()); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	service := teaching.NewService(teaching.NewSQLiteRepository(pool))
	authService := authn.NewService(authn.NewSQLiteRepository(pool), cfg)
	handler := newBootstrapHandler(service, cfg, nil, authService)

	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/bootstrap/status", nil)
	statusRec := httptest.NewRecorder()
	handler.status(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("status code: %d body=%s", statusRec.Code, statusRec.Body.String())
	}

	body := bytes.NewBufferString(`{"username":"admin1","display_name":"Admin One","employee_no":"A001","password":"test-pass"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/bootstrap/admin", body)
	createRec := httptest.NewRecorder()
	handler.createAdmin(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create code: %d body=%s", createRec.Code, createRec.Body.String())
	}
	var created struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if created.User.ID == "" {
		t.Fatalf("missing created user id: %+v", created)
	}
	if len(createRec.Result().Cookies()) == 0 {
		t.Fatal("expected session cookie after bootstrap")
	}
}
