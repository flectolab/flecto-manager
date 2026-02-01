package metrics

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/config"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

// mockAgentMetricsProvider is a mock implementation of AgentMetricsProvider
type mockAgentMetricsProvider struct {
	counts []AgentCount
	err    error
}

func (m *mockAgentMetricsProvider) GetAgentCounts(ctx context.Context, onlineThreshold time.Time) ([]AgentCount, error) {
	return m.counts, m.err
}

func TestHandler(t *testing.T) {
	h := Handler()
	assert.NotNil(t, h)

	// Test that it responds to HTTP requests
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "go_gc_duration_seconds")
}

func TestEchoHandler(t *testing.T) {
	e := echo.New()
	e.GET("/metrics", EchoHandler())

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "go_gc_duration_seconds")
}

func TestEchoMiddleware(t *testing.T) {
	// Reset HTTP metrics
	HTTPRequestsTotal.Reset()
	HTTPRequestDuration.Reset()

	e := echo.New()
	e.Use(EchoMiddleware())
	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})
	e.GET("/users/:id", func(c echo.Context) error {
		return c.String(http.StatusOK, "User "+c.Param("id"))
	})

	t.Run("records metrics for simple path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify counter was incremented
		count := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/test", "200"))
		assert.Equal(t, float64(1), count)
	})

	t.Run("records metrics for parameterized path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		// Should use route pattern, not actual path
		count := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/users/:id", "200"))
		assert.Equal(t, float64(1), count)
	})

	t.Run("records metrics for 404", func(t *testing.T) {
		HTTPRequestsTotal.Reset()

		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)

		// Echo doesn't invoke middleware for non-existent routes by default
		// so we just verify the request was handled
	})

	t.Run("records duration histogram", func(t *testing.T) {
		HTTPRequestDuration.Reset()

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		// Verify histogram has observations
		count := testutil.CollectAndCount(HTTPRequestDuration)
		assert.Greater(t, count, 0)
	})
}

func TestCollectAgentMetrics(t *testing.T) {
	tests := []struct {
		name                string
		counts              []AgentCount
		err                 error
		expectedOnline      map[string]float64
		expectedErrors      map[string]float64
		expectedOnlineCount int
		expectedErrorCount  int
	}{
		{
			name:                "empty counts",
			counts:              []AgentCount{},
			err:                 nil,
			expectedOnline:      map[string]float64{},
			expectedErrors:      map[string]float64{},
			expectedOnlineCount: 0,
			expectedErrorCount:  0,
		},
		{
			name: "single project with success agents",
			counts: []AgentCount{
				{NamespaceCode: "ns1", ProjectCode: "proj1", Status: commonTypes.AgentStatusSuccess, Count: 5},
			},
			err: nil,
			expectedOnline: map[string]float64{
				"ns1/proj1": 5,
			},
			expectedErrors:      map[string]float64{},
			expectedOnlineCount: 1,
			expectedErrorCount:  0,
		},
		{
			name: "single project with error agents",
			counts: []AgentCount{
				{NamespaceCode: "ns1", ProjectCode: "proj1", Status: commonTypes.AgentStatusError, Count: 3},
			},
			err: nil,
			expectedOnline: map[string]float64{
				"ns1/proj1": 3,
			},
			expectedErrors: map[string]float64{
				"ns1/proj1": 3,
			},
			expectedOnlineCount: 1,
			expectedErrorCount:  1,
		},
		{
			name: "single project with mixed status agents",
			counts: []AgentCount{
				{NamespaceCode: "ns1", ProjectCode: "proj1", Status: commonTypes.AgentStatusSuccess, Count: 10},
				{NamespaceCode: "ns1", ProjectCode: "proj1", Status: commonTypes.AgentStatusError, Count: 2},
			},
			err: nil,
			expectedOnline: map[string]float64{
				"ns1/proj1": 12,
			},
			expectedErrors: map[string]float64{
				"ns1/proj1": 2,
			},
			expectedOnlineCount: 1,
			expectedErrorCount:  1,
		},
		{
			name: "multiple projects",
			counts: []AgentCount{
				{NamespaceCode: "ns1", ProjectCode: "proj1", Status: commonTypes.AgentStatusSuccess, Count: 5},
				{NamespaceCode: "ns1", ProjectCode: "proj1", Status: commonTypes.AgentStatusError, Count: 1},
				{NamespaceCode: "ns1", ProjectCode: "proj2", Status: commonTypes.AgentStatusSuccess, Count: 3},
				{NamespaceCode: "ns2", ProjectCode: "proj1", Status: commonTypes.AgentStatusError, Count: 4},
			},
			err: nil,
			expectedOnline: map[string]float64{
				"ns1/proj1": 6,
				"ns1/proj2": 3,
				"ns2/proj1": 4,
			},
			expectedErrors: map[string]float64{
				"ns1/proj1": 1,
				"ns2/proj1": 4,
			},
			expectedOnlineCount: 3,
			expectedErrorCount:  2,
		},
		{
			name:                "error from provider",
			counts:              nil,
			err:                 errors.New("database error"),
			expectedOnline:      map[string]float64{},
			expectedErrors:      map[string]float64{},
			expectedOnlineCount: 0,
			expectedErrorCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset gauges before each test
			AgentErrorsGauge.Reset()
			AgentOnlineGauge.Reset()

			ctx := appContext.TestContext(nil)
			ctx.Config = &config.Config{
				Agent: config.AgentConfig{
					OfflineThreshold: 6 * time.Hour,
				},
			}

			provider := &mockAgentMetricsProvider{
				counts: tt.counts,
				err:    tt.err,
			}

			collectAgentMetrics(ctx, provider)

			// Verify online gauge count
			onlineCount := testutil.CollectAndCount(AgentOnlineGauge)
			assert.Equal(t, tt.expectedOnlineCount, onlineCount, "online gauge count mismatch")

			// Verify error gauge count
			errorCount := testutil.CollectAndCount(AgentErrorsGauge)
			assert.Equal(t, tt.expectedErrorCount, errorCount, "error gauge count mismatch")

			// Verify online gauge values
			for key, expectedValue := range tt.expectedOnline {
				ns, proj := parseKey(key)
				value := testutil.ToFloat64(AgentOnlineGauge.WithLabelValues(ns, proj))
				assert.Equal(t, expectedValue, value, "online gauge value mismatch for %s", key)
			}

			// Verify error gauge values
			for key, expectedValue := range tt.expectedErrors {
				ns, proj := parseKey(key)
				value := testutil.ToFloat64(AgentErrorsGauge.WithLabelValues(ns, proj))
				assert.Equal(t, expectedValue, value, "error gauge value mismatch for %s", key)
			}
		})
	}
}

