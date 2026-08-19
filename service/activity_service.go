package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flectolab/flecto-manager/auth/usercontext"
	commonTypes "github.com/flectolab/flecto-manager/common/types"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/types"
	"gorm.io/gorm"
)

type ActivityService interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	// Record persists an activity event. tx must be the transaction of the recorded
	// operation, so that the event cannot survive a rollback of the change it
	// describes; pass nil to write outside of any transaction.
	Record(ctx context.Context, tx *gorm.DB, in model.ActivityInput) error
	Search(ctx context.Context, query *gorm.DB) ([]model.ActivityEvent, error)
	SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.ActivityEventList, error)
	// Purge trims every project down to activity.max_events_per_project and reports
	// how many events it removed. It is idempotent: running it twice in a row leaves
	// the second run with nothing to do.
	Purge(ctx context.Context) (int64, error)
	// TruncateProject clears the whole journal of one project and reports how many
	// entries went. It records a single entry afterwards, so the journal explains its
	// own emptiness instead of silently losing its history.
	TruncateProject(ctx context.Context, namespaceCode, projectCode string) (int64, error)
}

// activityPurgeBatchSize bounds a single delete statement of the purge.
const activityPurgeBatchSize = 500

type activityService struct {
	ctx  *appContext.Context
	repo repository.ActivityEventRepository
}

func NewActivityService(ctx *appContext.Context, repo repository.ActivityEventRepository) ActivityService {
	return &activityService{
		ctx:  ctx,
		repo: repo,
	}
}

func (s *activityService) GetTx(ctx context.Context) *gorm.DB {
	return s.repo.GetTx(ctx)
}

func (s *activityService) GetQuery(ctx context.Context) *gorm.DB {
	return s.repo.GetQuery(ctx)
}

func (s *activityService) Record(ctx context.Context, tx *gorm.DB, in model.ActivityInput) error {
	event := &model.ActivityEvent{
		NamespaceCode: in.NamespaceCode,
		ProjectCode:   in.ProjectCode,
		Resource:      in.Resource,
		Action:        in.Action,
		ResourceID:    in.ResourceID,
		Actor:         model.ActivityActorSystem,
		OccurredAt:    time.Now(),
	}

	if subject := usercontext.GetUser(ctx); subject != nil {
		event.Actor = subject.Username
		event.AuthType = subject.AuthType
		// API tokens authenticate without a user account and report id 0.
		if subject.IsUser() {
			event.UserID = types.Ptr(subject.UserID)
		}
	}

	if in.Data != nil {
		data, err := json.Marshal(in.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal activity data: %w", err)
		}
		event.Data = data
	}

	if tx != nil {
		return tx.Create(event).Error
	}
	return s.repo.Create(ctx, event)
}

// countDraftsByChangeType breaks the pending drafts of a project down by change
// type in a single query. draftModel is a draft model instance, used by GORM to
// resolve the table.
func countDraftsByChangeType(tx *gorm.DB, draftModel any, namespaceCode, projectCode string) (*model.ActivityDraftCounts, error) {
	var rows []struct {
		ChangeType model.DraftChangeType
		Total      int64
	}

	err := tx.Model(draftModel).
		Select("change_type, COUNT(*) AS total").
		Where(fmt.Sprintf("%s = ? AND %s = ?", model.ColumnNamespaceCode, model.ColumnProjectCode), namespaceCode, projectCode).
		Group("change_type").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	counts := &model.ActivityDraftCounts{}
	for _, row := range rows {
		switch row.ChangeType {
		case model.DraftChangeTypeCreate:
			counts.Create = row.Total
		case model.DraftChangeTypeUpdate:
			counts.Update = row.Total
		case model.DraftChangeTypeDelete:
			counts.Delete = row.Total
		}
	}

	return counts, nil
}

func (s *activityService) Search(ctx context.Context, query *gorm.DB) ([]model.ActivityEvent, error) {
	events, _, err := s.repo.SearchPaginate(ctx, query, 0, 0)
	return events, err
}

func (s *activityService) SearchPaginate(ctx context.Context, pagination *commonTypes.PaginationInput, query *gorm.DB) (*model.ActivityEventList, error) {
	events, total, err := s.repo.SearchPaginate(ctx, query, pagination.GetLimit(), pagination.GetOffset())
	if err != nil {
		return nil, err
	}

	return &model.ActivityEventList{
		Total:  int(total),
		Offset: pagination.GetOffset(),
		Limit:  pagination.GetLimit(),
		Items:  events,
	}, nil
}

func (s *activityService) Purge(ctx context.Context) (int64, error) {
	keep := s.ctx.Config.Activity.MaxEventsPerProject
	if keep <= 0 {
		// No cap configured: an explicit choice to let the table grow.
		return 0, nil
	}

	projects, err := s.repo.FindProjectsWithEvents(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list projects with activity events: %w", err)
	}

	var purged int64
	for _, project := range projects {
		cursor, hasCursor, errCursor := s.repo.FindPurgeCursor(ctx, project, keep)
		if errCursor != nil {
			return purged, fmt.Errorf("failed to find purge cursor for %s/%s: %w", project.NamespaceCode, project.ProjectCode, errCursor)
		}
		// Under the cap, nothing to trim
		if !hasCursor {
			continue
		}

		deleted, errDelete := s.repo.DeleteBelow(ctx, project, cursor, activityPurgeBatchSize)
		purged += deleted
		if errDelete != nil {
			return purged, fmt.Errorf("failed to purge activity events of %s/%s: %w", project.NamespaceCode, project.ProjectCode, errDelete)
		}

		if deleted > 0 {
			s.ctx.Logger.Info("activity events purged",
				"namespace", project.NamespaceCode, "project", project.ProjectCode,
				"deleted", deleted, "kept", keep)
		}
	}

	return purged, nil
}

func (s *activityService) TruncateProject(ctx context.Context, namespaceCode, projectCode string) (int64, error) {
	project := repository.ProjectRef{NamespaceCode: namespaceCode, ProjectCode: projectCode}

	deleted, err := s.repo.DeleteByProject(ctx, project, activityPurgeBatchSize)
	if err != nil {
		return deleted, fmt.Errorf("failed to truncate activity of %s/%s: %w", namespaceCode, projectCode, err)
	}

	// Recorded after the wipe, and deliberately outside any transaction: this entry
	// must survive, it is the only trace left of who cleared the journal.
	errRecord := s.Record(ctx, nil, model.ActivityInput{
		NamespaceCode: namespaceCode,
		ProjectCode:   projectCode,
		Resource:      model.ActivityResourceActivity,
		Action:        model.ActivityActionTruncate,
		Data:          model.ActivityTruncate{Published: deleted},
	})
	if errRecord != nil {
		return deleted, errRecord
	}

	s.ctx.Logger.Info("activity truncated", "namespace", namespaceCode, "project", projectCode, "deleted", deleted)
	return deleted, nil
}
