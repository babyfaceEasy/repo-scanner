package env

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds environment variables
type Config struct {
	GitHubToken string
	LogEnv      string
}

// Load and validates environment variables
func Load() (*Config, error) {

	
	path := os.Getenv("GODOTENV_PATH")
	if path == "" {
		path = ".env"
	}

	if err := godotenv.Load(path); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading .env file: %w", err)
	}

	cfg := &Config{
		GitHubToken: os.Getenv("GITHUB_TOKEN"),
		LogEnv:      os.Getenv("LOG_ENV"),
	}

	if cfg.GitHubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN is required")
	}

	// set production as the default environment
	if cfg.LogEnv == "" {
		cfg.LogEnv = "production"
	}

	return cfg, nil
}
