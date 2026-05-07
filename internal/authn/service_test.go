package authn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/database"
	"github.com/kenichiLyon/loong64-b1-go/internal/migrate"
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
)

func TestLoginResolveAndLogout(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DBDriver:          "sqlite",
		SQLitePath:        filepath.Join(t.TempDir(), "auth.db"),
		MigrationsDir:     "../../migrations",
		SessionCookieName: "test_session",
		SessionTTL:        24 * time.Hour,
	}
	pool, err := database.Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer pool.Close()
	if _, err := migrate.NewRunner(pool, cfg.MigrationsDir).Up(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	teachingService := teaching.NewService(teaching.NewSQLiteRepository(pool))
	if _, err := teachingService.BootstrapCreateAdmin(context.Background(), teaching.BootstrapCreateAdminInput{
		Username:    "admin1",
		DisplayName: "Admin One",
		EmployeeNo:  "A001",
		Password:    "test-pass",
	}, teaching.AuditEntry{}); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}

	service := NewService(NewSQLiteRepository(pool), cfg)
	session, token, err := service.Login(context.Background(), "admin1", "test-pass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if token == "" || session.User.ID == "" {
		t.Fatalf("unexpected session: %+v token=%q", session, token)
	}

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: cfg.SessionCookieName, Value: token})
	actor, err := service.ResolveRequestActor(context.Background(), req)
	if err != nil {
		t.Fatalf("resolve actor: %v", err)
	}
	if actor.ID != session.User.ID || !actor.Has(teaching.RoleAdmin) {
		t.Fatalf("unexpected actor: %+v", actor)
	}

	if err := service.Logout(context.Background(), req); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := service.ResolveRequestActor(context.Background(), req); teaching.ErrorKindOf(err) != teaching.KindNotFound && teaching.ErrorKindOf(err) != teaching.KindUnauthorized {
		t.Fatalf("expected unauthorized/not_found after logout, got %v", err)
	}
}
