package repository

import (
	"context"
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupActivityEventTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.User{}, &model.ActivityEvent{})
	assert.NoError(t, err)

	return db
}

func createTestActivityEvent(t *testing.T, db *gorm.DB, namespaceCode, projectCode string, action model.ActivityAction, occurredAt time.Time) *model.ActivityEvent {
	event := &model.ActivityEvent{
		NamespaceCode: namespaceCode,
		ProjectCode:   projectCode,
		Resource:      model.ActivityResourceRedirect,
		Action:        action,
		Actor:         "alice",
		AuthType:      types.AuthTypeBasic,
		OccurredAt:    occurredAt,
	}
	err := db.Create(event).Error
	assert.NoError(t, err)
	return event
}

func TestNewActivityEventRepository(t *testing.T) {
	db := setupActivityEventTestDB(t)
	repo := NewActivityEventRepository(db)

	assert.NotNil(t, repo)
}

func TestActivityEventRepository_GetTx(t *testing.T) {
	db := setupActivityEventTestDB(t)
	repo := NewActivityEventRepository(db)

	tx := repo.GetTx(context.Background())
	assert.NotNil(t, tx)

	var events []model.ActivityEvent
	assert.NoError(t, tx.Find(&events).Error)
}

func TestActivityEventRepository_GetQuery(t *testing.T) {
	db := setupActivityEventTestDB(t)
	repo := NewActivityEventRepository(db)

	query := repo.GetQuery(context.Background())
	assert.NotNil(t, query)

	var events []model.ActivityEvent
	assert.NoError(t, query.Find(&events).Error)
}

func TestActivityEventRepository_Create(t *testing.T) {
	t.Run("success with full payload", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)
		ctx := context.Background()

		occurredAt := time.Now().UTC().Truncate(time.Second)
		event := &model.ActivityEvent{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			Resource:      model.ActivityResourceRedirect,
			Action:        model.ActivityActionCreate,
			UserID:        types.Ptr(int64(42)),
			Actor:         "alice",
			AuthType:      types.AuthTypeBasic,
			ResourceID:    types.Ptr(int64(7)),
			Data:          model.ActivityData(`{"after":{"source":"/a"}}`),
			OccurredAt:    occurredAt,
		}

		assert.NoError(t, repo.Create(ctx, event))
		assert.NotZero(t, event.ID)

		var stored model.ActivityEvent
		assert.NoError(t, db.First(&stored, event.ID).Error)
		assert.Equal(t, "ns", stored.NamespaceCode)
		assert.Equal(t, model.ActivityResourceRedirect, stored.Resource)
		assert.Equal(t, model.ActivityActionCreate, stored.Action)
		assert.Equal(t, int64(42), *stored.UserID)
		assert.Equal(t, "alice", stored.Actor)
		assert.Equal(t, types.AuthTypeBasic, stored.AuthType)
		assert.Equal(t, int64(7), *stored.ResourceID)
		// Data must come back as the exact JSON that was written, not re-encoded.
		assert.JSONEq(t, `{"after":{"source":"/a"}}`, string(stored.Data))
		assert.Equal(t, occurredAt.Unix(), stored.OccurredAt.Unix())
	})

	t.Run("success without user and without data", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)
		ctx := context.Background()

		event := &model.ActivityEvent{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			Resource:      model.ActivityResourceProject,
			Action:        model.ActivityActionPublish,
			Actor:         model.ActivityActorSystem,
			OccurredAt:    time.Now(),
		}

		assert.NoError(t, repo.Create(ctx, event))

		var stored model.ActivityEvent
		assert.NoError(t, db.First(&stored, event.ID).Error)
		assert.Nil(t, stored.UserID)
		assert.Nil(t, stored.ResourceID)
		assert.Empty(t, stored.Data)
	})
}

