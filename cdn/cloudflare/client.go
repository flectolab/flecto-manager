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

// defaultBaseURL is not configurable, so the API tokens cannot be sent elsewhere.
const defaultBaseURL = "https://api.cloudflare.com/client/v4"

// defaultTimeout applies when the instance sets none.
const defaultTimeout = 10 * time.Second

// maxErrorBodySize keeps e.g. a proxy's HTML error page out of log lines.
const maxErrorBodySize = 4 * 1024

// client makes one Cloudflare API call and maps its outcome to nil, a permanent
// error or a cdn.TemporaryError.
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

// statusError is a call the API rejected, with its status code.
type statusError struct {
	StatusCode int
	Status     string
	Details    string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("cloudflare returned %s: %s", e.Status, e.Details)
}

func isForbidden(err error) bool {
	var status *statusError
	return errors.As(err, &status) && status.StatusCode == http.StatusForbidden
}

// do performs one call. Success needs both a 200 and "success": true, since
// Cloudflare answers 200 with "success": false on some failures.
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

	// Only rate limits and server errors are retryable.
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError {
		return cdn.Temporary(rejected, retryAfter(response))
	}
	return rejected
}

// describe renders the API errors, or the raw body without the usual envelope.
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

// retryAfter reads Retry-After in seconds; Cloudflare does not use the HTTP-date form.
func retryAfter(response *http.Response) time.Duration {
	seconds, err := strconv.Atoi(response.Header.Get("Retry-After"))
	if err != nil || seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
