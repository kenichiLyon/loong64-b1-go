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
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
	"github.com/kenichiLyon/loong64-b1-go/internal/upgrade"
)

func TestLoginResolveAndLogout(t *testing.T) {
	t.Parallel()

	cfg, pool, service := newSQLiteAuthService(t, "auth.db")
	defer pool.Close()

	session, token, err := service.Login(context.Background(), "admin1", "test-pass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if token == "" || session.User.ID == "" {
		t.Fatalf("unexpected session: %+v token=%q", session, token)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
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

func TestChangePasswordRevokesAllSessions(t *testing.T) {
	t.Parallel()

	cfg, pool, service := newSQLiteAuthService(t, "auth-change-password.db")
	defer pool.Close()

	_, firstToken, err := service.Login(context.Background(), "admin1", "test-pass")
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	_, secondToken, err := service.Login(context.Background(), "admin1", "test-pass")
	if err != nil {
		t.Fatalf("second login: %v", err)
	}

	changeReq := httptest.NewRequest(http.MethodPut, "/api/v1/auth/password", nil)
	changeReq.AddCookie(&http.Cookie{Name: cfg.SessionCookieName, Value: firstToken})
	if err := service.ChangePassword(context.Background(), changeReq, "test-pass", "new-pass"); err != nil {
		t.Fatalf("change password: %v", err)
	}

	firstReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	firstReq.AddCookie(&http.Cookie{Name: cfg.SessionCookieName, Value: firstToken})
	if _, err := service.ResolveRequestActor(context.Background(), firstReq); teaching.ErrorKindOf(err) != teaching.KindNotFound && teaching.ErrorKindOf(err) != teaching.KindUnauthorized {
		t.Fatalf("expected first session invalid, got %v", err)
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	secondReq.AddCookie(&http.Cookie{Name: cfg.SessionCookieName, Value: secondToken})
	if _, err := service.ResolveRequestActor(context.Background(), secondReq); teaching.ErrorKindOf(err) != teaching.KindNotFound && teaching.ErrorKindOf(err) != teaching.KindUnauthorized {
		t.Fatalf("expected second session invalid, got %v", err)
	}

	if _, _, err := service.Login(context.Background(), "admin1", "test-pass"); teaching.ErrorKindOf(err) != teaching.KindUnauthorized {
		t.Fatalf("expected old password to fail, got %v", err)
	}
	if _, _, err := service.Login(context.Background(), "admin1", "new-pass"); err != nil {
		t.Fatalf("expected new password login to succeed, got %v", err)
	}
}

func TestCleanupExpiredSessionsDeletesExpiredRows(t *testing.T) {
	t.Parallel()

	cfg, pool, service := newSQLiteAuthService(t, "auth-cleanup.db")
	defer pool.Close()

	_, token, err := service.Login(context.Background(), "admin1", "test-pass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	tokenHash := hashToken(token)
	expiredAt := time.Now().UTC().Add(-2 * time.Hour)
	if _, err := pool.SQLDB().ExecContext(context.Background(), `UPDATE auth_sessions SET expires_at = ?, last_seen_at = ? WHERE token_hash = ?`, expiredAt, expiredAt, tokenHash); err != nil {
		t.Fatalf("expire session manually: %v", err)
	}

	deleted, err := service.CleanupExpiredSessions(context.Background())
	if err != nil {
		t.Fatalf("cleanup expired sessions: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected one deleted session, got %d", deleted)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: cfg.SessionCookieName, Value: token})
	if _, err := service.ResolveRequestActor(context.Background(), req); teaching.ErrorKindOf(err) != teaching.KindNotFound && teaching.ErrorKindOf(err) != teaching.KindUnauthorized {
		t.Fatalf("expected expired session to be removed, got %v", err)
	}
}

func TestRefreshSessionIfDueExtendsExpiryAndRewritesCookies(t *testing.T) {
	t.Parallel()

	cfg, pool, _ := newSQLiteAuthService(t, "auth-refresh.db")
	defer pool.Close()
	cfg.SessionRefreshInterval = time.Minute
	service := NewService(NewSQLiteRepository(pool), cfg)

	_, token, err := service.Login(context.Background(), "admin1", "test-pass")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	tokenHash := hashToken(token)
	oldLastSeen := time.Now().UTC().Add(-2 * time.Minute)
	oldExpires := time.Now().UTC().Add(30 * time.Second)
	if _, err := pool.SQLDB().ExecContext(context.Background(), `UPDATE auth_sessions SET last_seen_at = ?, expires_at = ? WHERE token_hash = ?`, oldLastSeen, oldExpires, tokenHash); err != nil {
		t.Fatalf("seed session timestamps: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Host = "example.com"
	req.AddCookie(&http.Cookie{Name: cfg.SessionCookieName, Value: token})
	req.AddCookie(&http.Cookie{Name: cfg.CSRFCookieName, Value: "csrf-token"})
	rec := httptest.NewRecorder()

	if err := service.RefreshSessionIfDue(context.Background(), req, rec); err != nil {
		t.Fatalf("refresh session: %v", err)
	}

	session, err := service.repo.GetSessionByTokenHash(context.Background(), tokenHash)
	if err != nil {
		t.Fatalf("load refreshed session: %v", err)
	}
	if !session.LastSeenAt.After(oldLastSeen) {
		t.Fatalf("expected last_seen_at to advance, got %s <= %s", session.LastSeenAt, oldLastSeen)
	}
	if !session.ExpiresAt.After(oldExpires) {
		t.Fatalf("expected expires_at to advance, got %s <= %s", session.ExpiresAt, oldExpires)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) < 2 {
		t.Fatalf("expected refreshed session and csrf cookies, got %d", len(cookies))
	}
}

func TestValidateCSRFAcceptsSameOriginAndRejectsCrossOrigin(t *testing.T) {
	t.Parallel()

	service := NewService(nil, config.Config{
		SessionCookieName: "test_session",
		CSRFCookieName:    "test_csrf",
	})

	sameOriginReq := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/auth/logout", nil)
	sameOriginReq.Host = "example.com"
	sameOriginReq.Header.Set("Origin", "http://example.com")
	sameOriginReq.Header.Set("X-CSRF-Token", "csrf-token")
	sameOriginReq.AddCookie(&http.Cookie{Name: "test_session", Value: "session-token"})
	sameOriginReq.AddCookie(&http.Cookie{Name: "test_csrf", Value: "csrf-token"})
	if err := service.ValidateCSRF(sameOriginReq); err != nil {
		t.Fatalf("expected same-origin request to pass, got %v", err)
	}

	crossOriginReq := httptest.NewRequest(http.MethodPost, "http://example.com/api/v1/auth/logout", nil)
	crossOriginReq.Host = "example.com"
	crossOriginReq.Header.Set("Origin", "https://evil.example")
	crossOriginReq.Header.Set("X-CSRF-Token", "csrf-token")
	crossOriginReq.AddCookie(&http.Cookie{Name: "test_session", Value: "session-token"})
	crossOriginReq.AddCookie(&http.Cookie{Name: "test_csrf", Value: "csrf-token"})
	if err := service.ValidateCSRF(crossOriginReq); teaching.ErrorKindOf(err) != teaching.KindForbidden {
		t.Fatalf("expected cross-origin request to be forbidden, got %v", err)
	}
}

func newSQLiteAuthService(t *testing.T, dbName string) (config.Config, *database.Pool, *Service) {
	t.Helper()

	cfg := config.Config{
		DBDriver:               "sqlite",
		SQLitePath:             filepath.Join(t.TempDir(), dbName),
		UpgradeDir:             "../../migrations",
		SessionCookieName:      "test_session",
		CSRFCookieName:         "test_csrf",
		SessionTTL:             24 * time.Hour,
		SessionCleanupInterval: time.Minute,
	}
	pool, err := database.Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := upgrade.NewRunner(pool, cfg.UpgradeDir).Up(context.Background()); err != nil {
		t.Fatalf("upgrade: %v", err)
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

	return cfg, pool, NewService(NewSQLiteRepository(pool), cfg)
}
