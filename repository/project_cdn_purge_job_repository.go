package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
)

type ProjectCdnPurgeJobRepository interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	Create(ctx context.Context, job *model.ProjectCdnPurgeJob) error
	Update(ctx context.Context, job *model.ProjectCdnPurgeJob) error
	// UpdateProgress saves only how far a job got. It is called after every batch,
	// where Update would rewrite the whole target list each time.
	UpdateProgress(ctx context.Context, job *model.ProjectCdnPurgeJob) error
	FindByID(ctx context.Context, id int64) (*model.ProjectCdnPurgeJob, error)
	// FindMergeable returns the job a new publish can fold itself into: one still
	// waiting for the agents of that project, on that configuration. Merging avoids
	// queueing a second purge behind a first one that has not even started.
	FindMergeable(ctx context.Context, tx *gorm.DB, cdnConfigID int64) (*model.ProjectCdnPurgeJob, error)
	// MergeWaiting saves a job a publish folded itself into, provided it is still
	// waiting at the version it was read at. It reports false when a worker started
	// the job, or another publish merged into it, in the meantime: the caller then
	// queues a job of its own rather than adding to batches already on their way.
	MergeWaiting(ctx context.Context, tx *gorm.DB, job *model.ProjectCdnPurgeJob, readVersion int) (bool, error)
	// MarkRunning moves a waiting job to running, provided it still covers the
	// version its agents were checked against. It reports false when a publish
	// merged into it since it was read: the in-memory targets are then stale, and
	// the job is looked at again on the next run.
	MarkRunning(ctx context.Context, job *model.ProjectCdnPurgeJob, startedAt time.Time, waitedForAgents bool) (bool, error)
	// FindActive returns the jobs a worker has to look at: up to limit running jobs,
	// then up to limit waiting ones, oldest first within each. Running jobs are read
	// apart so a pile of jobs waiting on lagging agents, in any project, cannot keep
	// the ones ready to send out of a run.
	FindActive(ctx context.Context, limit int) ([]model.ProjectCdnPurgeJob, error)
	FindByProject(ctx context.Context, namespaceCode, projectCode string, limit int) ([]model.ProjectCdnPurgeJob, error)
	// DeleteFinishedBefore drops the finished jobs older than the cutoff and reports
	// how many went.
	DeleteFinishedBefore(ctx context.Context, cutoff time.Time, batchSize int) (int64, error)
}

type projectCdnPurgeJobRepository struct {
	db *gorm.DB
}

func NewProjectCdnPurgeJobRepository(db *gorm.DB) ProjectCdnPurgeJobRepository {
	return &projectCdnPurgeJobRepository{db: db}
}

func (r *projectCdnPurgeJobRepository) GetTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

func (r *projectCdnPurgeJobRepository) GetQuery(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Model(&model.ProjectCdnPurgeJob{})
}

func (r *projectCdnPurgeJobRepository) Create(ctx context.Context, job *model.ProjectCdnPurgeJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *projectCdnPurgeJobRepository) Update(ctx context.Context, job *model.ProjectCdnPurgeJob) error {
	return r.db.WithContext(ctx).Save(job).Error
}

func (r *projectCdnPurgeJobRepository) UpdateProgress(ctx context.Context, job *model.ProjectCdnPurgeJob) error {
	job.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).
		Model(&model.ProjectCdnPurgeJob{}).
		Where("id = ?", job.ID).
		Updates(map[string]any{
			"cursor":     job.Cursor,
			"purged":     job.Purged,
			"failed":     job.Failed,
			"attempts":   job.Attempts,
			"error":      job.Error,
			"updated_at": job.UpdatedAt,
		}).Error
}

