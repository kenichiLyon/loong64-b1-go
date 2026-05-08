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
	SessionRefreshInterval    time.Duration
	SessionCleanupInterval    time.Duration
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
	runtimeConfigPath := getenv("RUNTIME_CONFIG_PATH", "./config/runtime.json")
	fileCfg, exists, loadErr := runtimecfg.New(runtimeConfigPath).Load()

	databaseURL := getenv("DATABASE_URL", "")
	dbDriver := strings.ToLower(strings.TrimSpace(getenv("DB_DRIVER", "")))
	sqlitePath := getenv("SQLITE_PATH", "")
	if dbDriver == "" {
		dbDriver = strings.ToLower(strings.TrimSpace(fileCfg.DBDriver))
	}
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(fileCfg.DatabaseURL)
	}
	if sqlitePath == "" {
		sqlitePath = strings.TrimSpace(fileCfg.SQLitePath)
	}
	if dbDriver == "" {
		if databaseURL != "" {
			dbDriver = "postgres"
		} else {
			dbDriver = "sqlite"
		}
	}
	if sqlitePath == "" {
		sqlitePath = "./data/loong64-b1-go.db"
	}
	autoMigrateDefault := dbDriver == "sqlite"
	autoMigrate := autoMigrateDefault
	if fileCfg.AutoMigrate != nil {
		autoMigrate = *fileCfg.AutoMigrate
	}
	if envAutoMigrate, ok := boolFromEnvWithSet("AUTO_MIGRATE"); ok {
		autoMigrate = envAutoMigrate
	}
	return Config{
		HTTPAddr:                  getenv("HTTP_ADDR", "127.0.0.1:8080"),
		AppEnv:                    getenv("APP_ENV", "development"),
		StorageRoot:               getenv("STORAGE_ROOT", "./storage"),
		RuntimeConfigPath:         runtimeConfigPath,
		RuntimeConfigExists:       exists,
		RuntimeConfigError:        errorString(loadErr),
		DBDriver:                  dbDriver,
		DatabaseURL:               databaseURL,
		SQLitePath:                sqlitePath,
		LLMBaseURL:                getenv("LLM_BASE_URL", ""),
		LLMModel:                  getenv("LLM_MODEL", ""),
		LLMAPIKey:                 getenv("LLM_API_KEY", ""),
		LLMTimeout:                durationFromEnv("LLM_TIMEOUT", 30*time.Second),
		MigrationsDir:             getenv("MIGRATIONS_DIR", "migrations"),
		AutoMigrate:               autoMigrate,
		DevAuthBypass:             boolFromEnv("DEV_AUTH_BYPASS", false),
		SessionCookieName:         getenv("SESSION_COOKIE_NAME", "loong64_b1_session"),
		CSRFCookieName:            getenv("CSRF_COOKIE_NAME", "loong64_b1_csrf"),
		SessionTTL:                durationFromEnv("SESSION_TTL", 168*time.Hour),
		SessionRefreshInterval:    durationFromEnv("SESSION_REFRESH_INTERVAL", 15*time.Minute),
		SessionCleanupInterval:    durationFromEnv("SESSION_CLEANUP_INTERVAL", time.Hour),
		SessionSecureCookie:       boolFromEnv("SESSION_SECURE_COOKIE", getenv("APP_ENV", "development") == "production"),
		MaxUploadBytes:            int64FromEnv("MAX_UPLOAD_BYTES", 50*1024*1024),
		MaxArtifactsPerSubmission: intFromEnv("MAX_ARTIFACTS_PER_SUBMISSION", 20),
		DBMaxConns:                int32FromEnv("DB_MAX_CONNS", 10),
		ReadHeaderTimeout:         durationFromEnv("HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
		ShutdownTimeout:           durationFromEnv("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		ReadyTimeout:              durationFromEnv("READY_TIMEOUT", 2*time.Second),
	}
}

func intFromEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func int64FromEnv(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func boolFromEnv(key string, fallback bool) bool {
	parsed, ok := boolFromEnvWithSet(key)
	if !ok {
		return fallback
	}
	return parsed
}

func boolFromEnvWithSet(key string) (bool, bool) {
	value := os.Getenv(key)
	if value == "" {
		return false, false
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, false
	}
	return parsed, true
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func int32FromEnv(key string, fallback int32) int32 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return int32(parsed)
}

func durationFromEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
