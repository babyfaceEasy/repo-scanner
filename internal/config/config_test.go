package config

import (
	"testing"

	"github.com/babyfaceeasy/repo-scanner/internal/model"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		expected *model.Config
	}{
		{
			name:  "valid config",
			input: `{"clone_url":"https://github.com/owner/repo.git","size":1.0}`,
			expected: &model.Config{
				CloneURL: "https://github.com/owner/repo.git",
				Size:     1.0,
			},
		},
		{
			name:    "empty clone_url",
			input:   `{"clone_url":"","size":1.0}`,
			wantErr: true,
		},
		{
			name:    "invalid clone_url",
			input:   `{"clone_url":"https://gitlab.com/owner/repo.git","size":1.0}`,
			wantErr: true,
		},
		{
			name:    "negative size",
			input:   `{"clone_url":"https://github.com/owner/repo.git","size":-1.0}`,
			wantErr: true,
		},
	}

	parser := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parser.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && cfg != nil {
				if cfg.CloneURL != tt.expected.CloneURL || cfg.Size != tt.expected.Size {
					t.Errorf("Parse() = %v, want %v", cfg, tt.expected)
				}
			}
		})
	}
}