func (r *projectCdnPurgeJobRepository) FindByID(ctx context.Context, id int64) (*model.ProjectCdnPurgeJob, error) {
	var job model.ProjectCdnPurgeJob
	if err := r.db.WithContext(ctx).First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// FindMergeable runs inside the publish transaction when one is given, so the job
// it returns cannot be picked up by a worker between the read and the merge.
func (r *projectCdnPurgeJobRepository) FindMergeable(ctx context.Context, tx *gorm.DB, cdnConfigID int64) (*model.ProjectCdnPurgeJob, error) {
	query := tx
	if query == nil {
		query = r.db.WithContext(ctx)
	}

	var job model.ProjectCdnPurgeJob
	err := query.
		Where("cdn_config_id = ? AND status = ?", cdnConfigID, model.CdnPurgeStatusWaitingAgents).
		Order("id").
		First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *projectCdnPurgeJobRepository) MergeWaiting(ctx context.Context, tx *gorm.DB, job *model.ProjectCdnPurgeJob, readVersion int) (bool, error) {
	query := tx
	if query == nil {
		query = r.db.WithContext(ctx)
	}

	// A struct rather than a map: only a struct goes through the JSON serializer of
	// the targets and the skip counters.
	job.UpdatedAt = time.Now()
	result := query.
		Model(&model.ProjectCdnPurgeJob{}).
		Where("id = ? AND status = ? AND project_version = ?", job.ID, model.CdnPurgeStatusWaitingAgents, readVersion).
		Select("targets", "requested", "skipped", "project_version", "deadline_at", "updated_at").
		Updates(job)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (r *projectCdnPurgeJobRepository) MarkRunning(ctx context.Context, job *model.ProjectCdnPurgeJob, startedAt time.Time, waitedForAgents bool) (bool, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.ProjectCdnPurgeJob{}).
		Where("id = ? AND status = ? AND project_version = ?", job.ID, model.CdnPurgeStatusWaitingAgents, job.ProjectVersion).
		Updates(map[string]any{
			"status":            model.CdnPurgeStatusRunning,
			"started_at":        startedAt,
			"waited_for_agents": waitedForAgents,
			"updated_at":        now,
		})
	if result.Error != nil || result.RowsAffected != 1 {
		return false, result.Error
	}

	job.Status = model.CdnPurgeStatusRunning
	job.StartedAt = &startedAt
	job.WaitedForAgents = waitedForAgents
	job.UpdatedAt = now
	return true, nil
}

func (r *projectCdnPurgeJobRepository) FindActive(ctx context.Context, limit int) ([]model.ProjectCdnPurgeJob, error) {
	running, err := r.findByStatus(ctx, model.CdnPurgeStatusRunning, limit)
	if err != nil {
		return nil, err
	}
	waiting, err := r.findByStatus(ctx, model.CdnPurgeStatusWaitingAgents, limit)
	if err != nil {
		return nil, err
	}
	return append(running, waiting...), nil
}

func (r *projectCdnPurgeJobRepository) findByStatus(ctx context.Context, status model.CdnPurgeStatus, limit int) ([]model.ProjectCdnPurgeJob, error) {
	var jobs []model.ProjectCdnPurgeJob
	query := r.db.WithContext(ctx).Where("status = ?", status).Order("id")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *projectCdnPurgeJobRepository) FindByProject(ctx context.Context, namespaceCode, projectCode string, limit int) ([]model.ProjectCdnPurgeJob, error) {
	var jobs []model.ProjectCdnPurgeJob
	query := r.db.WithContext(ctx).
		Where(fmt.Sprintf("%s = ? AND %s = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode).
		Order("id DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *projectCdnPurgeJobRepository) DeleteFinishedBefore(ctx context.Context, cutoff time.Time, batchSize int) (int64, error) {
	var total int64

	for {
		query := r.db.WithContext(ctx).
			Where("finished_at IS NOT NULL AND finished_at < ?", cutoff)
		if batchSize > 0 {
			query = query.Limit(batchSize)
		}

		result := query.Delete(&model.ProjectCdnPurgeJob{})
		if result.Error != nil {
			return total, result.Error
		}

		total += result.RowsAffected
		if batchSize <= 0 || result.RowsAffected < int64(batchSize) {
			return total, nil
		}
	}
}
