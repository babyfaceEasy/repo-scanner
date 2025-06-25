package model

import (
	"fmt"
	"strings"
)

// Config represents the input JSON structure
type Config struct {
	CloneURL string  `json:"clone_url"`
	Size     float64 `json:"size"` // Size threshold in MB
}

// Validate validates the Config struct
func (c *Config) Validate() error {
	if c.CloneURL == "" {
		return fmt.Errorf("clone_url is required")
	}
	if !strings.HasPrefix(c.CloneURL, "https://github.com/") {
		return fmt.Errorf("clone_url must be a valid GitHub HTTPS URL")
	}
	if c.Size <= 0 {
		return fmt.Errorf("size must be positive")
	}
	return nil
}

// FileInfo represents a file exceeding the size threshold
type FileInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"` // Size in bytes
}

// Output represents the output JSON structure
type Output struct {
	Total int        `json:"total"`
	Files []FileInfo `json:"files"`
}
