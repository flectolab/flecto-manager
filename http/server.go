package http

import (
	builtinCtx "context"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/flectolab/flecto-manager/auth"
	"github.com/flectolab/flecto-manager/auth/openid"
	"github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/database"
	"github.com/flectolab/flecto-manager/graph"
	"github.com/flectolab/flecto-manager/graph/resolver"
	"github.com/flectolab/flecto-manager/http/route"
	"github.com/flectolab/flecto-manager/http/route/api/project"
	routeAuth "github.com/flectolab/flecto-manager/http/route/auth"
	"github.com/flectolab/flecto-manager/http/route/health"
	"github.com/flectolab/flecto-manager/jwt"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/service"
	"github.com/flectolab/flecto-manager/webui"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/vektah/gqlparser/v2/ast"
)

func CreateServerHTTP(ctx *context.Context) (*echo.Echo, error) {
	e := createServerHTTP()
	e.Logger.SetOutput(os.Stdout)

	setupCORS(e, ctx)

	db, err := database.CreateDB(ctx)
	if err != nil {
		return nil, err
	}

	jwtService := jwt.NewServiceJWT(&ctx.Config.Auth.JWT)
	repos := repository.NewRepositories(db)
	services := service.NewServices(ctx, repos, jwtService)
	permissionChecker := auth.NewPermissionChecker(services.Role)

	authMiddleware := auth.UserCtxAuthMiddleware(&ctx.Config.Auth.JWT, services.User, services.Role, services.Token)

	e.GET("/health/ping", health.GetPing())
	if err = setupAuthRoutes(ctx, e, services, jwtService, authMiddleware); err != nil {
		return nil, err
	}
	setupGraphQLRoutes(ctx, e, services, permissionChecker, authMiddleware)
	setupAPIRoutes(e, services, permissionChecker, authMiddleware)
	registerUI(ctx, e)

	return e, nil
}

func createServerHTTP() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Logger.SetOutput(io.Discard)
	return e
}

func setupCORS(e *echo.Echo, ctx *context.Context) {
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PATCH, echo.OPTIONS},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, ctx.Config.Auth.JWT.HeaderName},
	}))
}

func setupAuthRoutes(ctx *context.Context, e *echo.Echo, services *service.Services, jwtService *jwt.ServiceJWT, authMiddleware echo.MiddlewareFunc) error {
	authGroup := e.Group("/auth")
	authGroup.POST("/login", routeAuth.GetLogin(ctx, services.Auth))
	authGroup.POST("/refresh", routeAuth.GetRefresh(ctx, services.Auth))
	authGroup.POST("/logout", routeAuth.GetLogout(ctx, services.Auth), authMiddleware)

	// OpenID Connect (if enabled)
	if ctx.Config.Auth.OpenID.Enabled {
		openidProvider, err := openid.NewProvider(builtinCtx.Background(), &ctx.Config.Auth.OpenID)
		if err != nil {
			return fmt.Errorf("failed to create OpenID provider: %w", err)
		}
		openidService := openid.NewService(openidProvider, services.User, jwtService)
		authGroup.GET("/openid", routeAuth.GetOpenIDConfig(&ctx.Config.Auth.OpenID, openidService))
		authGroup.GET("/openid/callback", routeAuth.GetOpenIDCallback(openidService))
	} else {
		authGroup.GET("/openid", routeAuth.GetOpenIDConfig(&ctx.Config.Auth.OpenID, nil))
	}

	return nil
}

func setupGraphQLRoutes(ctx *context.Context, e *echo.Echo, services *service.Services, permissionChecker *auth.PermissionChecker, authMiddleware echo.MiddlewareFunc) {
	srv := createGraphQLHandler(ctx, services, permissionChecker)

	graphqlGroup := e.Group("")
	graphqlGroup.Use(authMiddleware)
	graphqlGroup.POST("/graphql", echo.WrapHandler(srv))
}

func createGraphQLHandler(ctx *context.Context, services *service.Services, permissionChecker *auth.PermissionChecker) *handler.Server {
	srv := handler.New(graph.NewExecutableSchema(graph.Config{
		Resolvers: &resolver.Resolver{
			PermissionChecker:       permissionChecker,
			NamespaceService:        services.Namespace,
			ProjectService:          services.Project,
			UserService:             services.User,
			RoleService:             services.Role,
			TokenService:            services.Token,
			RedirectService:         services.Redirect,
			RedirectDraftService:    services.RedirectDraft,
			RedirectImportService:   services.RedirectImport,
			PageService:             services.Page,
			PageDraftService:        services.PageDraft,
			AgentService:            services.Agent,
			ProjectDashboardService: services.ProjectDashboard,
			AgentConfig:             ctx.Config.Agent,
		},
		Directives: graph.DirectiveRoot{Public: graph.PublicDirective},
	}))

	srv.AroundFields(graph.AuthMiddleware)

	// Add transports
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{
		MaxMemory:     2 << 20, // 2MB
		MaxUploadSize: 2 << 20, // 2MB
	})

	// Add extensions
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	return srv
}

func setupAPIRoutes(e *echo.Echo, services *service.Services, permissionChecker *auth.PermissionChecker, authMiddleware echo.MiddlewareFunc) {
	apiGroup := e.Group("/api")
	apiGroup.Use(authMiddleware)

	namespacesGroup := apiGroup.Group("/namespace")
	namespaceGroup := namespacesGroup.Group("/:" + route.NamespaceCodeKey)
	projectsGroup := namespaceGroup.Group("/project")
	projectGroup := projectsGroup.Group("/:" + route.ProjectCodeKey)

	projectGroup.GET("/version", project.GetVersion(permissionChecker, services.Project))
	projectGroup.GET("/redirects", project.GetRedirects(permissionChecker, services.Redirect))
	projectGroup.GET("/pages", project.GetPages(permissionChecker, services.Page))
	projectGroup.POST("/agents", project.PostAgent(permissionChecker, services.Agent))
	projectGroup.PATCH(fmt.Sprintf("/agents/:%s/hit", route.NameKey), project.PatchAgentHit(permissionChecker, services.Agent))
}

func registerUI(ctx *context.Context, e *echo.Echo) {
	distFS, _ := fs.Sub(webui.StaticFS, "dist")
	e.StaticFS("", distFS)

	handlerIndex := func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
		indexTmpl, err := template.New("index.html").Funcs(map[string]any{
			"AuthHeaderName": func() string { return ctx.Config.Auth.JWT.HeaderName },
		}).ParseFS(distFS, "index.html")
		if err != nil {
			return c.String(http.StatusNotFound, "Not found")
		}

		return indexTmpl.Execute(c.Response().Writer, ctx.Config.HTTP)
	}

	staticMiddleware := middleware.StaticWithConfig(middleware.StaticConfig{
		Filesystem: http.FS(distFS),
		HTML5:      true,
	})

	e.GET("", handlerIndex)
	e.GET("/*", handlerIndex, staticMiddleware)
}
