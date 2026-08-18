package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/auth/usercontext"
	commonTypes "github.com/flectolab/flecto-manager/common/types"
	appContext "github.com/flectolab/flecto-manager/context"
	mockFlectoRepository "github.com/flectolab/flecto-manager/mocks/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupActivityServiceTest(t *testing.T) (*gomock.Controller, *mockFlectoRepository.MockActivityEventRepository, ActivityService) {
	ctrl := gomock.NewController(t)
	mockRepo := mockFlectoRepository.NewMockActivityEventRepository(ctrl)
	svc := NewActivityService(appContext.TestContext(nil), mockRepo)
	return ctrl, mockRepo, svc
}

// setupActivityServiceDB returns a service backed by a real in-memory database, for
// the cases where the transactional behaviour is what is under test.
func setupActivityServiceDB(t *testing.T) (*gorm.DB, ActivityService) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&model.ActivityEvent{}))

	return db, NewActivityService(appContext.TestContext(nil), repository.NewActivityEventRepository(db))
}

// newTestActivityService gives the services that emit activity events a real activity
// service writing to their own test database, so their tests can assert on the
// rows actually written. Pass a nil db when the service under test never records.
func newTestActivityService(t *testing.T, db *gorm.DB) ActivityService {
	t.Helper()

	if db == nil {
		return NewActivityService(appContext.TestContext(nil), nil)
	}

	assert.NoError(t, db.AutoMigrate(&model.ActivityEvent{}))
	return NewActivityService(appContext.TestContext(nil), repository.NewActivityEventRepository(db))
}

// lastActivityEvent returns the most recent activity event of a test database, nil when
// none was recorded.
func lastActivityEvent(t *testing.T, db *gorm.DB) *model.ActivityEvent {
	t.Helper()

	var events []model.ActivityEvent
	assert.NoError(t, db.Order("id DESC").Limit(1).Find(&events).Error)
	if len(events) == 0 {
		return nil
	}
	return &events[0]
}

// countActivityEvents returns how many activity events a test database holds.
func countActivityEvents(t *testing.T, db *gorm.DB) int64 {
	t.Helper()

	var count int64
	assert.NoError(t, db.Model(&model.ActivityEvent{}).Count(&count).Error)
	return count
}

func TestNewActivityService(t *testing.T) {
	ctrl, mockRepo, svc := setupActivityServiceTest(t)
	defer ctrl.Finish()

	assert.NotNil(t, svc)
	assert.NotNil(t, mockRepo)
}

func TestActivityService_Record_Actor(t *testing.T) {
	baseInput := model.ActivityInput{
		NamespaceCode: "ns",
		ProjectCode:   "proj",
		Resource:      model.ActivityResourceRedirect,
		Action:        model.ActivityActionCreate,
	}

	tests := []struct {
		name         string
		subject      *usercontext.UserContext
		wantActor    string
		wantUserID   *int64
		wantAuthType types.AuthType
	}{
		{
			name: "authenticated user",
			subject: &usercontext.UserContext{
				UserID:   42,
				Username: "alice",
				AuthType: types.AuthTypeBasic,
			},
			wantActor:    "alice",
			wantUserID:   types.Ptr(int64(42)),
			wantAuthType: types.AuthTypeBasic,
		},
		{
			name: "openid user",
			subject: &usercontext.UserContext{
				UserID:   7,
				Username: "bob",
				AuthType: types.AuthTypeOpenID,
			},
			wantActor:    "bob",
			wantUserID:   types.Ptr(int64(7)),
			wantAuthType: types.AuthTypeOpenID,
		},
		{
			// API tokens authenticate without a user account and report id 0,
			// which must not be stored as a user reference.
			name: "api token",
			subject: &usercontext.UserContext{
				UserID:   0,
				Username: "ci-token",
				AuthType: types.AuthTypeToken,
			},
			wantActor:    "ci-token",
			wantUserID:   nil,
			wantAuthType: types.AuthTypeToken,
		},
		{
			name:       "no subject in context",
			subject:    nil,
			wantActor:  model.ActivityActorSystem,
			wantUserID: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl, mockRepo, svc := setupActivityServiceTest(t)
			defer ctrl.Finish()

			ctx := context.Background()
			if tt.subject != nil {
				ctx = usercontext.SetUserContext(ctx, tt.subject)
			}

			var recorded *model.ActivityEvent
			mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, event *model.ActivityEvent) error {
					recorded = event
					return nil
				})

			assert.NoError(t, svc.Record(ctx, nil, baseInput))

			assert.Equal(t, tt.wantActor, recorded.Actor)
			assert.Equal(t, tt.wantAuthType, recorded.AuthType)
			if tt.wantUserID == nil {
				assert.Nil(t, recorded.UserID)
			} else {
				assert.Equal(t, *tt.wantUserID, *recorded.UserID)
			}
			assert.False(t, recorded.OccurredAt.IsZero())
		})
	}
}

