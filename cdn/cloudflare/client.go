package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/flectolab/flecto-manager/cdn"
)

// defaultBaseURL is not configurable: an operator able to point the client at
// another host would be able to collect the API token of every project.
const defaultBaseURL = "https://api.cloudflare.com/client/v4"

// defaultTimeout bounds a single call when the instance sets no timeout of its
// own. A purge is retried by the worker, so waiting longer buys little.
const defaultTimeout = 10 * time.Second

// maxErrorBodySize bounds what is read from a response that is not valid JSON,
// so a proxy returning an HTML error page does not end up whole in a log line.
const maxErrorBodySize = 4 * 1024

// client talks to the Cloudflare API. It only knows how to make one call and
// turn its outcome into either nil, a permanent error, or a cdn.TemporaryError.
type client struct {
	baseURL string
	token   string
	http    *http.Client
}

func newClient(token string, timeout time.Duration) *client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &client{
		baseURL: defaultBaseURL,
		token:   token,
		http:    &http.Client{Timeout: timeout},
	}
}

// apiResponse is the envelope every Cloudflare endpoint answers with.
type apiResponse struct {
	Success bool       `json:"success"`
	Errors  []apiError `json:"errors"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e apiError) String() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

// statusError is a call the API rejected. It keeps the status code so that a
// caller can tell a narrowly scoped token, which is a normal setup, from a
// credential that is simply wrong.
type statusError struct {
	StatusCode int
	Status     string
	Details    string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("cloudflare returned %s: %s", e.Status, e.Details)
}

// isForbidden reports whether the call was rejected for lack of permission.
func isForbidden(err error) bool {
	var status *statusError
	return errors.As(err, &status) && status.StatusCode == http.StatusForbidden
}

// do performs one call and reports whether the API accepted it. A response is
// only a success when both the status code and the envelope say so: Cloudflare
// answers 200 with "success": false on some failures.
func (c *client) do(ctx context.Context, method, path string, body any) error {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode request: %w", err)
		}
		payload = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, payload)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.http.Do(request)
	if err != nil {
		// A transport error is a network problem, never a rejection: it is worth
		// another attempt.
		return cdn.Temporary(fmt.Errorf("cloudflare request failed: %w", err), 0)
	}
	defer func() { _ = response.Body.Close() }()

	return readResponse(response)
}

func readResponse(response *http.Response) error {
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxErrorBodySize))
	if err != nil {
		return cdn.Temporary(fmt.Errorf("failed to read cloudflare response: %w", err), 0)
	}

	var decoded apiResponse
	decodeErr := json.Unmarshal(raw, &decoded)

	if response.StatusCode == http.StatusOK && decodeErr == nil && decoded.Success {
		return nil
	}

	rejected := &statusError{
		StatusCode: response.StatusCode,
		Status:     response.Status,
		Details:    describe(decoded, decodeErr, raw),
	}

	// A rate limit and a server-side failure are worth another attempt. Everything
	// else - a bad token, an unknown zone, a malformed body - fails identically on
	// every try.
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError {
		return cdn.Temporary(rejected, retryAfter(response))
	}
	return rejected
}

// describe renders what the API said, falling back to the raw body when it did
// not answer with the usual envelope.
func describe(decoded apiResponse, decodeErr error, raw []byte) string {
	if decodeErr != nil || len(decoded.Errors) == 0 {
		body := strings.TrimSpace(string(raw))
		if body == "" {
			return "no details"
		}
		return body
	}

	messages := make([]string, 0, len(decoded.Errors))
	for _, apiErr := range decoded.Errors {
		messages = append(messages, apiErr.String())
	}
	return strings.Join(messages, ", ")
}

// retryAfter reads the delay the API asked for. Only the delay-seconds form is
// handled: Cloudflare does not use the HTTP-date form, and guessing wrong on a
// rate limit is worse than falling back to the caller's own backoff.
func retryAfter(response *http.Response) time.Duration {
	seconds, err := strconv.Atoi(response.Header.Get("Retry-After"))
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
