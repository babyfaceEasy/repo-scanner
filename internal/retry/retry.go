package retry

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"syscall"
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
	maxDelay   time.Duration
}

// NewRetirer creates a new Retrier Decorator
func NewRetrier(client github.GitHubClient, logger logger.Logger, maxRetries int, baseDelay, maxDelay time.Duration) *Retrier {
	return &Retrier{
		client:     client,
		logger:     logger,
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
		maxDelay:   maxDelay,
	}
}

// DownloadRepo implements GitHubClient with retry logic
func (r *Retrier) DownloadRepoOLD(cloneURL, destDir string) error {
	var lastErr error
	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		err := r.client.DownloadRepo(cloneURL, destDir)
		if err == nil {
			r.logger.Info("Download succeeded")
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

// DownloadRepo implements GitHubClient with retry logic
func (r *Retrier) DownloadRepo(cloneURL, destDir string) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			r.logger.Error("Recovered from panic", "panic", rec)
			err = fmt.Errorf("panic recovered: %v", rec)
		}
	}()

	var lastErr error
	for attempt := 0; attempt < r.maxRetries; attempt++ {
		err := r.client.DownloadRepo(cloneURL, destDir)
		if err == nil {
			r.logger.Info("Download succeeded", "attempt", attempt+1)
			return nil
		}

		lastErr = err

		if !isRetryable(err) {
			r.logger.Warn("Non-retryable error", "error", err, "attempt", attempt+1)
			return err
		}

		if attempt == r.maxRetries {
			break
		}

		var delay time.Duration
		if raErr, ok := err.(*github.RetryAfterError); ok {
			delay = raErr.RetryAfter
			r.logger.Warn("Received 429 Too Many Requests", "retry_after_sec", delay.Seconds())
		} else {
			delay = r.calculateDelay(attempt)
		}

		r.logger.Info("Retrying after delay", "attempt", attempt+1, "delay_ms", delay.Milliseconds(), "error", err)
		time.Sleep(delay)
	}

	r.logger.Error("Max retries exceeded", "error", lastErr, "max_retries", r.maxRetries)
	return lastErr
}

// calculateDelay computes exponential backoff with jitter
func (r *Retrier) calculateDelay(attempt int) time.Duration {
	delay := r.baseDelay * time.Duration(1<<attempt)
	jitter := time.Duration(rand.Intn(250)) * time.Millisecond
	final := delay + jitter

	if final > r.maxDelay {
		return r.maxDelay
	}

	return final
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// RetryAfterError from 429
	var raErr *github.RetryAfterError
	if errors.As(err, &raErr) {
		return true
	}

	// net.Error (timeouts)
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	/*
		// if underlying error is net.Error with Timeout
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			if nErr, ok := urlErr.Err.(net.Error); ok && nErr.Timeout() {
				return true
			}
		}
	*/

	// check for common retryable error substrings
	errStr := strings.ToLower(err.Error())
	retryableSubstrings := []string{
		"connection reset",
		"connection refused",
		"timeout",
		"temporary failure",
		"tls handshake timeout",
		"eof",
		"no such host",
		"broken pipe",
		"i/o timeout",
	}

	for _, substr := range retryableSubstrings {
		if strings.Contains(errStr, substr) {
			return true
		}
	}

	// check for retryable syscall errors
	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case syscall.ECONNRESET, syscall.ETIMEDOUT, syscall.ECONNREFUSED, syscall.EPIPE:
			return true
		}
	}

	return false
}