func TestActivityService_Record_Payload(t *testing.T) {
	t.Run("marshals the payload", func(t *testing.T) {
		ctrl, mockRepo, svc := setupActivityServiceTest(t)
		defer ctrl.Finish()

		var recorded *model.ActivityEvent
		mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, event *model.ActivityEvent) error {
				recorded = event
				return nil
			})

		err := svc.Record(context.Background(), nil, model.ActivityInput{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			Resource:      model.ActivityResourceRedirect,
			Action:        model.ActivityActionUpdate,
			ResourceID:    types.Ptr(int64(12)),
			Data: model.ActivityChange[model.RedirectSnapshot]{
				Before: &model.RedirectSnapshot{Source: "/a", Target: "/b"},
				After:  &model.RedirectSnapshot{Source: "/a", Target: "/c"},
			},
		})
		assert.NoError(t, err)

		assert.Equal(t, int64(12), *recorded.ResourceID)
		assert.JSONEq(t,
			`{"before":{"type":"","source":"/a","target":"/b","status":""},"after":{"type":"","source":"/a","target":"/c","status":""}}`,
			string(recorded.Data),
		)
	})

	t.Run("nil payload leaves data empty", func(t *testing.T) {
		ctrl, mockRepo, svc := setupActivityServiceTest(t)
		defer ctrl.Finish()

		var recorded *model.ActivityEvent
		mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, event *model.ActivityEvent) error {
				recorded = event
				return nil
			})

		assert.NoError(t, svc.Record(context.Background(), nil, model.ActivityInput{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			Resource:      model.ActivityResourceProject,
			Action:        model.ActivityActionPublish,
		}))
		assert.Empty(t, recorded.Data)
	})

	t.Run("unmarshallable payload is reported", func(t *testing.T) {
		ctrl, _, svc := setupActivityServiceTest(t)
		defer ctrl.Finish()

		err := svc.Record(context.Background(), nil, model.ActivityInput{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			Resource:      model.ActivityResourceProject,
			Action:        model.ActivityActionPublish,
			Data:          math.Inf(1),
		})
		assert.ErrorContains(t, err, "failed to marshal activity data")
	})

	t.Run("page payload never carries the content", func(t *testing.T) {
		ctrl, mockRepo, svc := setupActivityServiceTest(t)
		defer ctrl.Finish()

		var recorded *model.ActivityEvent
		mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, event *model.ActivityEvent) error {
				recorded = event
				return nil
			})

		secret := strings.Repeat("x", 2048)
		page := &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			Path:        "/robots.txt",
			Content:     secret,
			ContentType: commonTypes.PageContentTypeTextPlain,
		}

		assert.NoError(t, svc.Record(context.Background(), nil, model.ActivityInput{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			Resource:      model.ActivityResourcePage,
			Action:        model.ActivityActionCreate,
			Data:          model.ActivityChange[model.PageSnapshot]{After: model.NewPageSnapshot(page, int64(len(secret)))},
		}))

		assert.NotContains(t, string(recorded.Data), secret)
		assert.NotContains(t, string(recorded.Data), "content\"")
		assert.Contains(t, string(recorded.Data), `"contentSize":2048`)
	})
}

