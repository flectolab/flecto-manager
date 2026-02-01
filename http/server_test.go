package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/auth"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/database"
	"github.com/flectolab/flecto-manager/jwt"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/service"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	// Register sqlite dialector for tests
	database.FactoryDialector[database.DbTypeSqlite] = database.CreateDialectorSqlite
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	return db
}

func setupTestContext(t *testing.T) *appContext.Context {
	ctx := appContext.TestContext(nil)
	ctx.Config.Auth.JWT.Secret = "test-secret-key-32-bytes-long!!!"
	ctx.Config.DB.Type = database.DbTypeSqlite
	ctx.Config.DB.Config = map[string]interface{}{"dsn": ":memory:"}
	return ctx
}

func setupTestServices(t *testing.T, ctx *appContext.Context) (*service.Services, *jwt.ServiceJWT) {
	db := setupTestDB(t)
	repos := repository.NewRepositories(db)
	jwtService := jwt.NewServiceJWT(&ctx.Config.Auth.JWT)
	services := service.NewServices(ctx, repos, jwtService)
	return services, jwtService
}

func TestCreateServerHTTP(t *testing.T) {
	ctx := setupTestContext(t)

	e, err := CreateServerHTTP(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, e)
}

func TestCreateServerHTTPInternalFunction(t *testing.T) {
	e := createServerHTTP()

	assert.NotNil(t, e)
	assert.True(t, e.HideBanner)
	assert.True(t, e.HidePort)
}

func TestSetupCORS(t *testing.T) {
	ctx := setupTestContext(t)
	e := createServerHTTP()

	setupCORS(e, ctx)

	// Verify CORS by making a request with Origin header
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestSetupAuthRoutes(t *testing.T) {
	t.Run("without openid", func(t *testing.T) {
		ctx := setupTestContext(t)
		ctx.Config.Auth.OpenID.Enabled = false
		e := createServerHTTP()
		services, jwtService := setupTestServices(t, ctx)
		authMiddleware := echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		})

		err := setupAuthRoutes(ctx, e, services, jwtService, authMiddleware)

		assert.NoError(t, err)

		// Verify routes are registered
		routes := e.Routes()
		routePaths := make(map[string]bool)
		for _, r := range routes {
			routePaths[r.Method+":"+r.Path] = true
		}

		assert.True(t, routePaths["POST:/auth/login"])
		assert.True(t, routePaths["POST:/auth/refresh"])
		assert.True(t, routePaths["POST:/auth/logout"])
		assert.True(t, routePaths["GET:/auth/openid"])
	})

	t.Run("with openid enabled but invalid provider", func(t *testing.T) {
		ctx := setupTestContext(t)
		ctx.Config.Auth.OpenID.Enabled = true
		ctx.Config.Auth.OpenID.ProviderURL = "http://invalid-provider"
		ctx.Config.Auth.OpenID.ClientID = "test-client"
		ctx.Config.Auth.OpenID.ClientSecret = "test-secret"
		ctx.Config.Auth.OpenID.RedirectURL = "http://localhost/callback"
		e := createServerHTTP()
		services, jwtService := setupTestServices(t, ctx)
		authMiddleware := echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		})

		err := setupAuthRoutes(ctx, e, services, jwtService, authMiddleware)

		// Should fail because provider URL is invalid
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create OpenID provider")
	})
}

func TestSetupGraphQLRoutes(t *testing.T) {
	ctx := setupTestContext(t)
	e := createServerHTTP()
	services, _ := setupTestServices(t, ctx)
	permissionChecker := auth.NewPermissionChecker(services.Role)
	authMiddleware := echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	})

	setupGraphQLRoutes(ctx, e, services, permissionChecker, authMiddleware)

	// Verify GraphQL route is registered
	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Method+":"+r.Path] = true
	}

	assert.True(t, routePaths["POST:/graphql"])
}

func TestCreateGraphQLHandler(t *testing.T) {
	ctx := setupTestContext(t)
	services, _ := setupTestServices(t, ctx)
	permissionChecker := auth.NewPermissionChecker(services.Role)

	handler := createGraphQLHandler(ctx, services, permissionChecker)

	assert.NotNil(t, handler)
}

func TestSetupAPIRoutes(t *testing.T) {
	ctx := setupTestContext(t)
	e := createServerHTTP()
	services, _ := setupTestServices(t, ctx)
	permissionChecker := auth.NewPermissionChecker(services.Role)
	authMiddleware := echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	})

	setupAPIRoutes(e, services, permissionChecker, authMiddleware)

	// Verify API routes are registered
	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Method+":"+r.Path] = true
	}

	assert.True(t, routePaths["GET:/api/namespace/:namespaceCode/project/:projectCode/version"])
	assert.True(t, routePaths["GET:/api/namespace/:namespaceCode/project/:projectCode/redirects"])
	assert.True(t, routePaths["GET:/api/namespace/:namespaceCode/project/:projectCode/pages"])
	assert.True(t, routePaths["POST:/api/namespace/:namespaceCode/project/:projectCode/agents"])
	assert.True(t, routePaths["PATCH:/api/namespace/:namespaceCode/project/:projectCode/agents/:name/hit"])
}

