package retry

import (
	"errors"
	"testing"
	"time"
)

type mockGitHubClient struct {
	downloadFunc func(cloneURL, destDir string) error
	attempts     int
}

func (m *mockGitHubClient) DownloadRepo(cloneURL, destDir string) error {
	m.attempts++
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

func TestRetrier(t *testing.T) {
	t.Run("Success on first attempt", func(t *testing.T) {
		mockClient := &mockGitHubClient{
			downloadFunc: func(cloneURL, destDir string) error { return nil },
		}
		mockLog := &mockLogger{}
		retrier := NewRetrier(mockClient, mockLog, 3, 10*time.Millisecond, 1*time.Second)

		err := retrier.DownloadRepo("url", "dir")
		if err != nil {
			t.Errorf("DownloadRepo() error = %v, want nil", err)
		}
		if mockClient.attempts != 1 {
			t.Errorf("Attempts = %d, want 1", mockClient.attempts)
		}
	})

	t.Run("Success after retries", func(t *testing.T) {
		attempts := 0
		mockClient := &mockGitHubClient{
			downloadFunc: func(cloneURL, destDir string) error {
				if attempts < 2 {
					attempts++
					return errors.New("rate limit")
				}
				return nil
			},
		}
		mockLog := &mockLogger{}
		retrier := NewRetrier(mockClient, mockLog, 3, 10*time.Millisecond, 1*time.Second)

		err := retrier.DownloadRepo("url", "dir")
		if err != nil {
			t.Errorf("DownloadRepo() error = %v, want nil", err)
		}
		if mockClient.attempts != 3 {
			t.Errorf("Attempts = %d, want 3", mockClient.attempts)
		}
		if len(mockLog.logs) < 2 || mockLog.logs[0] != "Retrying after delay" {
			t.Errorf("Logs = %v, want at least two 'Retrying after delay' entries", mockLog.logs)
		}
	})

	t.Run("Max retries exceeded", func(t *testing.T) {
		mockClient := &mockGitHubClient{
			downloadFunc: func(cloneURL, destDir string) error { return errors.New("rate limit") },
		}
		mockLog := &mockLogger{}
		retrier := NewRetrier(mockClient, mockLog, 2, 10*time.Millisecond, 1*time.Second)

		err := retrier.DownloadRepo("url", "dir")
		if err == nil || err.Error() != "rate limit" {
			t.Errorf("DownloadRepo() error = %v, want rate limit", err)
		}
		if mockClient.attempts != 3 {
			t.Errorf("Attempts = %d, want 3", mockClient.attempts)
		}

		if !containsLog(mockLog.logs, "Max retries exceeded") {
			t.Errorf("Logs = %v, want 'Max retries exceeded'", mockLog.logs)
		}
	})
}


func containsLog(logs []string, want string) bool {
	for _, l := range logs {
		if l == want {
			return true
		}
	}

	return false
}