func TestActivityService_Record_Transaction(t *testing.T) {
	input := model.ActivityInput{
		NamespaceCode: "ns",
		ProjectCode:   "proj",
		Resource:      model.ActivityResourceRedirect,
		Action:        model.ActivityActionCreate,
	}

	t.Run("writes through the given transaction", func(t *testing.T) {
		db, svc := setupActivityServiceDB(t)

		err := db.Transaction(func(tx *gorm.DB) error {
			return svc.Record(context.Background(), tx, input)
		})
		assert.NoError(t, err)

		var count int64
		assert.NoError(t, db.Model(&model.ActivityEvent{}).Count(&count).Error)
		assert.Equal(t, int64(1), count)
	})

	t.Run("event is rolled back with the recorded operation", func(t *testing.T) {
		db, svc := setupActivityServiceDB(t)

		errBoom := errors.New("boom")
		err := db.Transaction(func(tx *gorm.DB) error {
			if errRecord := svc.Record(context.Background(), tx, input); errRecord != nil {
				return errRecord
			}
			return errBoom
		})
		assert.ErrorIs(t, err, errBoom)

		var count int64
		assert.NoError(t, db.Model(&model.ActivityEvent{}).Count(&count).Error)
		assert.Zero(t, count, "an activity event must not survive the rollback of the change it describes")
	})

	t.Run("repository error is propagated", func(t *testing.T) {
		ctrl, mockRepo, svc := setupActivityServiceTest(t)
		defer ctrl.Finish()

		errDB := errors.New("db down")
		mockRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errDB)

		assert.ErrorIs(t, svc.Record(context.Background(), nil, input), errDB)
	})
}

func TestActivityService_Search(t *testing.T) {
	ctrl, mockRepo, svc := setupActivityServiceTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	expected := []model.ActivityEvent{{ID: 1}, {ID: 2}}
	mockRepo.EXPECT().SearchPaginate(ctx, nil, 0, 0).Return(expected, int64(2), nil)

	events, err := svc.Search(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, expected, events)
}

func TestActivityService_SearchPaginate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockRepo, svc := setupActivityServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		pagination := &commonTypes.PaginationInput{Limit: types.Ptr(10), Offset: types.Ptr(20)}
		expected := []model.ActivityEvent{{ID: 1}}
		mockRepo.EXPECT().SearchPaginate(ctx, nil, 10, 20).Return(expected, int64(31), nil)

		list, err := svc.SearchPaginate(ctx, pagination, nil)
		assert.NoError(t, err)
		assert.Equal(t, 31, list.Total)
		assert.Equal(t, 10, list.Limit)
		assert.Equal(t, 20, list.Offset)
		assert.Equal(t, expected, list.Items)
	})

	t.Run("repository error", func(t *testing.T) {
		ctrl, mockRepo, svc := setupActivityServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		errDB := errors.New("db down")
		mockRepo.EXPECT().SearchPaginate(ctx, nil, gomock.Any(), gomock.Any()).Return(nil, int64(0), errDB)

		list, err := svc.SearchPaginate(ctx, &commonTypes.PaginationInput{}, nil)
		assert.ErrorIs(t, err, errDB)
		assert.Nil(t, list)
	})
}

func TestActivityService_GetTxAndGetQuery(t *testing.T) {
	db, svc := setupActivityServiceDB(t)
	ctx := context.Background()

	assert.NotNil(t, svc.GetTx(ctx))
	assert.NotNil(t, svc.GetQuery(ctx))

	var count int64
	assert.NoError(t, svc.GetQuery(ctx).Count(&count).Error)
	assert.Zero(t, count)
	assert.NotNil(t, db)
}

// setupActivityPurgeService returns a service with a real database and the given
// per-project cap.
func setupActivityPurgeService(t *testing.T, maxEventsPerProject int) (*gorm.DB, ActivityService) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&model.ActivityEvent{}))

	appCtx := appContext.TestContext(nil)
	appCtx.Config.Activity.MaxEventsPerProject = maxEventsPerProject

	return db, NewActivityService(appCtx, repository.NewActivityEventRepository(db))
}

