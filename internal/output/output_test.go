package output

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/babyfaceeasy/repo-scanner/internal/model"
)

func TestWrite(t *testing.T) {
	result := &model.Output{
		Total: 1,
		Files: []model.FileInfo{
			{Name: "large.txt", Size: 2000},
		},
	}

	// Redirect stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	writer := New()
	err := writer.Write(result)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	buf.ReadFrom(r)

	var got model.Output
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	if got.Total != result.Total || len(got.Files) != len(result.Files) {
		t.Errorf("Got = %v, want %v", got, result)
	}
	if got.Files[0].Name != result.Files[0].Name || got.Files[0].Size != result.Files[0].Size {
		t.Errorf("Got Files[0] = %v, want %v", got.Files[0], result.Files[0])
	}
}
