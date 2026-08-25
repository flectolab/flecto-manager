package cdn

import (
	"errors"
	"fmt"
	"time"
)

// TemporaryError marks a failure worth trying again: a rate limit, a timeout, an
// outage on the provider side. Anything else - a bad credential, an unknown zone,
// a malformed request - fails the same way on every attempt and is returned bare,
// so the caller can stop instead of burning its retries.
type TemporaryError struct {
	// RetryAfter is how long to wait before trying again, when the provider says
	// so. Zero leaves the delay to the caller's backoff.
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

// AsTemporary reports whether the error is retryable, and how long the provider
// asked to wait.
func AsTemporary(err error) (time.Duration, bool) {
	var temporary *TemporaryError
	if errors.As(err, &temporary) {
		return temporary.RetryAfter, true
	}
	return 0, false
}