func seedEvents(t *testing.T, db *gorm.DB, namespaceCode, projectCode string, n int) {
	t.Helper()

	for i := 0; i < n; i++ {
		assert.NoError(t, db.Create(&model.ActivityEvent{
			NamespaceCode: namespaceCode,
			ProjectCode:   projectCode,
			Resource:      model.ActivityResourceRedirect,
			Action:        model.ActivityActionCreate,
			Actor:         "alice",
			OccurredAt:    time.Now(),
		}).Error)
	}
}

func countEventsOf(t *testing.T, db *gorm.DB, namespaceCode, projectCode string) int64 {
	t.Helper()

	var count int64
	assert.NoError(t, db.Model(&model.ActivityEvent{}).
		Where("namespace_code = ? AND project_code = ?", namespaceCode, projectCode).
		Count(&count).Error)
	return count
}

func TestActivityService_Purge(t *testing.T) {
	t.Run("trims a project down to the cap", func(t *testing.T) {
		db, svc := setupActivityPurgeService(t, 5)
		seedEvents(t, db, "ns", "proj", 12)

		purged, err := svc.Purge(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, int64(7), purged)
		assert.Equal(t, int64(5), countEventsOf(t, db, "ns", "proj"))
	})

	t.Run("keeps the most recent events", func(t *testing.T) {
		db, svc := setupActivityPurgeService(t, 3)
		seedEvents(t, db, "ns", "proj", 6)

		var expected []int64
		assert.NoError(t, db.Model(&model.ActivityEvent{}).Order("id DESC").Limit(3).Pluck("id", &expected).Error)

		_, err := svc.Purge(context.Background())
		assert.NoError(t, err)

		var remaining []int64
		assert.NoError(t, db.Model(&model.ActivityEvent{}).Order("id DESC").Pluck("id", &remaining).Error)
		assert.Equal(t, expected, remaining, "the purge must keep the newest events, not the oldest")
	})

	t.Run("leaves a project under the cap alone", func(t *testing.T) {
		db, svc := setupActivityPurgeService(t, 10)
		seedEvents(t, db, "ns", "proj", 4)

		purged, err := svc.Purge(context.Background())
		assert.NoError(t, err)
		assert.Zero(t, purged)
		assert.Equal(t, int64(4), countEventsOf(t, db, "ns", "proj"))
	})

	t.Run("caps each project independently", func(t *testing.T) {
		db, svc := setupActivityPurgeService(t, 2)
		seedEvents(t, db, "ns", "busy", 7)
		seedEvents(t, db, "ns", "quiet", 1)
		seedEvents(t, db, "other", "busy", 5)

		purged, err := svc.Purge(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, int64(8), purged)
		assert.Equal(t, int64(2), countEventsOf(t, db, "ns", "busy"))
		assert.Equal(t, int64(1), countEventsOf(t, db, "ns", "quiet"))
		assert.Equal(t, int64(2), countEventsOf(t, db, "other", "busy"))
	})

	t.Run("is idempotent", func(t *testing.T) {
		db, svc := setupActivityPurgeService(t, 4)
		seedEvents(t, db, "ns", "proj", 9)

		first, err := svc.Purge(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, int64(5), first)

		// The scheduler runs this hourly: a second pass must find nothing to do
		second, err := svc.Purge(context.Background())
		assert.NoError(t, err)
		assert.Zero(t, second)
		assert.Equal(t, int64(4), countEventsOf(t, db, "ns", "proj"))
	})

	t.Run("a cap of zero disables the purge", func(t *testing.T) {
		db, svc := setupActivityPurgeService(t, 0)
		seedEvents(t, db, "ns", "proj", 20)

		purged, err := svc.Purge(context.Background())
		assert.NoError(t, err)
		assert.Zero(t, purged)
		assert.Equal(t, int64(20), countEventsOf(t, db, "ns", "proj"), "an unset cap is an explicit choice to keep everything")
	})

	t.Run("empty table", func(t *testing.T) {
		db, svc := setupActivityPurgeService(t, 5)

		purged, err := svc.Purge(context.Background())
		assert.NoError(t, err)
		assert.Zero(t, purged)
		assert.Zero(t, countActivityEvents(t, db))
	})

	t.Run("purges more than one batch", func(t *testing.T) {
		db, svc := setupActivityPurgeService(t, 1)
		// Above activityPurgeBatchSize so the batching loop runs several rounds
		seedEvents(t, db, "ns", "proj", activityPurgeBatchSize+50)

		purged, err := svc.Purge(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, int64(activityPurgeBatchSize+49), purged)
		assert.Equal(t, int64(1), countEventsOf(t, db, "ns", "proj"))
	})

	t.Run("reports a repository failure", func(t *testing.T) {
		ctrl, mockRepo, svc := setupActivityServiceTest(t)
		defer ctrl.Finish()

		errDB := errors.New("db down")
		mockRepo.EXPECT().FindProjectsWithEvents(gomock.Any()).Return(nil, errDB)

		svcCtx := svc.(*activityService)
		svcCtx.ctx.Config.Activity.MaxEventsPerProject = 5

		purged, err := svc.Purge(context.Background())
		assert.ErrorIs(t, err, errDB)
		assert.Zero(t, purged)
	})
}

