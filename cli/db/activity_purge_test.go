package db

import (
	"errors"
	"testing"
	"time"

	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupActivityPurgeTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.ActivityEvent{})
	require.NoError(t, err)

	return db
}

// withActivityPurgeDB points the command at db for the duration of the test.
func withActivityPurgeDB(t *testing.T, db *gorm.DB, err error) {
	t.Helper()

	previous := NewActivityPurgeDB
	NewActivityPurgeDB = func(*appContext.Context) (*gorm.DB, error) {
		return db, err
	}
	t.Cleanup(func() { NewActivityPurgeDB = previous })
}

func seedActivityEvents(t *testing.T, db *gorm.DB, namespaceCode, projectCode string, n int) {
	t.Helper()

	for i := 0; i < n; i++ {
		require.NoError(t, db.Create(&model.ActivityEvent{
			NamespaceCode: namespaceCode,
			ProjectCode:   projectCode,
			Resource:      model.ActivityResourceRedirect,
			Action:        model.ActivityActionCreate,
			Actor:         "alice",
			OccurredAt:    time.Now(),
		}).Error)
	}
}

func countActivityEvents(t *testing.T, db *gorm.DB, namespaceCode, projectCode string) int64 {
	t.Helper()

	var count int64
	require.NoError(t, db.Model(&model.ActivityEvent{}).
		Where("namespace_code = ? AND project_code = ?", namespaceCode, projectCode).
		Count(&count).Error)
	return count
}

func TestGetActivityPurgeCmd(t *testing.T) {
	ctx := appContext.TestContext(nil)
	cmd := GetActivityPurgeCmd(ctx)

	assert.Equal(t, "activity-purge", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestGetActivityPurgeRunFn_Success(t *testing.T) {
	db := setupActivityPurgeTestDB(t)
	withActivityPurgeDB(t, db, nil)

	ctx := appContext.TestContext(nil)
	ctx.Config.Activity.MaxEventsPerProject = 3

	seedActivityEvents(t, db, "ns", "proj", 10)

	cmd := GetActivityPurgeCmd(ctx)
	assert.NoError(t, cmd.Execute())

	assert.Equal(t, int64(3), countActivityEvents(t, db, "ns", "proj"))
}

func TestGetActivityPurgeRunFn_KeepsTheMostRecentEvents(t *testing.T) {
	db := setupActivityPurgeTestDB(t)
	withActivityPurgeDB(t, db, nil)

	ctx := appContext.TestContext(nil)
	ctx.Config.Activity.MaxEventsPerProject = 2

	seedActivityEvents(t, db, "ns", "proj", 5)

	var expected []int64
	require.NoError(t, db.Model(&model.ActivityEvent{}).Order("id DESC").Limit(2).Pluck("id", &expected).Error)

	cmd := GetActivityPurgeCmd(ctx)
	require.NoError(t, cmd.Execute())

	var remaining []int64
	require.NoError(t, db.Model(&model.ActivityEvent{}).Order("id DESC").Pluck("id", &remaining).Error)
	assert.Equal(t, expected, remaining, "the purge must keep the newest events")
}

func TestGetActivityPurgeRunFn_CapsEachProjectIndependently(t *testing.T) {
	db := setupActivityPurgeTestDB(t)
	withActivityPurgeDB(t, db, nil)

	ctx := appContext.TestContext(nil)
	ctx.Config.Activity.MaxEventsPerProject = 2

	seedActivityEvents(t, db, "ns", "busy", 6)
	seedActivityEvents(t, db, "ns", "quiet", 1)

	cmd := GetActivityPurgeCmd(ctx)
	require.NoError(t, cmd.Execute())

	assert.Equal(t, int64(2), countActivityEvents(t, db, "ns", "busy"))
	assert.Equal(t, int64(1), countActivityEvents(t, db, "ns", "quiet"), "a project under the cap is left alone")
}

func TestGetActivityPurgeRunFn_NothingToPurge(t *testing.T) {
	db := setupActivityPurgeTestDB(t)
	withActivityPurgeDB(t, db, nil)

	ctx := appContext.TestContext(nil)
	ctx.Config.Activity.MaxEventsPerProject = 10

	seedActivityEvents(t, db, "ns", "proj", 4)

	cmd := GetActivityPurgeCmd(ctx)
	assert.NoError(t, cmd.Execute())

	assert.Equal(t, int64(4), countActivityEvents(t, db, "ns", "proj"))
}

func TestGetActivityPurgeRunFn_EmptyTable(t *testing.T) {
	db := setupActivityPurgeTestDB(t)
	withActivityPurgeDB(t, db, nil)

	ctx := appContext.TestContext(nil)
	ctx.Config.Activity.MaxEventsPerProject = 5

	cmd := GetActivityPurgeCmd(ctx)
	assert.NoError(t, cmd.Execute())
}

func TestGetActivityPurgeRunFn_RefusesWithoutCap(t *testing.T) {
	tests := []struct {
		name string
		cap  int
	}{
		{name: "unset", cap: 0},
		{name: "negative", cap: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupActivityPurgeTestDB(t)
			// The command must refuse before touching the database: purging against
			// no cap would either delete everything or silently do nothing, and both
			// are worse than telling the operator the config is missing.
			withActivityPurgeDB(t, nil, errors.New("database must not be opened"))

			ctx := appContext.TestContext(nil)
			ctx.Config.Activity.MaxEventsPerProject = tt.cap

			seedActivityEvents(t, db, "ns", "proj", 5)

			cmd := GetActivityPurgeCmd(ctx)
			err := cmd.Execute()

			assert.ErrorContains(t, err, "activity.max_events_per_project is not set")
			assert.Equal(t, int64(5), countActivityEvents(t, db, "ns", "proj"), "nothing must have been purged")
		})
	}
}

func TestGetActivityPurgeRunFn_DatabaseError(t *testing.T) {
	errDB := errors.New("connection refused")
	withActivityPurgeDB(t, nil, errDB)

	ctx := appContext.TestContext(nil)
	ctx.Config.Activity.MaxEventsPerProject = 5

	cmd := GetActivityPurgeCmd(ctx)
	err := cmd.Execute()

	assert.ErrorIs(t, err, errDB)
}

func TestGetActivityPurgeRunFn_PurgeError(t *testing.T) {
	// A database without the activity_events table: the purge must report the
	// failure rather than pretend it trimmed anything.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	withActivityPurgeDB(t, db, nil)

	ctx := appContext.TestContext(nil)
	ctx.Config.Activity.MaxEventsPerProject = 5

	cmd := GetActivityPurgeCmd(ctx)
	assert.Error(t, cmd.Execute())
}
