package retry

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/babyfaceeasy/repo-scanner/internal/github"
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

func TestRetrier_DownloadRepo(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(*int) func(string, string) error
		wantErr       bool
		wantAttempts  int
		expectLogPart string
	}{
		{
			name: "Immediate success",
			setupFunc: func(attempts *int) func(string, string) error {
				return func(cloneURL, destDir string) error {
					*attempts++
					return nil
				}
			},
			wantErr:       false,
			wantAttempts:  1,
			expectLogPart: "Download succeeded",
		},
		{
			name: "Success after retries",
			setupFunc: func(attempts *int) func(string, string) error {
				return func(cloneURL, destDir string) error {
					*attempts++
					if *attempts < 3 {
						return &github.RetryAfterError{
							Err:        errors.New("rate limit"),
							RetryAfter: 10 * time.Millisecond,
						}
					}
					return nil
				}
			},
			wantErr:       false,
			wantAttempts:  3,
			expectLogPart: "Retrying after delay",
		},
		{
			name: "Failure after max retries",
			setupFunc: func(attempts *int) func(string, string) error {
				return func(cloneURL, destDir string) error {
					*attempts++
					return &github.RetryAfterError{
						Err:        errors.New("rate limit"),
						RetryAfter: 10 * time.Millisecond,
					}
				}
			},
			wantErr:       true,
			wantAttempts:  3,
			expectLogPart: "Retrying after delay",
		},
		{
			name: "Non-retryable error",
			setupFunc: func(attempts *int) func(string, string) error {
				return func(cloneURL, destDir string) error {
					*attempts++
					return errors.New("fatal config error")
				}
			},
			wantErr:       true,
			wantAttempts:  1,
			expectLogPart: "Non-retryable error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var attempts int
			mockLog := &mockLogger{}
			mockClient := &mockGitHubClient{}

			mockClient.downloadFunc = tt.setupFunc(&attempts)

			retrier := NewRetrier(mockClient, mockLog, 3, 10*time.Millisecond, 1*time.Second)

			err := retrier.DownloadRepo("https://github.com/owner/repo.git", "some/dest")
			if (err != nil) != tt.wantErr {
				t.Errorf("DownloadRepo() error = %v, wantErr %v", err, tt.wantErr)
			}

			if attempts != tt.wantAttempts {
				t.Errorf("attempts = %d, want %d", attempts, tt.wantAttempts)
			}

			if !containsLog(mockLog.logs, tt.expectLogPart) {
				t.Errorf("Expected log containing %q, got logs: %v", tt.expectLogPart, mockLog.logs)
			}
		})
	}
}

func containsLog(logs []string, substr string) bool {
	for _, log := range logs {
		if strings.Contains(log, substr) {
			return true
		}
	}
	return false
}
