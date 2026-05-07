package authn

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
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
}

type Service struct {
	repo Repository
	cfg  config.Config
}

func NewService(repo Repository, cfg config.Config) *Service {
	return &Service{repo: repo, cfg: cfg}
}

func (s *Service) Login(ctx context.Context, username, password string) (Session, string, error) {
	if s == nil || s.repo == nil {
		return Session{}, "", unavailableError("auth service is not configured", nil)
	}
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
		ExpiresAt:  now.Add(s.cfg.SessionTTL),
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
	cookie, err := r.Cookie(s.cfg.SessionCookieName)
	if err != nil {
		return teaching.Actor{}, unauthorizedError("session cookie is required")
	}
	session, err := s.repo.GetSessionByTokenHash(ctx, hashToken(cookie.Value))
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
	cookie, err := r.Cookie(s.cfg.SessionCookieName)
	if err != nil {
		return nil
	}
	return s.repo.DeleteSessionByTokenHash(ctx, hashToken(cookie.Value))
}

func (s *Service) WriteSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SessionSecureCookie,
		Expires:  time.Now().UTC().Add(s.cfg.SessionTTL),
		MaxAge:   int(s.cfg.SessionTTL.Seconds()),
	})
}

func (s *Service) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.SessionSecureCookie,
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
	})
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
