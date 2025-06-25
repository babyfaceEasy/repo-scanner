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
			Mode: 0644,
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

	// Verify logs
	expectedLogs := []string{
		"Converted clone URL",
		"Fetched tarball",
		"Repository extracted",
	}
	for i, log := range expectedLogs {
		if i >= len(mockLog.logs) || mockLog.logs[i] != log {
			t.Errorf("Log %d = %v, want %v", i, mockLog.logs[i], log)
		}
	}
}
