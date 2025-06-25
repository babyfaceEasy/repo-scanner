package service

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/babyfaceeasy/repo-scanner/internal/config"
	"github.com/babyfaceeasy/repo-scanner/internal/model"
	"github.com/babyfaceeasy/repo-scanner/internal/output"
	"github.com/babyfaceeasy/repo-scanner/internal/scanner"
)

type mockGitHubClient struct {
	downloadFunc func(cloneURL, destDir string) error
}

func (m *mockGitHubClient) DownloadRepo(cloneURL, destDir string) error {
	return m.downloadFunc(cloneURL, destDir)
}

type mockLogger struct {
	logs []string
}

func (m *mockLogger) Info(msg string, fields ...interface{})  { m.logs = append(m.logs, msg) }
func (m *mockLogger) Error(msg string, fields ...interface{}) { m.logs = append(m.logs, msg) }
func (m *mockLogger) Warn(msg string, fields ...interface{})  { m.logs = append(m.logs, msg) }
func (m *mockLogger) Debug(msg string, fields ...interface{}) { m.logs = append(m.logs, msg) }
func (m *mockLogger) Fatal(msg string, fields ...interface{}) { m.logs = append(m.logs, msg) }

func TestScan(t *testing.T) {
	mockLog := &mockLogger{}
	tmpDir := t.TempDir()
	createFile(t, filepath.Join(tmpDir, "large.txt"), 2000)

	mockGH := &mockGitHubClient{
		downloadFunc: func(cloneURL, destDir string) error {
			return os.Rename(tmpDir, destDir)
		},
	}

	svc := New(
		config.New(),
		mockGH,
		scanner.New(mockLog),
		output.New(),
		mockLog,
	)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	input := `{"clone_url":"https://github.com/owner/repo.git","size":1.0}`
	err := svc.Scan(input)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	buf.ReadFrom(r)

	var got model.Output
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Failed to parse output JSON: %v", err)
	}

	if got.Total != 1 {
		t.Errorf("Output.Total = %d, want 1", got.Total)
	}
	if len(got.Files) != 1 || got.Files[0].Name != "large.txt" || got.Files[0].Size != 2000 {
		t.Errorf("Output.Files = %v, want [{Name:large.txt Size:2000}]", got.Files)
	}

	// Verify logs
	expectedLogs := []string{
		"Config parsed",
		"Created temp dir",
		"Repository downloaded",
		"Found large file",
		"File scan completed",
	}
	for i, log := range expectedLogs {
		if i >= len(mockLog.logs) || mockLog.logs[i] != log {
			t.Errorf("Log %d = %v, want %v", i, mockLog.logs[i], log)
		}
	}
}

func createFile(t *testing.T, path string, size int64) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()
	if err := f.Truncate(size); err != nil {
		t.Fatalf("Failed to set file size: %v", err)
	}
}
