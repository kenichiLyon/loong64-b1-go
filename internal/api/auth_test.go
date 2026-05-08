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
		CSRFCookieName:    "test_csrf",
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
	mux.HandleFunc("PUT /api/v1/auth/password", authHandler.changePassword)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin1","password":"test-pass"}`))
	loginRec := httptest.NewRecorder()
	mux.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login code: %d body=%s", loginRec.Code, loginRec.Body.String())
	}
	cookies := loginRec.Result().Cookies()
	if len(cookies) < 2 {
		t.Fatal("expected session and csrf cookies")
	}
	csrf := cookieByName(cookies, cfg.CSRFCookieName)
	if csrf == nil {
		t.Fatal("expected csrf cookie")
	}
	sessionCookie := cookieByName(cookies, cfg.SessionCookieName)
	if sessionCookie == nil {
		t.Fatal("expected session cookie")
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meReq.AddCookie(sessionCookie)
	meRec := httptest.NewRecorder()
	mux.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me code: %d body=%s", meRec.Code, meRec.Body.String())
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq.AddCookie(sessionCookie)
	logoutReq.AddCookie(csrf)
	logoutReq.Header.Set("X-CSRF-Token", csrf.Value)
	logoutRec := httptest.NewRecorder()
	mux.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout code: %d", logoutRec.Code)
	}

	meAfterReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meAfterReq.AddCookie(sessionCookie)
	meAfterRec := httptest.NewRecorder()
	mux.ServeHTTP(meAfterRec, meAfterReq)
	if meAfterRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d body=%s", meAfterRec.Code, meAfterRec.Body.String())
	}
}

func TestAdminCanSetUserPasswordAndLogin(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DBDriver:          "sqlite",
		SQLitePath:        filepath.Join(t.TempDir(), "auth-admin.db"),
		MigrationsDir:     "../../migrations",
		RuntimeConfigPath: filepath.Join(t.TempDir(), "runtime.json"),
		SessionCookieName: "test_session",
		CSRFCookieName:    "test_csrf",
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
	authService := authn.NewService(authn.NewSQLiteRepository(pool), cfg)
	if _, err := teachingService.BootstrapCreateAdmin(t.Context(), teaching.BootstrapCreateAdminInput{
		Username:    "admin1",
		DisplayName: "Admin One",
		EmployeeNo:  "A001",
		Password:    "test-pass",
	}, teaching.AuditEntry{}); err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}

	authHandler := newAuthHandler(authService, cfg, nil, false)
	resolver := authHandler.resolveActor
	mux := http.NewServeMux()
	teaching.RegisterRoutes(mux, teaching.HTTPDependencies{
		Service:      teachingService,
		AppEnv:       cfg.AppEnv,
		ResolveActor: resolver,
	})
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.login)
	mux.HandleFunc("PUT /api/v1/auth/password", authHandler.changePassword)

	loginRec := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin1","password":"test-pass"}`))
	mux.ServeHTTP(loginRec, loginReq)
	cookies := loginRec.Result().Cookies()
	if len(cookies) < 2 {
		t.Fatal("expected admin session and csrf cookies")
	}
	sessionCookie := cookieByName(cookies, cfg.SessionCookieName)
	csrf := cookieByName(cookies, cfg.CSRFCookieName)
	if sessionCookie == nil || csrf == nil {
		t.Fatal("expected session and csrf cookies by name")
	}

	createUserReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBufferString(`{"username":"student1","display_name":"Student One","student_no":"S001","roles":["student"]}`))
	createUserReq.AddCookie(sessionCookie)
	createUserReq.AddCookie(csrf)
	createUserReq.Header.Set("X-CSRF-Token", csrf.Value)
	createUserRec := httptest.NewRecorder()
	mux.ServeHTTP(createUserRec, createUserReq)
	if createUserRec.Code != http.StatusCreated {
		t.Fatalf("create user code: %d body=%s", createUserRec.Code, createUserRec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createUserRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created user: %v", err)
	}

	setPasswordReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/"+created.ID+"/password", bytes.NewBufferString(`{"password":"student-pass"}`))
	setPasswordReq.SetPathValue("userID", created.ID)
	setPasswordReq.AddCookie(sessionCookie)
	setPasswordReq.AddCookie(csrf)
	setPasswordReq.Header.Set("X-CSRF-Token", csrf.Value)
	setPasswordRec := httptest.NewRecorder()
	mux.ServeHTTP(setPasswordRec, setPasswordReq)
	if setPasswordRec.Code != http.StatusNoContent {
		t.Fatalf("set password code: %d body=%s", setPasswordRec.Code, setPasswordRec.Body.String())
	}

	studentLoginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"student1","password":"student-pass"}`))
	studentLoginRec := httptest.NewRecorder()
	mux.ServeHTTP(studentLoginRec, studentLoginReq)
	if studentLoginRec.Code != http.StatusOK {
		t.Fatalf("student login code: %d body=%s", studentLoginRec.Code, studentLoginRec.Body.String())
	}
	studentSession := cookieByName(studentLoginRec.Result().Cookies(), cfg.SessionCookieName)
	if studentSession == nil {
		t.Fatal("expected student session cookie")
	}

	resetAgainReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/users/"+created.ID+"/password", bytes.NewBufferString(`{"password":"student-pass-2"}`))
	resetAgainReq.SetPathValue("userID", created.ID)
	resetAgainReq.AddCookie(sessionCookie)
	resetAgainReq.AddCookie(csrf)
	resetAgainReq.Header.Set("X-CSRF-Token", csrf.Value)
	resetAgainRec := httptest.NewRecorder()
	mux.ServeHTTP(resetAgainRec, resetAgainReq)
	if resetAgainRec.Code != http.StatusNoContent {
		t.Fatalf("second set password code: %d body=%s", resetAgainRec.Code, resetAgainRec.Body.String())
	}

	studentMeReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	studentMeReq.AddCookie(studentSession)
	studentMeRec := httptest.NewRecorder()
	mux.ServeHTTP(studentMeRec, studentMeReq)
	if studentMeRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked student session, got %d body=%s", studentMeRec.Code, studentMeRec.Body.String())
	}

	studentOldLoginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"student1","password":"student-pass"}`))
	studentOldLoginRec := httptest.NewRecorder()
	mux.ServeHTTP(studentOldLoginRec, studentOldLoginReq)
	if studentOldLoginRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected old student password to fail, got %d body=%s", studentOldLoginRec.Code, studentOldLoginRec.Body.String())
	}

	studentNewLoginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"student1","password":"student-pass-2"}`))
	studentNewLoginRec := httptest.NewRecorder()
	mux.ServeHTTP(studentNewLoginRec, studentNewLoginReq)
	if studentNewLoginRec.Code != http.StatusOK {
		t.Fatalf("student new login code: %d body=%s", studentNewLoginRec.Code, studentNewLoginRec.Body.String())
	}
}

func TestAuthRejectsMutatingRequestWithoutCSRFFromSession(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DBDriver:          "sqlite",
		SQLitePath:        filepath.Join(t.TempDir(), "auth-csrf.db"),
		MigrationsDir:     "../../migrations",
		RuntimeConfigPath: filepath.Join(t.TempDir(), "runtime.json"),
		SessionCookieName: "test_session",
		CSRFCookieName:    "test_csrf",
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
	mux := http.NewServeMux()
	teaching.RegisterRoutes(mux, teaching.HTTPDependencies{
		Service:      teachingService,
		AppEnv:       cfg.AppEnv,
		ResolveActor: authHandler.resolveActor,
	})
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.login)
	mux.HandleFunc("PUT /api/v1/auth/password", authHandler.changePassword)

	loginRec := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin1","password":"test-pass"}`))
	mux.ServeHTTP(loginRec, loginReq)
	sessionCookie := cookieByName(loginRec.Result().Cookies(), cfg.SessionCookieName)
	if sessionCookie == nil {
		t.Fatal("expected session cookie")
	}

	createUserReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBufferString(`{"username":"student1","display_name":"Student One","student_no":"S001","roles":["student"]}`))
	createUserReq.AddCookie(sessionCookie)
	createUserRec := httptest.NewRecorder()
	mux.ServeHTTP(createUserRec, createUserReq)
	if createUserRec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without csrf header, got %d body=%s", createUserRec.Code, createUserRec.Body.String())
	}
}

