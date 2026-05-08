package authn

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
)

const (
	defaultSessionCookieName = "loong64_b1_session"
	defaultCSRFCookieName    = "loong64_b1_csrf"
	defaultSessionTTL        = 168 * time.Hour
)

type UserAuth struct {
	ID           string
	Username     string
	DisplayName  string
	Status       string
	Roles        []teaching.Role
	PasswordHash string
}

type Session struct {
	ID         string
	UserID     string
	TokenHash  string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	LastSeenAt time.Time
	User       UserAuth
}

type Repository interface {
	GetUserAuthByUsername(context.Context, string) (UserAuth, error)
	GetUserAuthByID(context.Context, string) (UserAuth, error)
	CreateSession(context.Context, Session) (Session, error)
	GetSessionByTokenHash(context.Context, string) (Session, error)
	TouchSession(context.Context, Session) (Session, error)
	DeleteSessionByTokenHash(context.Context, string) error
	RotatePassword(context.Context, string, string) error
	DeleteExpiredSessions(context.Context, time.Time) (int64, error)
}

type Service struct {
	repo          Repository
	cfg           config.Config
	cleanupMu     sync.Mutex
	lastCleanupAt time.Time
}

func NewService(repo Repository, cfg config.Config) *Service {
	return &Service{repo: repo, cfg: cfg}
}

func (s *Service) Login(ctx context.Context, username, password string) (Session, string, error) {
	if s == nil || s.repo == nil {
		return Session{}, "", unavailableError("auth service is not configured", nil)
	}
	s.cleanupExpiredSessionsIfDue(ctx)
	user, err := s.repo.GetUserAuthByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		return Session{}, "", err
	}
	if user.Status != "active" {
		return Session{}, "", forbiddenError("user is disabled")
	}
	if strings.TrimSpace(user.PasswordHash) == "" {
		return Session{}, "", unauthorizedError("password is not configured for this account")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return Session{}, "", unauthorizedError("invalid username or password")
	}
	return s.createSession(ctx, user)
}

func (s *Service) CreateSessionForUser(ctx context.Context, userID string) (Session, string, error) {
	if s == nil || s.repo == nil {
		return Session{}, "", unavailableError("auth service is not configured", nil)
	}
	user, err := s.repo.GetUserAuthByID(ctx, strings.TrimSpace(userID))
	if err != nil {
		return Session{}, "", err
	}
	return s.createSession(ctx, user)
}

func (s *Service) createSession(ctx context.Context, user UserAuth) (Session, string, error) {
	token, tokenHash, err := newSessionToken()
	if err != nil {
		return Session{}, "", unavailableError("failed to create session token", err)
	}
	now := time.Now().UTC()
	session := Session{
		ID:         teaching.NewID("ses"),
		UserID:     user.ID,
		TokenHash:  tokenHash,
		ExpiresAt:  now.Add(s.sessionTTL()),
		LastSeenAt: now,
		User:       user,
	}
	created, err := s.repo.CreateSession(ctx, session)
	if err != nil {
		return Session{}, "", err
	}
	return created, token, nil
}

func (s *Service) ResolveRequestActor(ctx context.Context, r *http.Request) (teaching.Actor, error) {
	if s == nil || s.repo == nil {
		return teaching.Actor{}, unavailableError("auth service is not configured", nil)
	}
	s.cleanupExpiredSessionsIfDue(ctx)
	session, err := s.sessionFromRequest(ctx, r)
	if err != nil {
		return teaching.Actor{}, err
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		_ = s.repo.DeleteSessionByTokenHash(ctx, session.TokenHash)
		return teaching.Actor{}, unauthorizedError("session has expired")
	}
	session.LastSeenAt = time.Now().UTC()
	updated, err := s.repo.TouchSession(ctx, session)
	if err == nil {
		session = updated
	}
	return teaching.NewActor(session.User.ID, session.User.Roles)
}

func (s *Service) Logout(ctx context.Context, r *http.Request) error {
	cookie, err := r.Cookie(s.sessionCookieName())
	if err != nil {
		return nil
	}
	return s.repo.DeleteSessionByTokenHash(ctx, hashToken(cookie.Value))
}

func (s *Service) ChangePassword(ctx context.Context, r *http.Request, currentPassword, newPassword string) error {
	if s == nil || s.repo == nil {
		return unavailableError("auth service is not configured", nil)
	}
	s.cleanupExpiredSessionsIfDue(ctx)
	session, err := s.sessionFromRequest(ctx, r)
	if err != nil {
		return err
	}
	if strings.TrimSpace(session.User.PasswordHash) == "" {
		return unauthorizedError("password is not configured for this account")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(session.User.PasswordHash), []byte(currentPassword)); err != nil {
		return unauthorizedError("current password is invalid")
	}
	if strings.TrimSpace(newPassword) == "" {
		return validationError("new password is required")
	}
	if strings.TrimSpace(currentPassword) == strings.TrimSpace(newPassword) {
		return validationError("new password must be different from current password")
	}
	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		return unavailableError("failed to hash password", err)
	}
	return s.repo.RotatePassword(ctx, session.UserID, passwordHash)
}

