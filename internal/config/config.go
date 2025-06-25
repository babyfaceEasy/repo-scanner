package config

import (
	"encoding/json"
	"fmt"

	"github.com/babyfaceeasy/repo-scanner/internal/model"
)

// ConfigParser handles configuration parsing
type ConfigParser struct{}

// New creates a new ConfigParser
func New() *ConfigParser {
	return &ConfigParser{}
}

// Parse parses JSON input into a Config struct
func (p *ConfigParser) Parse(jsonStr string) (*model.Config, error) {
	var cfg model.Config
	if err := json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}