func TestActivityEventRepository_FindByProject(t *testing.T) {
	t.Run("returns only the project events, most recent first", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)
		ctx := context.Background()

		now := time.Now()
		first := createTestActivityEvent(t, db, "ns", "proj", model.ActivityActionCreate, now.Add(-2*time.Hour))
		second := createTestActivityEvent(t, db, "ns", "proj", model.ActivityActionUpdate, now.Add(-time.Hour))
		createTestActivityEvent(t, db, "ns", "other", model.ActivityActionCreate, now)
		createTestActivityEvent(t, db, "other", "proj", model.ActivityActionCreate, now)

		events, err := repo.FindByProject(ctx, "ns", "proj")
		assert.NoError(t, err)
		assert.Len(t, events, 2)
		assert.Equal(t, second.ID, events[0].ID)
		assert.Equal(t, first.ID, events[1].ID)
	})

	t.Run("empty project", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)

		events, err := repo.FindByProject(context.Background(), "ns", "proj")
		assert.NoError(t, err)
		assert.Empty(t, events)
	})
}

func TestActivityEventRepository_SearchPaginate(t *testing.T) {
	t.Run("paginates and reports the full total", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)
		ctx := context.Background()

		now := time.Now()
		for i := 0; i < 5; i++ {
			createTestActivityEvent(t, db, "ns", "proj", model.ActivityActionCreate, now.Add(time.Duration(i)*time.Minute))
		}

		query := repo.GetQuery(ctx).Order("id DESC")
		events, total, err := repo.SearchPaginate(ctx, query, 2, 1)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, events, 2)
	})

	t.Run("nil query falls back to the whole table", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)
		ctx := context.Background()

		createTestActivityEvent(t, db, "ns", "proj", model.ActivityActionCreate, time.Now())

		events, total, err := repo.SearchPaginate(ctx, nil, 0, 0)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, events, 1)
	})

	t.Run("filtered query", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)
		ctx := context.Background()

		now := time.Now()
		createTestActivityEvent(t, db, "ns", "proj", model.ActivityActionCreate, now)
		createTestActivityEvent(t, db, "ns", "proj", model.ActivityActionPublish, now)

		query := repo.GetQuery(ctx).Where("action = ?", model.ActivityActionPublish)
		events, total, err := repo.SearchPaginate(ctx, query, 10, 0)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, events, 1)
		assert.Equal(t, model.ActivityActionPublish, events[0].Action)
	})

	t.Run("empty table", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)
		ctx := context.Background()

		events, total, err := repo.SearchPaginate(ctx, repo.GetQuery(ctx), 10, 0)
		assert.NoError(t, err)
		assert.Zero(t, total)
		assert.Empty(t, events)
	})
}

// seedActivityEvents inserts n events for a project and returns their ids, oldest first.
func seedActivityEvents(t *testing.T, db *gorm.DB, namespaceCode, projectCode string, n int) []int64 {
	t.Helper()

	ids := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		event := createTestActivityEvent(t, db, namespaceCode, projectCode, model.ActivityActionCreate, time.Now())
		ids = append(ids, event.ID)
	}
	return ids
}

func TestActivityEventRepository_FindProjectsWithEvents(t *testing.T) {
	t.Run("lists each project once", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)

		seedActivityEvents(t, db, "ns1", "proj1", 3)
		seedActivityEvents(t, db, "ns1", "proj2", 1)
		seedActivityEvents(t, db, "ns2", "proj1", 2)

		projects, err := repo.FindProjectsWithEvents(context.Background())
		assert.NoError(t, err)
		assert.ElementsMatch(t, []ProjectRef{
			{NamespaceCode: "ns1", ProjectCode: "proj1"},
			{NamespaceCode: "ns1", ProjectCode: "proj2"},
			{NamespaceCode: "ns2", ProjectCode: "proj1"},
		}, projects)
	})

	t.Run("empty table", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)

		projects, err := repo.FindProjectsWithEvents(context.Background())
		assert.NoError(t, err)
		assert.Empty(t, projects)
	})
}

