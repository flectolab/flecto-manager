package cloudflare

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/cdn"
	"github.com/stretchr/testify/assert"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	c := newClient("test-token", 0)
	c.baseURL = server.URL
	return c
}

func writeAPIResponse(t *testing.T, w http.ResponseWriter, status int, body apiResponse) {
	t.Helper()

	w.WriteHeader(status)
	assert.NoError(t, json.NewEncoder(w).Encode(body))
}

func TestClient_Do(t *testing.T) {
	t.Run("successWithAcceptedCall", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeAPIResponse(t, w, http.StatusOK, apiResponse{Success: true})
		})

		err := c.do(context.Background(), http.MethodGet, "/zones/abc", nil)

		assert.NoError(t, err)
	})

	t.Run("successSendsBearerToken", func(t *testing.T) {
		var got string
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			got = r.Header.Get("Authorization")
			writeAPIResponse(t, w, http.StatusOK, apiResponse{Success: true})
		})

		assert.NoError(t, c.do(context.Background(), http.MethodGet, "/zones/abc", nil))
		assert.Equal(t, "Bearer test-token", got)
	})

	t.Run("successSendsJSONBody", func(t *testing.T) {
		var (
			contentType string
			body        []byte
		)
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			contentType = r.Header.Get("Content-Type")
			body, _ = io.ReadAll(r.Body)
			writeAPIResponse(t, w, http.StatusOK, apiResponse{Success: true})
		})

		err := c.do(context.Background(), http.MethodPost, "/zones/abc/purge_cache", purgeRequest{Files: []string{"https://a.com/x"}})

		assert.NoError(t, err)
		assert.Equal(t, "application/json", contentType)
		assert.JSONEq(t, `{"files":["https://a.com/x"]}`, string(body))
	})

	t.Run("successOmitsContentTypeWithoutBody", func(t *testing.T) {
		var contentType string
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			contentType = r.Header.Get("Content-Type")
			writeAPIResponse(t, w, http.StatusOK, apiResponse{Success: true})
		})

		assert.NoError(t, c.do(context.Background(), http.MethodGet, "/zones/abc", nil))
		assert.Equal(t, "", contentType)
	})

	t.Run("failWithUnsuccessfulEnvelope", func(t *testing.T) {
		// Cloudflare answers 200 with success false on some rejections.
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeAPIResponse(t, w, http.StatusOK, apiResponse{
				Success: false,
				Errors:  []apiError{{Code: 1012, Message: "Request failed"}},
			})
		})

		err := c.do(context.Background(), http.MethodGet, "/zones/abc", nil)

		assert.ErrorContains(t, err, "1012: Request failed")
		_, temporary := cdn.AsTemporary(err)
		assert.False(t, temporary)
	})

	t.Run("failWithBadCredential", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeAPIResponse(t, w, http.StatusUnauthorized, apiResponse{
				Errors: []apiError{{Code: 10000, Message: "Authentication error"}},
			})
		})

		err := c.do(context.Background(), http.MethodGet, "/zones/abc", nil)

		assert.ErrorContains(t, err, "10000: Authentication error")
		_, temporary := cdn.AsTemporary(err)
		assert.False(t, temporary)
		assert.False(t, isForbidden(err))
	})

	t.Run("failWithForbidden", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeAPIResponse(t, w, http.StatusForbidden, apiResponse{
				Errors: []apiError{{Code: 9109, Message: "Unauthorized to access requested resource"}},
			})
		})

		err := c.do(context.Background(), http.MethodGet, "/zones/abc", nil)

		assert.True(t, isForbidden(err))
	})

	t.Run("failTemporarilyWithRateLimit", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "30")
			writeAPIResponse(t, w, http.StatusTooManyRequests, apiResponse{
				Errors: []apiError{{Code: 10000, Message: "rate limited"}},
			})
		})

		err := c.do(context.Background(), http.MethodGet, "/zones/abc", nil)

		retryAfter, temporary := cdn.AsTemporary(err)
		assert.True(t, temporary)
		assert.Equal(t, 30*time.Second, retryAfter)
	})

	t.Run("failTemporarilyWithServerError", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("<html>bad gateway</html>"))
		})

		err := c.do(context.Background(), http.MethodGet, "/zones/abc", nil)

		retryAfter, temporary := cdn.AsTemporary(err)
		assert.True(t, temporary)
		assert.Equal(t, time.Duration(0), retryAfter)
		assert.ErrorContains(t, err, "bad gateway")
	})

	t.Run("failTemporarilyWithTransportError", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		c := newClient("test-token", 0)
		c.baseURL = server.URL
		server.Close()

		err := c.do(context.Background(), http.MethodGet, "/zones/abc", nil)

		_, temporary := cdn.AsTemporary(err)
		assert.True(t, temporary)
	})

	t.Run("failWithoutDetails", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		})

		err := c.do(context.Background(), http.MethodGet, "/zones/abc", nil)

		assert.ErrorContains(t, err, "no details")
	})

	t.Run("failWithUnencodableBody", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeAPIResponse(t, w, http.StatusOK, apiResponse{Success: true})
		})

		err := c.do(context.Background(), http.MethodPost, "/zones/abc/purge_cache", make(chan int))

		assert.ErrorContains(t, err, "failed to encode request")
	})

	t.Run("failWithUnbuildableRequest", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {})

		err := c.do(context.Background(), "not a method", "/zones/abc", nil)

		assert.ErrorContains(t, err, "failed to build request")
	})

	t.Run("failWithCancelledContext", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeAPIResponse(t, w, http.StatusOK, apiResponse{Success: true})
		})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := c.do(ctx, http.MethodGet, "/zones/abc", nil)

		assert.Error(t, err)
	})
}

func TestRetryAfter(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{name: "successWithSeconds", header: "30", want: 30 * time.Second},
		{name: "successWithoutHeader", header: "", want: 0},
		{name: "successWithHTTPDate", header: "Wed, 21 Oct 2026 07:28:00 GMT", want: 0},
		{name: "successWithZero", header: "0", want: 0},
		{name: "successWithNegative", header: "-5", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := &http.Response{Header: http.Header{}}
			if tt.header != "" {
				response.Header.Set("Retry-After", tt.header)
			}

			assert.Equal(t, tt.want, retryAfter(response))
		})
	}
}

func TestApiError_String(t *testing.T) {
	assert.Equal(t, "1012: Request failed", apiError{Code: 1012, Message: "Request failed"}.String())
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		want    time.Duration
	}{
		{name: "successWithTheConfiguredTimeout", timeout: 3 * time.Second, want: 3 * time.Second},
		{name: "successFallsBackToTheDefault", timeout: 0, want: defaultTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newClient("test-token", tt.timeout)

			assert.Equal(t, tt.want, got.http.Timeout)
		})
	}
}
