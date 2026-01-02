package repository

import "gorm.io/gorm"

type Repositories struct {
	Namespace     NamespaceRepository
	Project       ProjectRepository
	User          UserRepository
	Role          RoleRepository
	Redirect      RedirectRepository
	RedirectDraft RedirectDraftRepository
	Page          PageRepository
	PageDraft     PageDraftRepository
	Agent         AgentRepository
	Token         TokenRepository
}

func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Namespace:     NewNamespaceRepository(db),
		Project:       NewProjectRepository(db),
		User:          NewUserRepository(db),
		Role:          NewRoleRepository(db),
		Redirect:      NewRedirectRepository(db),
		RedirectDraft: NewRedirectDraftRepository(db),
		Page:          NewPageRepository(db),
		PageDraft:     NewPageDraftRepository(db),
		Agent:         NewAgentRepository(db),
		Token:         NewTokenRepository(db),
	}
}