func TestActivityEventRepository_FindPurgeCursor(t *testing.T) {
	t.Run("returns the oldest event worth keeping", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)

		ids := seedActivityEvents(t, db, "ns", "proj", 5)

		// Keeping 2 means the cursor is the 2nd most recent event
		cursor, found, err := repo.FindPurgeCursor(context.Background(), ProjectRef{"ns", "proj"}, 2)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, ids[3], cursor)
	})

	t.Run("no cursor when the project is under the cap", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)

		seedActivityEvents(t, db, "ns", "proj", 3)

		cursor, found, err := repo.FindPurgeCursor(context.Background(), ProjectRef{"ns", "proj"}, 10)
		assert.NoError(t, err)
		assert.False(t, found, "nothing to purge below the cap")
		assert.Zero(t, cursor)
	})

	t.Run("no cursor when exactly at the cap", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)

		ids := seedActivityEvents(t, db, "ns", "proj", 3)

		// The cursor is the oldest kept event, and nothing sits below it
		cursor, found, err := repo.FindPurgeCursor(context.Background(), ProjectRef{"ns", "proj"}, 3)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, ids[0], cursor)
	})

	t.Run("ignores other projects", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)

		seedActivityEvents(t, db, "ns", "other", 10)
		seedActivityEvents(t, db, "ns", "proj", 2)

		_, found, err := repo.FindPurgeCursor(context.Background(), ProjectRef{"ns", "proj"}, 5)
		assert.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("keep of zero purges nothing", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)

		seedActivityEvents(t, db, "ns", "proj", 3)

		_, found, err := repo.FindPurgeCursor(context.Background(), ProjectRef{"ns", "proj"}, 0)
		assert.NoError(t, err)
		assert.False(t, found)
	})
}

func TestActivityEventRepository_DeleteBelow(t *testing.T) {
	t.Run("deletes only below the cursor and only for that project", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)

		ids := seedActivityEvents(t, db, "ns", "proj", 5)
		otherIDs := seedActivityEvents(t, db, "ns", "other", 3)

		deleted, err := repo.DeleteBelow(context.Background(), ProjectRef{"ns", "proj"}, ids[2], 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), deleted)

		var remaining []int64
		assert.NoError(t, db.Model(&model.ActivityEvent{}).Where("namespace_code = ? AND project_code = ?", "ns", "proj").Order("id").Pluck("id", &remaining).Error)
		assert.Equal(t, ids[2:], remaining)

		var untouched int64
		assert.NoError(t, db.Model(&model.ActivityEvent{}).Where("namespace_code = ? AND project_code = ?", "ns", "other").Count(&untouched).Error)
		assert.Equal(t, int64(len(otherIDs)), untouched)
	})

	t.Run("batches until exhausted", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)

		ids := seedActivityEvents(t, db, "ns", "proj", 10)

		// A batch smaller than the work forces several rounds; the count must still
		// be exact, which a DELETE ... LIMIT would not guarantee across dialects.
		deleted, err := repo.DeleteBelow(context.Background(), ProjectRef{"ns", "proj"}, ids[9], 3)
		assert.NoError(t, err)
		assert.Equal(t, int64(9), deleted)

		assert.Equal(t, int64(1), countActivityRows(t, db))
	})

	t.Run("nothing below the cursor", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)

		ids := seedActivityEvents(t, db, "ns", "proj", 3)

		deleted, err := repo.DeleteBelow(context.Background(), ProjectRef{"ns", "proj"}, ids[0], 10)
		assert.NoError(t, err)
		assert.Zero(t, deleted)
		assert.Equal(t, int64(3), countActivityRows(t, db))
	})

	t.Run("rejects a non-positive batch size", func(t *testing.T) {
		db := setupActivityEventTestDB(t)
		repo := NewActivityEventRepository(db)

		_, err := repo.DeleteBelow(context.Background(), ProjectRef{"ns", "proj"}, 100, 0)
		assert.ErrorContains(t, err, "batch size must be positive")
	})
}

func countActivityRows(t *testing.T, db *gorm.DB) int64 {
	t.Helper()

	var count int64
	assert.NoError(t, db.Model(&model.ActivityEvent{}).Count(&count).Error)
	return count
}
