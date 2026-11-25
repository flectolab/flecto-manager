package cdn

import (
	"errors"
	"fmt"
	"time"
)

// TemporaryError marks a retryable failure (rate limit, timeout, outage). Errors
// that would fail identically on every attempt are returned bare.
type TemporaryError struct {
	// RetryAfter is the delay asked by the provider; zero leaves it to the caller's backoff.
	RetryAfter time.Duration
	Err        error
}

func (e *TemporaryError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("%v (retry after %s)", e.Err, e.RetryAfter)
	}
	return e.Err.Error()
}

func (e *TemporaryError) Unwrap() error {
	return e.Err
}

// Temporary wraps an error as retryable.
func Temporary(err error, retryAfter time.Duration) error {
	return &TemporaryError{RetryAfter: retryAfter, Err: err}
}

// AsTemporary reports whether the error is retryable, and the delay asked.
func AsTemporary(err error) (time.Duration, bool) {
	var temporary *TemporaryError
	if errors.As(err, &temporary) {
		return temporary.RetryAfter, true
	}
	return 0, false
}