func TestRegisterUI(t *testing.T) {
	ctx := setupTestContext(t)
	e := createServerHTTP()

	registerUI(ctx, e)

	// Verify UI routes are registered
	routes := e.Routes()
	hasRootRoute := false
	hasWildcardRoute := false
	for _, r := range routes {
		if r.Method == "GET" && (r.Path == "" || r.Path == "/") {
			hasRootRoute = true
		}
		if r.Method == "GET" && r.Path == "/*" {
			hasWildcardRoute = true
		}
	}

	assert.True(t, hasRootRoute, "should have root GET route")
	assert.True(t, hasWildcardRoute, "should have wildcard GET route")
}

func TestCreateServerHTTP_WithDatabaseConfig(t *testing.T) {
	ctx := setupTestContext(t)
	ctx.Config.DB.Type = database.DbTypeSqlite
	ctx.Config.DB.Config = map[string]interface{}{"dsn": ":memory:"}
	ctx.Config.Auth.JWT.Secret = "test-secret-key-32-bytes-long!!!"
	ctx.Config.Auth.JWT.AccessTokenTTL = 15 * time.Minute
	ctx.Config.Auth.JWT.RefreshTokenTTL = 24 * time.Hour

	e, err := CreateServerHTTP(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, e)

	if e != nil {
		// Verify all routes are registered
		routes := e.Routes()
		assert.Greater(t, len(routes), 0)
	}
}

func TestSetupMetrics(t *testing.T) {
	t.Run("without separate listen address", func(t *testing.T) {
		ctx := setupTestContext(t)
		ctx.Config.Metrics.Enabled = true
		ctx.Config.Metrics.Listen = ""
		ctx.Config.Agent.OfflineThreshold = 6 * time.Hour
		e := createServerHTTP()
		services, _ := setupTestServices(t, ctx)

		setupMetrics(ctx, e, services.Agent)

		// Verify /metrics route is registered
		routes := e.Routes()
		routePaths := make(map[string]bool)
		for _, r := range routes {
			routePaths[r.Method+":"+r.Path] = true
		}
		assert.True(t, routePaths["GET:/metrics"])

		// Test metrics endpoint works
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "go_gc_duration_seconds")
	})

	t.Run("with separate listen address", func(t *testing.T) {
		ctx := setupTestContext(t)
		ctx.Config.Metrics.Enabled = true
		ctx.Config.Metrics.Listen = "127.0.0.1:0"
		ctx.Config.Agent.OfflineThreshold = 6 * time.Hour
		e := createServerHTTP()
		services, _ := setupTestServices(t, ctx)

		setupMetrics(ctx, e, services.Agent)

		// Verify /metrics route is NOT registered on main server
		routes := e.Routes()
		routePaths := make(map[string]bool)
		for _, r := range routes {
			routePaths[r.Method+":"+r.Path] = true
		}
		assert.False(t, routePaths["GET:/metrics"])
	})

	t.Run("metrics middleware records requests", func(t *testing.T) {
		ctx := setupTestContext(t)
		ctx.Config.Metrics.Enabled = true
		ctx.Config.Metrics.Listen = ""
		ctx.Config.Agent.OfflineThreshold = 6 * time.Hour
		e := createServerHTTP()
		services, _ := setupTestServices(t, ctx)

		setupMetrics(ctx, e, services.Agent)

		// Add a test route
		e.GET("/test", func(c echo.Context) error {
			return c.String(http.StatusOK, "OK")
		})

		// Make a request
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify metrics were recorded
		metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		metricsRec := httptest.NewRecorder()
		e.ServeHTTP(metricsRec, metricsReq)

		assert.Contains(t, metricsRec.Body.String(), "flecto_http_requests_total")
		assert.Contains(t, metricsRec.Body.String(), "flecto_http_request_duration_seconds")
	})
}

func TestCreateServerHTTP_WithMetricsEnabled(t *testing.T) {
	ctx := setupTestContext(t)
	ctx.Config.DB.Type = database.DbTypeSqlite
	ctx.Config.DB.Config = map[string]interface{}{"dsn": ":memory:"}
	ctx.Config.Auth.JWT.Secret = "test-secret-key-32-bytes-long!!!"
	ctx.Config.Auth.JWT.AccessTokenTTL = 15 * time.Minute
	ctx.Config.Auth.JWT.RefreshTokenTTL = 24 * time.Hour
	ctx.Config.Metrics.Enabled = true
	ctx.Config.Metrics.Listen = ""
	ctx.Config.Agent.OfflineThreshold = 6 * time.Hour

	e, err := CreateServerHTTP(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, e)

	if e != nil {
		// Verify metrics route is registered
		routes := e.Routes()
		hasMetrics := false
		for _, r := range routes {
			if r.Method == "GET" && r.Path == "/metrics" {
				hasMetrics = true
				break
			}
		}
		assert.True(t, hasMetrics, "should have /metrics route when metrics enabled")
	}
}
