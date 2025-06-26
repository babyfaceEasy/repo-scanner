package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/babyfaceeasy/repo-scanner/internal/model"
)

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
	createFile(t, filepath.Join(tmpDir, "small.txt"), 100)
	createFile(t, filepath.Join(tmpDir, "large.txt"), 2000)
	createFile(t, filepath.Join(tmpDir, "sub/dir/file.txt"), 1500)

	scanner := New(mockLog)
	result, err := scanner.Scan(tmpDir, 1000)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if result.Total != 2 {
		t.Errorf("Result.Total = %d, want 2", result.Total)
	}

	expectedFiles := []model.FileInfo{
		{Name: "large.txt", Size: 2000},
		{Name: "sub/dir/file.txt", Size: 1500},
	}
	for i, f := range result.Files {
		if f.Name != expectedFiles[i].Name || f.Size != expectedFiles[i].Size {
			t.Errorf("File %d = %v, want %v", i, f, expectedFiles[i])
		}
	}

	// Verify logs
	var foundCount int
	for _, log := range mockLog.logs {
		if log == "Found large file" {
			foundCount++
		}
	}
	if foundCount != 2 {
		t.Errorf("Expected 2 'Found large file' logs, got %d. All logs: %v", foundCount, mockLog.logs)
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
