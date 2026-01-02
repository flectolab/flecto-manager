package service

import (
	"context"
	"time"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"gorm.io/gorm"
)

type AgentService interface {
	Upsert(ctx context.Context, agent *model.Agent) error
	GetByName(ctx context.Context, namespaceCode, projectCode, name string) (*model.Agent, error)
	FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.Agent, error)
	SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.AgentList, error)
	CountByProjectAndStatus(ctx context.Context, namespaceCode, projectCode string, status commonTypes.AgentStatus, lastHitAfter time.Time) (int64, error)
	UpdateLastHit(ctx context.Context, namespaceCode, projectCode, name string) error
	Delete(ctx context.Context, namespaceCode, projectCode, name string) error
}

type agentService struct {
	repo repository.AgentRepository
}

func NewAgentService(repo repository.AgentRepository) AgentService {
	return &agentService{
		repo: repo,
	}
}

func (s *agentService) Upsert(ctx context.Context, agent *model.Agent) error {
	if err := commonTypes.ValidateAgent(agent.Agent); err != nil {
		return err
	}
	return s.repo.Upsert(ctx, agent)
}

func (s *agentService) GetByName(ctx context.Context, namespaceCode, projectCode, name string) (*model.Agent, error) {
	return s.repo.FindByName(ctx, namespaceCode, projectCode, name)
}

func (s *agentService) FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.Agent, error) {
	return s.repo.FindByProject(ctx, namespaceCode, projectCode)
}

func (s *agentService) SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.AgentList, error) {
	agents, total, err := s.repo.SearchPaginate(ctx, query, pagination.GetLimit(), pagination.GetOffset())
	if err != nil {
		return nil, err
	}

	return &model.AgentList{
		Total:  int(total),
		Offset: pagination.GetOffset(),
		Limit:  pagination.GetLimit(),
		Items:  agents,
	}, nil
}

func (s *agentService) CountByProjectAndStatus(ctx context.Context, namespaceCode, projectCode string, status commonTypes.AgentStatus, lastHitAfter time.Time) (int64, error) {
	return s.repo.CountByProjectAndStatus(ctx, namespaceCode, projectCode, status, lastHitAfter)
}

func (s *agentService) UpdateLastHit(ctx context.Context, namespaceCode, projectCode, name string) error {
	return s.repo.UpdateLastHit(ctx, namespaceCode, projectCode, name)
}

func (s *agentService) Delete(ctx context.Context, namespaceCode, projectCode, name string) error {
	return s.repo.Delete(ctx, namespaceCode, projectCode, name)
}