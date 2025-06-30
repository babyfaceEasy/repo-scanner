package github

import "time"

// RetryAfterError represents a 429 rate-limit response with retry time.
type RetryAfterError struct {
	Err        error
	RetryAfter time.Duration
}

func (e *RetryAfterError) Error() string {
	return e.Err.Error()
}

func (e *RetryAfterError) Unwrap() error {
	return e.Err
}
