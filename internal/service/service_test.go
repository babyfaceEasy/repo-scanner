package service

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/babyfaceeasy/repo-scanner/internal/config"
	"github.com/babyfaceeasy/repo-scanner/internal/model"
	"github.com/babyfaceeasy/repo-scanner/internal/output"
	"github.com/babyfaceeasy/repo-scanner/internal/retry"
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

	info, _ := os.Stat(filepath.Join(tmpDir, "large.txt"))
	t.Logf("DEBUG: Created file size = %d", info.Size())

	mockGH := &mockGitHubClient{
		downloadFunc: func(cloneURL, destDir string) error {
			err := copyDir(tmpDir, destDir)

			files, _ := os.ReadDir(destDir)
			for _, f := range files {
				t.Logf("DEBUG: Copied file = %s", f.Name())
			}

			return err
			// return os.Rename(tmpDir, destDir)
		},
	}
	retryGH := retry.NewRetrier(mockGH, mockLog, 3, 10*time.Millisecond)

	svc := New(
		config.New(),
		retryGH,
		scanner.New(mockLog),
		output.New(),
		mockLog,
	)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	input := `{"clone_url":"https://github.com/owner/repo.git","size":0.001}`
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

	t.Logf("Captured logs: %+v", mockLog.logs)

	expectedLogs := []string{
		"Config parsed",
		"Created temp dir",
		"Repository downloaded",
		"Found large file",
		"File scan completed",
	}

	for _, expected := range expectedLogs {
		found := false
		for _, actual := range mockLog.logs {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected log message %q not found in logs: %v", expected, mockLog.logs)
		}
	}

	/*
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
	*/
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

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		
		return os.WriteFile(target, data, info.Mode())
	})
}