func TestAuthChangeOwnPasswordRequiresRelogin(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DBDriver:               "sqlite",
		SQLitePath:             filepath.Join(t.TempDir(), "auth-self-password.db"),
		MigrationsDir:          "../../migrations",
		RuntimeConfigPath:      filepath.Join(t.TempDir(), "runtime.json"),
		SessionCookieName:      "test_session",
		CSRFCookieName:         "test_csrf",
		SessionTTL:             24 * time.Hour,
		SessionCleanupInterval: time.Minute,
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
	mux.HandleFunc("PUT /api/v1/auth/password", authHandler.changePassword)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin1","password":"test-pass"}`))
	loginRec := httptest.NewRecorder()
	mux.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login code: %d body=%s", loginRec.Code, loginRec.Body.String())
	}
	sessionCookie := cookieByName(loginRec.Result().Cookies(), cfg.SessionCookieName)
	csrf := cookieByName(loginRec.Result().Cookies(), cfg.CSRFCookieName)
	if sessionCookie == nil || csrf == nil {
		t.Fatal("expected session and csrf cookies")
	}

	changeReq := httptest.NewRequest(http.MethodPut, "/api/v1/auth/password", bytes.NewBufferString(`{"current_password":"test-pass","new_password":"new-pass"}`))
	changeReq.AddCookie(sessionCookie)
	changeReq.AddCookie(csrf)
	changeReq.Header.Set("X-CSRF-Token", csrf.Value)
	changeRec := httptest.NewRecorder()
	mux.ServeHTTP(changeRec, changeReq)
	if changeRec.Code != http.StatusNoContent {
		t.Fatalf("change password code: %d body=%s", changeRec.Code, changeRec.Body.String())
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meReq.AddCookie(sessionCookie)
	meRec := httptest.NewRecorder()
	mux.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked session after self password change, got %d body=%s", meRec.Code, meRec.Body.String())
	}

	oldLoginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin1","password":"test-pass"}`))
	oldLoginRec := httptest.NewRecorder()
	mux.ServeHTTP(oldLoginRec, oldLoginReq)
	if oldLoginRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected old password to fail, got %d body=%s", oldLoginRec.Code, oldLoginRec.Body.String())
	}

	newLoginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin1","password":"new-pass"}`))
	newLoginRec := httptest.NewRecorder()
	mux.ServeHTTP(newLoginRec, newLoginReq)
	if newLoginRec.Code != http.StatusOK {
		t.Fatalf("expected new password login to succeed, got %d body=%s", newLoginRec.Code, newLoginRec.Body.String())
	}
}

func cookieByName(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
