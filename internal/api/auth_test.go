package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/kenichiLyon/loong64-b1-go/internal/authn"
	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/database"
	"github.com/kenichiLyon/loong64-b1-go/internal/migrate"
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
)

func TestAuthLoginLogoutAndMe(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DBDriver:          "sqlite",
		SQLitePath:        filepath.Join(t.TempDir(), "auth-api.db"),
		MigrationsDir:     "../../migrations",
		RuntimeConfigPath: filepath.Join(t.TempDir(), "runtime.json"),
		SessionCookieName: "test_session",
		SessionTTL:        24 * time.Hour,
	}
	pool, err := database.Open(t.Context(), cfg)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer pool.Close()
	if _, err := migrate.NewRunner(pool, cfg.MigrationsDir).Up(t.Context()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	teachingService := teaching.NewService(teaching.NewSQLiteRepository(pool))
	if _, err := teachingService.BootstrapCreateAdmin(t.Context(), teaching.BootstrapCreateAdminInput{
		Username:    "admin1",
		DisplayName: "Admin One",
		EmployeeNo:  "A001",
		Password:    "test-pass",
	}, teaching.AuditEntry{}); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}

	authService := authn.NewService(authn.NewSQLiteRepository(pool), cfg)
	authHandler := newAuthHandler(authService, cfg, nil, false)
	resolver := authHandler.resolveActor
	mux := http.NewServeMux()
	teaching.RegisterRoutes(mux, teaching.HTTPDependencies{
		Service:      teachingService,
		AppEnv:       cfg.AppEnv,
		ResolveActor: resolver,
	})
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.login)
	mux.HandleFunc("POST /api/v1/auth/logout", authHandler.logout)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin1","password":"test-pass"}`))
	loginRec := httptest.NewRecorder()
	mux.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login code: %d body=%s", loginRec.Code, loginRec.Body.String())
	}
	cookies := loginRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meReq.AddCookie(cookies[0])
	meRec := httptest.NewRecorder()
	mux.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me code: %d body=%s", meRec.Code, meRec.Body.String())
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq.AddCookie(cookies[0])
	logoutRec := httptest.NewRecorder()
	mux.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout code: %d", logoutRec.Code)
	}

	meAfterReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meAfterReq.AddCookie(cookies[0])
	meAfterRec := httptest.NewRecorder()
	mux.ServeHTTP(meAfterRec, meAfterReq)
	if meAfterRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d body=%s", meAfterRec.Code, meAfterRec.Body.String())
	}
}
