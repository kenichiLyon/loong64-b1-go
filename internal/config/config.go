package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kenichiLyon/loong64-b1-go/internal/runtimecfg"
)

// Config contains process-level settings loaded from environment variables.
type Config struct {
	HTTPAddr                  string
	AppEnv                    string
	StorageRoot               string
	RuntimeConfigPath         string
	RuntimeConfigExists       bool
	RuntimeConfigError        string
	DBDriver                  string
	DatabaseURL               string
	SQLitePath                string
	LLMBaseURL                string
	LLMModel                  string
	LLMAPIKey                 string
	LLMTimeout                time.Duration
	MigrationsDir             string
	AutoMigrate               bool
	DevAuthBypass             bool
	SessionCookieName         string
	CSRFCookieName            string
	SessionTTL                time.Duration
	SessionSecureCookie       bool
	MaxUploadBytes            int64
	MaxArtifactsPerSubmission int
	DBMaxConns                int32
	ReadHeaderTimeout         time.Duration
	ShutdownTimeout           time.Duration
	ReadyTimeout              time.Duration
}

// Load returns configuration with safe local-development defaults.
func Load() Config {
	...existing body...
		SessionCookieName:         getenv("SESSION_COOKIE_NAME", "loong64_b1_session"),
		CSRFCookieName:            getenv("CSRF_COOKIE_NAME", "loong64_b1_csrf"),
		SessionTTL:                durationFromEnv("SESSION_TTL", 168*time.Hour),
		SessionSecureCookie:       boolFromEnv("SESSION_SECURE_COOKIE", getenv("APP_ENV", "development") == "production"),
	...existing body...
}