func (s *Service) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, unavailableError("auth service is not configured", nil)
	}
	return s.repo.DeleteExpiredSessions(ctx, time.Now().UTC())
}

func (s *Service) ValidateCSRF(r *http.Request) error {
	if s == nil || !csrfProtectedMethod(r.Method) {
		return nil
	}
	if _, err := r.Cookie(s.sessionCookieName()); err != nil {
		return nil
	}
	csrfCookie, err := r.Cookie(s.csrfCookieName())
	if err != nil {
		return forbiddenError("csrf cookie is required")
	}
	headerToken := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
	if headerToken == "" {
		return forbiddenError("csrf token is required")
	}
	if subtle.ConstantTimeCompare([]byte(headerToken), []byte(strings.TrimSpace(csrfCookie.Value))) != 1 {
		return forbiddenError("csrf token is invalid")
	}
	return nil
}

func (s *Service) NewCSRFCookieValue() (string, error) {
	token, _, err := newSessionToken()
	if err != nil {
		return "", unavailableError("failed to create csrf token", err)
	}
	return token, nil
}

func (s *Service) WriteSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.sessionCookieName(),
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SessionSecureCookie,
		Expires:  time.Now().UTC().Add(s.sessionTTL()),
		MaxAge:   int(s.sessionTTL().Seconds()),
	})
}

func (s *Service) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.sessionCookieName(),
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SessionSecureCookie,
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
	})
}

func (s *Service) WriteCSRFCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.csrfCookieName(),
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SessionSecureCookie,
		Expires:  time.Now().UTC().Add(s.sessionTTL()),
		MaxAge:   int(s.sessionTTL().Seconds()),
	})
}

func (s *Service) ClearCSRFCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.csrfCookieName(),
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SessionSecureCookie,
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
	})
}

func (s *Service) sessionCookieName() string {
	if s != nil && strings.TrimSpace(s.cfg.SessionCookieName) != "" {
		return strings.TrimSpace(s.cfg.SessionCookieName)
	}
	return defaultSessionCookieName
}

func (s *Service) csrfCookieName() string {
	if s != nil && strings.TrimSpace(s.cfg.CSRFCookieName) != "" {
		return strings.TrimSpace(s.cfg.CSRFCookieName)
	}
	return defaultCSRFCookieName
}

func (s *Service) sessionTTL() time.Duration {
	if s != nil && s.cfg.SessionTTL > 0 {
		return s.cfg.SessionTTL
	}
	return defaultSessionTTL
}

func (s *Service) sessionCleanupInterval() time.Duration {
	if s != nil {
		return s.cfg.SessionCleanupInterval
	}
	return 0
}

func (s *Service) sessionFromRequest(ctx context.Context, r *http.Request) (Session, error) {
	cookie, err := r.Cookie(s.sessionCookieName())
	if err != nil {
		return Session{}, unauthorizedError("session cookie is required")
	}
	return s.repo.GetSessionByTokenHash(ctx, hashToken(cookie.Value))
}

func (s *Service) cleanupExpiredSessionsIfDue(ctx context.Context) {
	interval := s.sessionCleanupInterval()
	if interval <= 0 || s == nil || s.repo == nil {
		return
	}
	now := time.Now().UTC()
	s.cleanupMu.Lock()
	if !s.lastCleanupAt.IsZero() && now.Sub(s.lastCleanupAt) < interval {
		s.cleanupMu.Unlock()
		return
	}
	s.lastCleanupAt = now
	s.cleanupMu.Unlock()
	_, _ = s.repo.DeleteExpiredSessions(ctx, now)
}

func csrfProtectedMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func HashPassword(password string) (string, error) {
	password = strings.TrimSpace(password)
	if password == "" {
		return "", errors.New("password is required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func newSessionToken() (string, string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(raw[:])
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func validationError(message string) error {
	return &teaching.Error{Kind: teaching.KindValidation, Code: "validation_error", Message: message}
}

func unauthorizedError(message string) error {
	return &teaching.Error{Kind: teaching.KindUnauthorized, Code: "unauthorized", Message: message}
}

func forbiddenError(message string) error {
	return &teaching.Error{Kind: teaching.KindForbidden, Code: "forbidden", Message: message}
}

func conflictError(message string) error {
	return &teaching.Error{Kind: teaching.KindConflict, Code: "conflict", Message: message}
}

func unavailableError(message string, err error) error {
	return &teaching.Error{Kind: teaching.KindUnavailable, Code: "service_unavailable", Message: message, Err: err}
}

func notFoundError(message string) error {
	return &teaching.Error{Kind: teaching.KindNotFound, Code: "not_found", Message: message}
}
