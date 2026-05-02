package config

import (
	"os"
	"strconv"
	"time"
)

// Config contains process-level settings loaded from environment variables.
type Config struct {
	HTTPAddr                  string
	AppEnv                    string
	StorageRoot               string
	DatabaseURL               string
	LLMBaseURL                string
	LLMModel                  string
	LLMAPIKey                 string
	LLMTimeout                time.Duration
	MigrationsDir             string
	DevAuthBypass             bool
	MaxUploadBytes            int64
	MaxArtifactsPerSubmission int
	DBMaxConns                int32
	ReadHeaderTimeout         time.Duration
	ShutdownTimeout           time.Duration
	ReadyTimeout              time.Duration
}

// Load returns configuration with safe local-development defaults.
func Load() Config {
	return Config{
		HTTPAddr:                  getenv("HTTP_ADDR", "127.0.0.1:8080"),
		AppEnv:                    getenv("APP_ENV", "development"),
		StorageRoot:               getenv("STORAGE_ROOT", "./storage"),
		DatabaseURL:               getenv("DATABASE_URL", ""),
		LLMBaseURL:                getenv("LLM_BASE_URL", ""),
		LLMModel:                  getenv("LLM_MODEL", ""),
		LLMAPIKey:                 getenv("LLM_API_KEY", ""),
		LLMTimeout:                durationFromEnv("LLM_TIMEOUT", 30*time.Second),
		MigrationsDir:             getenv("MIGRATIONS_DIR", "migrations"),
		DevAuthBypass:             boolFromEnv("DEV_AUTH_BYPASS", false),
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
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
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
