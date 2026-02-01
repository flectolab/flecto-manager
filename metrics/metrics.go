package metrics

import (
	"context"
	"net/http"
	"strconv"
	"time"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/service"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// AgentErrorsGauge tracks the number of agents in error status per namespace/project
	AgentErrorsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "flecto_agent_errors_total",
			Help: "Number of agents in error status (excluding offline agents)",
		},
		[]string{"namespace", "project"},
	)

	// AgentOnlineGauge tracks the number of online agents per namespace/project
	AgentOnlineGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "flecto_agent_online_total",
			Help: "Number of online agents",
		},
		[]string{"namespace", "project"},
	)

	// HTTPRequestsTotal counts HTTP requests
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "flecto_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration tracks HTTP request duration
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "flecto_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(AgentErrorsGauge)
	prometheus.MustRegister(AgentOnlineGauge)
	prometheus.MustRegister(HTTPRequestsTotal)
	prometheus.MustRegister(HTTPRequestDuration)
}

// AgentCount represents agent count for a namespace/project/status combination
type AgentCount struct {
	NamespaceCode string
	ProjectCode   string
	Status        commonTypes.AgentStatus
	Count         int64
}

// AgentMetricsProvider provides agent metrics data
type AgentMetricsProvider interface {
	GetAgentCounts(ctx context.Context, onlineThreshold time.Time) ([]AgentCount, error)
}

// agentMetricsProvider implements AgentMetricsProvider using AgentService
type agentMetricsProvider struct {
	agentService service.AgentService
}

// NewAgentMetricsProvider creates a new AgentMetricsProvider
func NewAgentMetricsProvider(agentService service.AgentService) AgentMetricsProvider {
	return &agentMetricsProvider{agentService: agentService}
}

func (p *agentMetricsProvider) GetAgentCounts(ctx context.Context, onlineThreshold time.Time) ([]AgentCount, error) {
	var counts []AgentCount
	err := p.agentService.GetQuery(ctx).
		Select("namespace_code, project_code, status, count(*) as count").
		Where("last_hit_at > ?", onlineThreshold).
		Group("namespace_code, project_code, status").
		Scan(&counts).Error

	return counts, err
}

// Handler returns the Prometheus metrics HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// EchoHandler returns an Echo handler for Prometheus metrics
func EchoHandler() echo.HandlerFunc {
	h := promhttp.Handler()
	return func(c echo.Context) error {
		h.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

// EchoMiddleware returns an Echo middleware that records HTTP metrics
func EchoMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			duration := time.Since(start).Seconds()
			status := c.Response().Status
			method := c.Request().Method
			path := c.Path()

			// Use the route path pattern, not the actual path
			if path == "" {
				path = c.Request().URL.Path
			}

			HTTPRequestsTotal.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
			HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)

			return err
		}
	}
}

// StartCollector starts a background goroutine that periodically updates agent metrics
func StartCollector(ctx *appContext.Context, provider AgentMetricsProvider, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Initial collection
		collectAgentMetrics(ctx, provider)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				collectAgentMetrics(ctx, provider)
			}
		}
	}()
}

func collectAgentMetrics(ctx *appContext.Context, provider AgentMetricsProvider) {
	onlineThreshold := time.Now().Add(-ctx.Config.Agent.OfflineThreshold)

	counts, err := provider.GetAgentCounts(context.Background(), onlineThreshold)
	if err != nil {
		ctx.Logger.Error("failed to collect agent metrics", "error", err)
		return
	}

	// Reset gauges to handle removed projects
	AgentErrorsGauge.Reset()
	AgentOnlineGauge.Reset()

	// Aggregate counts per namespace/project
	type projectKey struct {
		namespace string
		project   string
	}
	onlineCounts := make(map[projectKey]int64)
	errorCounts := make(map[projectKey]int64)

	for _, c := range counts {
		key := projectKey{namespace: c.NamespaceCode, project: c.ProjectCode}
		onlineCounts[key] += c.Count
		if c.Status == commonTypes.AgentStatusError {
			errorCounts[key] = c.Count
		}
	}

	// Update gauges
	for key, count := range onlineCounts {
		AgentOnlineGauge.WithLabelValues(key.namespace, key.project).Set(float64(count))
	}
	for key, count := range errorCounts {
		AgentErrorsGauge.WithLabelValues(key.namespace, key.project).Set(float64(count))
	}
}

// StartServer starts a dedicated metrics server on the specified address
func StartServer(ctx *appContext.Context, listen string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", Handler())

	server := &http.Server{
		Addr:    listen,
		Handler: mux,
	}

	go func() {
		ctx.Logger.Info("starting metrics server", "listen", listen)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ctx.Logger.Error("metrics server error", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()

	return server
}
