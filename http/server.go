package http

import (
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/flectolab/flecto-manager/auth"
	"github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/database"
	"github.com/flectolab/flecto-manager/graph"
	"github.com/flectolab/flecto-manager/graph/resolver"
	"github.com/flectolab/flecto-manager/http/api/project"
	"github.com/flectolab/flecto-manager/http/key"
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

	// CORS middleware
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST, echo.OPTIONS},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, ctx.Config.Auth.JWT.HeaderName},
	}))

	db, errDb := database.CreateDB(ctx)
	if errDb != nil {
		return nil, errDb
	}

	// Auth setup
	jwtService := jwt.NewServiceJWT(&ctx.Config.Auth.JWT)
	authService := auth.NewAuthService(db, jwtService)
	authHandler := auth.NewHandler(authService, jwtService, &ctx.Config.Auth.JWT)
	authHandler.RegisterRoutes(e)

	namespaceRepo := repository.NewNamespaceRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	resourcePermRepo := repository.NewResourcePermissionRepository(db)
	adminPermRepo := repository.NewAdminPermissionRepository(db)
	redirectRepo := repository.NewRedirectRepository(db)
	redirectDraftRepo := repository.NewRedirectDraftRepository(db)
	pageRepo := repository.NewPageRepository(db)
	pageDraftRepo := repository.NewPageDraftRepository(db)

	projectService := service.NewProjectService(ctx.Validator, projectRepo, pageRepo, redirectDraftRepo, pageDraftRepo, ctx.Config.Page, db)
	userService := service.NewUserService(ctx.Validator, userRepo, roleRepo)
	roleService := service.NewRoleService(ctx.Validator, roleRepo, userRepo, resourcePermRepo, adminPermRepo, db)
	redirectService := service.NewRedirectService(redirectRepo)
	pageSerivce := service.NewPageService(pageRepo)
	pageDraftService := service.NewPageDraftService(ctx.Validator, pageDraftRepo, pageRepo, db, ctx.Config.Page)
	permissionChecker := auth.NewPermissionChecker(roleService)
	// GraphQL setup
	srv := handler.New(graph.NewExecutableSchema(graph.Config{
		Resolvers: &resolver.Resolver{
			DB:                    db,
			PermissionChecker:     permissionChecker,
			NamespaceService:      service.NewNamespaceService(ctx.Validator, namespaceRepo, projectRepo),
			ProjectService:        projectService,
			UserService:           userService,
			RoleService:           roleService,
			RedirectService:       redirectService,
			RedirectDraftService:  service.NewRedirectDraftService(ctx.Validator, redirectDraftRepo, db),
			RedirectImportService: service.NewRedirectImportService(ctx.Validator, db, redirectDraftRepo),
			PageService:           pageSerivce,
			PageDraftService:      pageDraftService,
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

	// GraphQL endpoint avec middleware d'auth
	graphqlGroup := e.Group("")
	graphqlGroup.Use(auth.GraphQLAuthMiddleware(&ctx.Config.Auth.JWT, userService, roleService))
	graphqlGroup.POST("/graphql", echo.WrapHandler(srv))

	apiGroup := e.Group("/api")
	apiGroup.Use(auth.GraphQLAuthMiddleware(&ctx.Config.Auth.JWT, userService, roleService))
	namespacesGroup := apiGroup.Group("/namespace")
	namespaceGroup := namespacesGroup.Group("/:" + key.NamespaceCodeKey)
	projectsGroup := namespaceGroup.Group("/project")
	projectGroup := projectsGroup.Group("/:" + key.ProjectCodeKey)
	projectGroup.GET("/version", project.GetVersion(permissionChecker, projectService))
	projectGroup.GET("/redirects", project.GetRedirects(permissionChecker, redirectService))
	projectGroup.GET("/pages", project.GetPages(permissionChecker, pageSerivce))

	// GraphQL playground (development only)
	e.GET("/playground", echo.WrapHandler(playground.Handler("GraphQL Playground", "/graphql")))
	registerUI(ctx, e)

	return e, nil
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

func createServerHTTP() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Logger.SetOutput(io.Discard)
	return e
}