func TestActivityService_TruncateProject(t *testing.T) {
	t.Run("clears the project and records who did it", func(t *testing.T) {
		db, svc := setupActivityPurgeService(t, 0)
		seedEvents(t, db, "ns", "proj", 12)

		ctx := usercontext.SetUserContext(context.Background(), &usercontext.UserContext{
			UserID: 42, Username: "alice", AuthType: types.AuthTypeBasic,
		})

		deleted, err := svc.TruncateProject(ctx, "ns", "proj")
		assert.NoError(t, err)
		assert.Equal(t, int64(12), deleted)

		// One entry left: the trace of the wipe itself, so the journal explains its
		// own emptiness
		assert.Equal(t, int64(1), countEventsOf(t, db, "ns", "proj"))

		event := lastActivityEvent(t, db)
		assert.Equal(t, model.ActivityResourceActivity, event.Resource)
		assert.Equal(t, model.ActivityActionTruncate, event.Action)
		assert.Equal(t, "alice", event.Actor)
		assert.JSONEq(t, `{"published":12,"drafts":0}`, string(event.Data))
	})

	t.Run("leaves other projects alone", func(t *testing.T) {
		db, svc := setupActivityPurgeService(t, 0)
		seedEvents(t, db, "ns", "proj", 5)
		seedEvents(t, db, "ns", "other", 3)

		_, err := svc.TruncateProject(context.Background(), "ns", "proj")
		assert.NoError(t, err)

		assert.Equal(t, int64(1), countEventsOf(t, db, "ns", "proj"), "only the wipe entry")
		assert.Equal(t, int64(3), countEventsOf(t, db, "ns", "other"))
	})

	t.Run("an already empty journal still records the wipe", func(t *testing.T) {
		db, svc := setupActivityPurgeService(t, 0)

		deleted, err := svc.TruncateProject(context.Background(), "ns", "proj")
		assert.NoError(t, err)
		assert.Zero(t, deleted)
		assert.Equal(t, int64(1), countEventsOf(t, db, "ns", "proj"))
	})

	t.Run("truncates more than one batch", func(t *testing.T) {
		db, svc := setupActivityPurgeService(t, 0)
		seedEvents(t, db, "ns", "proj", activityPurgeBatchSize+40)

		deleted, err := svc.TruncateProject(context.Background(), "ns", "proj")
		assert.NoError(t, err)
		assert.Equal(t, int64(activityPurgeBatchSize+40), deleted)
	})

	t.Run("reports a repository failure", func(t *testing.T) {
		ctrl, mockRepo, svc := setupActivityServiceTest(t)
		defer ctrl.Finish()

		errDB := errors.New("db down")
		mockRepo.EXPECT().DeleteByProject(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(0), errDB)

		_, err := svc.TruncateProject(context.Background(), "ns", "proj")
		assert.ErrorIs(t, err, errDB)
	})
}