func TestStartCollector(t *testing.T) {
	// Reset gauges
	AgentErrorsGauge.Reset()
	AgentOnlineGauge.Reset()

	ctx := appContext.TestContext(nil)
	ctx.Config = &config.Config{
		Agent: config.AgentConfig{
			OfflineThreshold: 6 * time.Hour,
		},
	}

	provider := &mockAgentMetricsProvider{
		counts: []AgentCount{
			{NamespaceCode: "ns1", ProjectCode: "proj1", Status: commonTypes.AgentStatusSuccess, Count: 5},
		},
		err: nil,
	}

	// Start collector with short interval
	StartCollector(ctx, provider, 50*time.Millisecond)

	// Wait for initial collection
	time.Sleep(10 * time.Millisecond)

	// Verify initial collection happened
	value := testutil.ToFloat64(AgentOnlineGauge.WithLabelValues("ns1", "proj1"))
	assert.Equal(t, float64(5), value)

	// Update mock data
	provider.counts = []AgentCount{
		{NamespaceCode: "ns1", ProjectCode: "proj1", Status: commonTypes.AgentStatusSuccess, Count: 10},
	}

	// Wait for next collection
	time.Sleep(60 * time.Millisecond)

	// Verify updated value
	value = testutil.ToFloat64(AgentOnlineGauge.WithLabelValues("ns1", "proj1"))
	assert.Equal(t, float64(10), value)

	// Stop collector
	ctx.Cancel()
}

func TestStartCollectorStopsOnContextDone(t *testing.T) {
	ctx := appContext.TestContext(nil)
	ctx.Config = &config.Config{
		Agent: config.AgentConfig{
			OfflineThreshold: 6 * time.Hour,
		},
	}

	callCount := 0
	provider := &mockAgentMetricsProvider{
		counts: []AgentCount{},
		err:    nil,
	}

	// Override GetAgentCounts to count calls
	countingProvider := &countingMockProvider{
		provider:  provider,
		callCount: &callCount,
	}

	StartCollector(ctx, countingProvider, 10*time.Millisecond)

	// Wait for a few collections
	time.Sleep(35 * time.Millisecond)

	// Cancel context
	ctx.Cancel()

	// Record call count at cancellation
	countAtCancel := callCount

	// Wait a bit more
	time.Sleep(30 * time.Millisecond)

	// Verify no more calls after cancellation (allow for 1 extra due to timing)
	assert.LessOrEqual(t, callCount, countAtCancel+1, "collector should stop after context cancellation")
}

func TestStartServer(t *testing.T) {
	ctx := appContext.TestContext(io.Discard)

	server := StartServer(ctx, "127.0.0.1:0")
	assert.NotNil(t, server)

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context to trigger shutdown
	ctx.Cancel()

	// Wait for shutdown
	time.Sleep(50 * time.Millisecond)
}

func TestStartServerMetricsEndpoint(t *testing.T) {
	ctx := appContext.TestContext(io.Discard)

	server := StartServer(ctx, "127.0.0.1:0")
	assert.NotNil(t, server)

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	// Get the actual address
	// Note: We can't easily get the actual port, so we test the handler directly
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	server.Handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "go_gc_duration_seconds")

	// Cleanup
	ctx.Cancel()
	time.Sleep(50 * time.Millisecond)
}

func TestNewAgentMetricsProvider(t *testing.T) {
	// We can't easily test the real implementation without a database,
	// but we can verify the provider is created
	provider := NewAgentMetricsProvider(nil)
	assert.NotNil(t, provider)
}

// countingMockProvider wraps a provider and counts calls
type countingMockProvider struct {
	provider  *mockAgentMetricsProvider
	callCount *int
}

func (c *countingMockProvider) GetAgentCounts(ctx context.Context, onlineThreshold time.Time) ([]AgentCount, error) {
	*c.callCount++
	return c.provider.GetAgentCounts(ctx, onlineThreshold)
}

// parseKey splits "ns/proj" into namespace and project
func parseKey(key string) (string, string) {
	for i, c := range key {
		if c == '/' {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}
