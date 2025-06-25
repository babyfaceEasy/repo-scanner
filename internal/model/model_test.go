package model

import (
	"encoding/json"
	"testing"
)

func TestConfigSerialization(t *testing.T) {
	cfg := &Config{
		CloneURL: "https://github.com/owner/repo.git",
		Size:     1.0,
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal Config: %v", err)
	}

	var got Config
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Failed to unmarshal Config: %v", err)
	}

	if got.CloneURL != cfg.CloneURL || got.Size != cfg.Size {
		t.Errorf("Got = %v, want %v", got, cfg)
	}
}

func TestOutputSerialization(t *testing.T) {
	out := &Output{
		Total: 1,
		Files: []FileInfo{
			{Name: "large.txt", Size: 2000},
		},
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("Failed to marshal Output: %v", err)
	}

	var got Output
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Failed to unmarshal Output: %v", err)
	}

	if got.Total != out.Total || len(got.Files) != len(out.Files) {
		t.Errorf("Got = %v, want %v", got, out)
	}
	if got.Files[0].Name != out.Files[0].Name || got.Files[0].Size != out.Files[0].Size {
		t.Errorf("Got Files[0] = %v, want %v", got.Files[0], out.Files[0])
	}
}
