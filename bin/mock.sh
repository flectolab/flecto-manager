#!/bin/bash

go install go.uber.org/mock/mockgen@latest

rm -rf mocks

mockgen -destination=mocks/flecto-manager/repository/mock.go -package=mockFlectoRepository github.com/flectolab/flecto-manager/repository NamespaceRepository,ProjectRepository,UserRepository,RoleRepository,ResourcePermissionRepository,AdminPermissionRepository,RedirectRepository,RedirectDraftRepository,PageRepository,PageDraftRepository,AgentRepository,TokenRepository

mockgen -destination=mocks/flecto-manager/service/mock.go -package=mockFlectoService github.com/flectolab/flecto-manager/service RoleService,AuthService,TokenService,UserService,ProjectService,RedirectService,RedirectDraftService,PageService,PageDraftService,AgentService

mockgen -destination=mocks/flecto-manager/cli/db/mock.go -package=mockMigratorDB github.com/flectolab/flecto-manager/cli/db Migrator

mockgen -destination=mocks/flecto-manager/auth/openid/mock.go -package=mockOpenID github.com/flectolab/flecto-manager/auth/openid Provider,Service

mockgen -destination=mocks/flecto-manager/metrics/mock.go -package=mockMetrics github.com/flectolab/flecto-manager/metrics AgentMetricsProvider
