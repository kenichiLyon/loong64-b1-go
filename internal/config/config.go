package config

import "os"

// Config contains process-level settings loaded from environment variables.
type Config struct {
	HTTPAddr    string
	AppEnv      string
	StorageRoot string
	DatabaseURL string
	LLMBaseURL  string
	LLMModel    string
}

// Load returns configuration with safe local-development defaults.
func Load() Config {
	return Config{
		HTTPAddr:    getenv("HTTP_ADDR", "127.0.0.1:8080"),
		AppEnv:      getenv("APP_ENV", "development"),
		StorageRoot: getenv("STORAGE_ROOT", "./storage"),
		DatabaseURL: getenv("DATABASE_URL", ""),
		LLMBaseURL:  getenv("LLM_BASE_URL", ""),
		LLMModel:    getenv("LLM_MODEL", ""),
	}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
