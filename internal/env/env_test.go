package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// create a temporary .env file
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	content := "GITHUB_TOKEN=ghp_testtoken\nLOG_ENV=development"
	if err := os.WriteFile(envFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	// set env
	os.Setenv("GODOTENV_PATH", envFile)
	defer os.Unsetenv("GODOTENV_PATH")

	// set the current dir to the tmpDir created
	os.Chdir(tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.GitHubToken != "ghp_testtoken" {
		t.Errorf("GitHubToken = %v, want ghp_testtoken", cfg.GitHubToken)
	}
	if cfg.LogEnv != "development" {
		t.Errorf("LogEnv = %v, want development", cfg.LogEnv)
	}
}

func TestLoadNoGitHubToken(t *testing.T) {
	// clear environment
	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("LOG_ENV")

	_, err := Load()
	if err == nil || err.Error() != "GITHUB_TOKEN is required" {
		t.Errorf("Load() error = %v, want GITHUB_TOKEN is required", err)
	}
}

func TestLoadNoEnvFile(t *testing.T) {
	// clear environment
	os.Unsetenv("GODOTENV_PATH")

	// set GITHUB_TOKEN directly
	os.Setenv("GITHUB_TOKEN", "ghp_testtoken")
	defer os.Unsetenv("GITHUB_TOKEN")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.GitHubToken != "ghp_testtoken" {
		t.Errorf("GitHubToken = %v, want ghp_testtoken", cfg.GitHubToken)
	}
	if cfg.LogEnv != "production" {
		t.Errorf("LogEnv = %v, want production", cfg.LogEnv)
	}
}
