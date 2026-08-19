package db

import (
	stdContext "context"
	"fmt"

	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/database"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/service"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// CreateActivityPurgeDBFn is a function type for creating database connection (used for testing)
type CreateActivityPurgeDBFn func(ctx *appContext.Context) (*gorm.DB, error)

// NewActivityPurgeDB is the function used to create database connection (can be replaced in tests)
var NewActivityPurgeDB CreateActivityPurgeDBFn = func(ctx *appContext.Context) (*gorm.DB, error) {
	return database.CreateDB(ctx)
}

// GetActivityPurgeCmd runs the purge on demand. The scheduler already runs it, but
// ops needs a way to trim right away after lowering the cap.
func GetActivityPurgeCmd(ctx *appContext.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "activity-purge",
		Short: "purge activity events beyond the configured per-project cap",
		RunE:  GetActivityPurgeRunFn(ctx),
	}
}

func GetActivityPurgeRunFn(ctx *appContext.Context) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		keep := ctx.Config.Activity.MaxEventsPerProject
		if keep <= 0 {
			return fmt.Errorf("activity.max_events_per_project is not set, nothing to purge against")
		}

		db, errDb := NewActivityPurgeDB(ctx)
		if errDb != nil {
			return errDb
		}

		activityService := service.NewActivityService(ctx, repository.NewActivityEventRepository(db))

		purged, err := activityService.Purge(stdContext.Background())
		if err != nil {
			return err
		}

		ctx.Logger.Info("activity purge completed", "purged", purged, "keptPerProject", keep)
		return nil
	}
}
