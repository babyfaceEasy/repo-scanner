package github

import (
	"archive/tar"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type mockLogger struct {
	logs []string
}

func (m *mockLogger) Info(msg string, fields ...interface{})  { m.logs = append(m.logs, msg) }
func (m *mockLogger) Error(msg string, fields ...interface{}) { m.logs = append(m.logs, msg) }
func (m *mockLogger) Warn(msg string, fields ...interface{})  { m.logs = append(m.logs, msg) }
func (m *mockLogger) Debug(msg string, fields ...interface{}) { m.logs = append(m.logs, msg) }
func (m *mockLogger) Fatal(msg string, fields ...interface{}) { m.logs = append(m.logs, msg) }

func TestCloneURLToTarballURL(t *testing.T) {
	tests := []struct {
		name     string
		cloneURL string
		wantURL  string
		wantErr  bool
	}{
		{
			name:     "valid clone URL",
			cloneURL: "https://github.com/owner/repo.git",
			wantURL:  "https://api.github.com/repos/owner/repo/tarball",
		},
		{
			name:     "clone URL without .git",
			cloneURL: "https://github.com/owner/repo",
			wantURL:  "https://api.github.com/repos/owner/repo/tarball",
		},
		{
			name:     "invalid clone URL",
			cloneURL: "https://gitlab.com/owner/repo.git",
			wantErr:  true,
		},
		{
			name:     "empty path",
			cloneURL: "https://github.com/",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cloneURLToTarballURL(tt.cloneURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("cloneURLToTarballURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantURL {
				t.Errorf("cloneURLToTarballURL() = %v, want %v", got, tt.wantURL)
			}
		})
	}
}

func TestDownloadRepo(t *testing.T) {
	mockLog := &mockLogger{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/repos/owner/repo/tarball") {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		gzw := gzip.NewWriter(w)
		tw := tar.NewWriter(gzw)
		hdr := &tar.Header{
			Name: "repo/file.txt",
			Mode: 0o644,
			Size: int64(len("test content")),
		}
		tw.WriteHeader(hdr)
		tw.Write([]byte("test content"))
		tw.Close()
		gzw.Close()
	}))
	defer server.Close()

	client := NewClient("test-token", mockLog)
	originalCloneURLToTarballURL := client.cloneURLToTarballURL
	client.cloneURLToTarballURL = func(cloneURL string) (string, error) {
		return server.URL + "/repos/owner/repo/tarball", nil
	}
	defer func() { client.cloneURLToTarballURL = originalCloneURLToTarballURL }()

	tmpDir := t.TempDir()
	err := client.DownloadRepo("https://github.com/owner/repo.git", tmpDir)
	if err != nil {
		t.Fatalf("DownloadRepo() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "file.txt"))
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("File content = %v, want %v", string(content), "test content")
	}

	// verify logs
	t.Logf("Captured logs: %+v", mockLog.logs)

	expectedLogs := []string{
		"Converted clone URL",
		"tarball stream got successfully",
		"about to call extract tarball",
		"Repository extracted",
	}

	for _, expected := range expectedLogs {
		found := false
		for _, actual := range mockLog.logs {
			if strings.Contains(actual, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected log message containing %q not found in logs: %v", expected, mockLog.logs)
		}
	}
}

func TestDownloadRepo_WithMultipleFiles(t *testing.T) {
	mockLog := &mockLogger{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		gzw := gzip.NewWriter(w)
		tw := tar.NewWriter(gzw)

		files := map[string]string{
			"repo/file1.txt":     "hello world",
			"repo/dir/file2.txt": "another file",
		}

		for name, content := range files {
			hdr := &tar.Header{
				Name: name,
				Mode: 0o644,
				Size: int64(len(content)),
			}
			tw.WriteHeader(hdr)
			tw.Write([]byte(content))
		}

		tw.Close()
		gzw.Close()
	}))
	defer server.Close()

	client := NewClient("test-token", mockLog)
	client.cloneURLToTarballURL = func(_ string) (string, error) {
		return server.URL + "/repos/owner/repo/tarball", nil
	}

	tmpDir := t.TempDir()
	err := client.DownloadRepo("https://github.com/owner/repo.git", tmpDir)
	if err != nil {
		t.Fatalf("DownloadRepo() error: %v", err)
	}

	assertFileContent(t, filepath.Join(tmpDir, "file1.txt"), "hello world")
	assertFileContent(t, filepath.Join(tmpDir, "dir/file2.txt"), "another file")
}

func TestDownloadRepo_Security_PathTraversal(t *testing.T) {
	mockLog := &mockLogger{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		gzw := gzip.NewWriter(w)
		tw := tar.NewWriter(gzw)

		// Dangerous path
		hdr := &tar.Header{
			Name: "repo/../malicious.txt",
			Mode: 0o644,
			Size: int64(len("malicious content")),
		}
		tw.WriteHeader(hdr)
		tw.Write([]byte("malicious content"))

		tw.Close()
		gzw.Close()
	}))
	defer server.Close()

	client := NewClient("test-token", mockLog)
	client.cloneURLToTarballURL = func(_ string) (string, error) {
		return server.URL + "/repos/owner/repo/tarball", nil
	}

	tmpDir := t.TempDir()
	err := client.DownloadRepo("https://github.com/owner/repo.git", tmpDir)
	if err == nil || !strings.Contains(err.Error(), "illegal file path") {
		t.Fatalf("expected path traversal error, got: %v", err)
	}
}

func TestDownloadRepo_429RetryAfter(t *testing.T) {
	mockLog := &mockLogger{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Header().Set("Retry-After", "3")
	}))
	defer server.Close()

	client := NewClient("test-token", mockLog)
	client.cloneURLToTarballURL = func(_ string) (string, error) {
		return server.URL + "/repos/owner/repo/tarball", nil
	}

	tmpDir := t.TempDir()
	err := client.DownloadRepo("https://github.com/owner/repo.git", tmpDir)
	if err == nil {
		t.Fatal("expected RetryAfterError, got nil")
	}

	retryErr, ok := err.(*RetryAfterError)
	if !ok {
		t.Fatalf("expected RetryAfterError, got %T", err)
	}

	if retryErr.RetryAfter != 3*time.Second {
		t.Errorf("expected RetryAfter = 3s, got %v", retryErr.RetryAfter)
	}
}

func assertFileContent(t *testing.T, path string, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	if string(data) != expected {
		t.Errorf("File content at %s = %q, want %q", path, data, expected)
	}
}
