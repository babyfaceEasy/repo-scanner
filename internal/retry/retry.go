package retry

import (
	"math/rand"
	"time"

	"github.com/babyfaceeasy/repo-scanner/internal/github"
	"github.com/babyfaceeasy/repo-scanner/pkg/logger"
)

// Retrier decorates a GithubClient with a retry logic
type Retrier struct {
	client     github.GitHubClient
	logger     logger.Logger
	maxRetries int
	baseDelay  time.Duration
}

// NewRetirer creates a new Retrier Decorator
func NewRetrier(client github.GitHubClient, logger logger.Logger, maxRetries int, baseDelay time.Duration) *Retrier {
	return &Retrier{
		client:     client,
		logger:     logger,
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
	}
}

// DownloadRepo implements GitHubClient with retry logic
func (r *Retrier) DownloadRepo(cloneURL, destDir string) error {
	var lastErr error
	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		err := r.client.DownloadRepo(cloneURL, destDir)
		if err == nil {
			return nil
		}

		lastErr = err
		if !isRetryable(err) {
			r.logger.Warn("Non-retryable error", "error", err, "attempt", attempt+1)
			return err
		}

		delay := r.calculateDelay(attempt)
		r.logger.Info("Retrying after delay", "attempt", attempt+1, "delay_ms", delay.Milliseconds(), "error", err)
		time.Sleep(delay)
	}

	r.logger.Error("Max retries exceeded", "error", lastErr, "max_retries", r.maxRetries)
	return lastErr
}

// calculateDelay computes exponential backoff with jitter
func (r *Retrier) calculateDelay(attempt int) time.Duration {
	delay := r.baseDelay * time.Duration(1<<attempt)
	jitter := time.Duration(rand.Intn(100)) * time.Millisecond

	return delay + jitter
}

// isRetryable checks if an error is retryable majorly for rate limit and network errors
func isRetryable(err error) bool {
	// TODO: check for HTTP 429 or specific network errors
	return true
}
