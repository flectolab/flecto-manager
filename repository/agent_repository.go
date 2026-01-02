package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
)

type AgentRepository interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	Upsert(ctx context.Context, agent *model.Agent) error
	FindByName(ctx context.Context, namespaceCode, projectCode, name string) (*model.Agent, error)
	FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.Agent, error)
	SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.Agent, int64, error)
	CountByProjectAndStatus(ctx context.Context, namespaceCode, projectCode string, status commonTypes.AgentStatus, lastHitAfter time.Time) (int64, error)
	UpdateLastHit(ctx context.Context, namespaceCode, projectCode, name string) error
	Delete(ctx context.Context, namespaceCode, projectCode, name string) error
}

type agentRepository struct {
	db *gorm.DB
}

func NewAgentRepository(db *gorm.DB) AgentRepository {
	return &agentRepository{db: db}
}

func (r *agentRepository) GetTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *agentRepository) GetQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.Agent{})
}

var (
	ErrAgentMissingStatus       = errors.New("agent status is required for creation")
	ErrAgentMissingType         = errors.New("agent type is required for creation")
	ErrAgentMissingLoadDuration = errors.New("agent load_duration is required for creation")
)

func (r *agentRepository) Upsert(ctx context.Context, agent *model.Agent) error {
	var existing model.Agent
	err := r.db.WithContext(ctx).
		Where(fmt.Sprintf("%s = ? AND %s = ? AND name = ?", model.ColumnNamespaceCode, model.ColumnProjectCode),
			agent.NamespaceCode, agent.ProjectCode, agent.Name).
		First(&existing).Error
	agent.LastHitAt = r.db.NowFunc()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if agent.Status == "" {
				return ErrAgentMissingStatus
			}
			if agent.Type == "" {
				return ErrAgentMissingType
			}
			if agent.LoadDuration == 0 {
				return ErrAgentMissingLoadDuration
			}
			return r.db.WithContext(ctx).Create(agent).Error
		}
		return err
	}

	agent.ID = existing.ID
	existing.Agent = agent.Agent
	existing.LastHitAt = agent.LastHitAt
	return r.db.WithContext(ctx).Save(existing).Error
}

func (r *agentRepository) FindByName(ctx context.Context, namespaceCode, projectCode, name string) (*model.Agent, error) {
	var agent model.Agent
	err := r.db.WithContext(ctx).
		Where(fmt.Sprintf("%s = ? AND %s = ? AND name = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode, name).
		First(&agent).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

func (r *agentRepository) FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.Agent, error) {
	var agents []model.Agent
	err := r.db.WithContext(ctx).
		Where(fmt.Sprintf("%s = ? AND %s = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode).
		Find(&agents).Error
	if err != nil {
		return nil, err
	}
	return agents, nil
}

func (r *agentRepository) SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.Agent, int64, error) {
	var total int64
	if query == nil {
		query = r.db.WithContext(ctx).Model(&model.Agent{})
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit != 0 {
		query = query.Limit(limit).Offset(offset)
	}

	var agents []model.Agent
	if err := query.Find(&agents).Error; err != nil {
		return nil, 0, err
	}

	return agents, total, nil
}

func (r *agentRepository) CountByProjectAndStatus(ctx context.Context, namespaceCode, projectCode string, status commonTypes.AgentStatus, lastHitAfter time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Agent{}).
		Where(fmt.Sprintf("%s = ? AND %s = ? AND status = ? AND last_hit_at >= ?", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode, status, lastHitAfter).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *agentRepository) UpdateLastHit(ctx context.Context, namespaceCode, projectCode, name string) error {
	agent, err := r.FindByName(ctx, namespaceCode, projectCode, name)
	if err != nil {
		return err
	}
	result := r.db.WithContext(ctx).
		Model(&model.Agent{}).
		Where("id = ?", agent.ID).
		UpdateColumn("last_hit_at", r.db.NowFunc())

	return result.Error
}

func (r *agentRepository) Delete(ctx context.Context, namespaceCode, projectCode, name string) error {
	result := r.db.WithContext(ctx).
		Where(fmt.Sprintf("%s = ? AND %s = ? AND name = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode, name).
		Delete(&model.Agent{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
