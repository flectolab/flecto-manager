#!/bin/bash

go install go.uber.org/mock/mockgen@latest

rm -rf mocks

mockgen -destination=mocks/flecto-manager/repository/mock.go -package=mockFlectoRepository github.com/flectolab/flecto-manager/repository NamespaceRepository,ProjectRepository,UserRepository,RoleRepository,ResourcePermissionRepository,AdminPermissionRepository,RedirectRepository,RedirectDraftRepository,PageRepository,PageDraftRepository,AgentRepository

mockgen -destination=mocks/flecto-manager/service/mock.go -package=mockFlectoService github.com/flectolab/flecto-manager/service RoleService
