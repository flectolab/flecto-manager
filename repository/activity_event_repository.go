package repository

import (
	"context"
	"fmt"
	"math"

	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
)

// ProjectRef identifies a project holding activity events.
type ProjectRef struct {
	NamespaceCode string
	ProjectCode   string
}

type ActivityEventRepository interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	Create(ctx context.Context, event *model.ActivityEvent) error
	FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.ActivityEvent, error)
	SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.ActivityEvent, int64, error)
	// FindProjectsWithEvents lists the projects that actually hold events, so a
	// purge only visits those.
	FindProjectsWithEvents(ctx context.Context) ([]ProjectRef, error)
	// FindPurgeCursor returns the id of the keep-th most recent event of a project,
	// everything below it being purgeable. It returns false when the project holds
	// fewer than keep events, meaning nothing to purge.
	FindPurgeCursor(ctx context.Context, project ProjectRef, keep int) (int64, bool, error)
	// DeleteBelow removes the events of a project older than cursor, in batches, and
	// reports how many rows went.
	DeleteBelow(ctx context.Context, project ProjectRef, cursor int64, batchSize int) (int64, error)
	// DeleteByProject removes every event of a project, in batches, and reports how
	// many rows went.
	DeleteByProject(ctx context.Context, project ProjectRef, batchSize int) (int64, error)
}

type activityEventRepository struct {
	db *gorm.DB
}

func NewActivityEventRepository(db *gorm.DB) ActivityEventRepository {
	return &activityEventRepository{db: db}
}

func (r *activityEventRepository) GetTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *activityEventRepository) GetQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.ActivityEvent{})
}

func (r *activityEventRepository) Create(ctx context.Context, event *model.ActivityEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// FindByProject returns the whole journal of a project, most recent first. The
// table is bounded per project by the purge, so this stays small.
func (r *activityEventRepository) FindByProject(ctx context.Context, namespaceCode, projectCode string) ([]model.ActivityEvent, error) {
	var events []model.ActivityEvent
	err := r.db.WithContext(ctx).
		Where(fmt.Sprintf("%s = ? AND %s = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode).
		Order("id DESC").
		Find(&events).Error
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (r *activityEventRepository) SearchPaginate(ctx context.Context, query *gorm.DB, limit, offset int) ([]model.ActivityEvent, int64, error) {
	var total int64
	if query == nil {
		query = r.GetQuery(ctx)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit != 0 {
		query = query.Limit(limit).Offset(offset)
	}

	var events []model.ActivityEvent
	if err := query.Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

func (r *activityEventRepository) FindProjectsWithEvents(ctx context.Context) ([]ProjectRef, error) {
	var projects []ProjectRef
	err := r.db.WithContext(ctx).
		Model(&model.ActivityEvent{}).
		Distinct(model.ColumnNamespaceCode, model.ColumnProjectCode).
		Find(&projects).Error
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func (r *activityEventRepository) FindPurgeCursor(ctx context.Context, project ProjectRef, keep int) (int64, bool, error) {
	if keep <= 0 {
		return 0, false, nil
	}

	var ids []int64
	// OFFSET keep-1 lands on the oldest event worth keeping; everything below it is
	// purgeable. Cheaper than counting, and served by idx_activity_events_ns_proj.
	err := r.db.WithContext(ctx).
		Model(&model.ActivityEvent{}).
		Where(fmt.Sprintf("%s = ? AND %s = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), project.NamespaceCode, project.ProjectCode).
		Order("id DESC").
		Limit(1).
		Offset(keep-1).
		Pluck("id", &ids).Error
	if err != nil {
		return 0, false, err
	}
	if len(ids) == 0 {
		return 0, false, nil
	}
	return ids[0], true, nil
}

func (r *activityEventRepository) DeleteBelow(ctx context.Context, project ProjectRef, cursor int64, batchSize int) (int64, error) {
	if batchSize <= 0 {
		return 0, fmt.Errorf("batch size must be positive")
	}

	where := fmt.Sprintf("%s = ? AND %s = ? AND id < ?", model.ColumnNamespaceCode, model.ColumnProjectCode)

	var deleted int64
	// Selecting the ids first, then deleting them by primary key, rather than a
	// DELETE ... LIMIT: MariaDB honours that LIMIT but SQLite silently ignores it,
	// which would turn the batching into one unbounded statement under test while
	// production batched properly. Batching matters because a project far above its
	// cap would otherwise hold locks over a very large row set.
	for {
		var ids []int64
		err := r.db.WithContext(ctx).
			Model(&model.ActivityEvent{}).
			Where(where, project.NamespaceCode, project.ProjectCode, cursor).
			Order("id").
			Limit(batchSize).
			Pluck("id", &ids).Error
		if err != nil {
			return deleted, err
		}
		if len(ids) == 0 {
			return deleted, nil
		}

		result := r.db.WithContext(ctx).Where("id IN ?", ids).Delete(&model.ActivityEvent{})
		if result.Error != nil {
			return deleted, result.Error
		}
		deleted += result.RowsAffected

		if len(ids) < batchSize {
			return deleted, nil
		}
	}
}

func (r *activityEventRepository) DeleteByProject(ctx context.Context, project ProjectRef, batchSize int) (int64, error) {
	// Reusing DeleteBelow with a cursor above every existing id: same batching, one
	// implementation to keep correct.
	return r.DeleteBelow(ctx, project, math.MaxInt64, batchSize)
}
