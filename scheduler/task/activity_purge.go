// Package task holds the background tasks run by the scheduler.
package task

import (
	"context"
	"time"

	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/service"
)

// activityPurgeTimeout bounds a single purge run. A purge that takes longer than
// this is not making progress in a way another attempt would not.
const activityPurgeTimeout = 10 * time.Minute

// ActivityPurge trims the activity journal down to its configured cap. It is the
// only guard on the table size, so it is scheduled rather than left to a manual
// command, though the command exists for ops.
type ActivityPurge struct {
	ctx      *appContext.Context
	activity service.ActivityService
}

func NewActivityPurge(ctx *appContext.Context, activity service.ActivityService) *ActivityPurge {
	return &ActivityPurge{ctx: ctx, activity: activity}
}

func (t *ActivityPurge) Name() string {
	return "activity-purge"
}

func (t *ActivityPurge) Interval() time.Duration {
	return t.ctx.Config.Activity.PurgeInterval
}

// RunOnStart trims once at boot: an instance that was down while events piled up
// should not wait a full interval to catch up.
func (t *ActivityPurge) RunOnStart() bool {
	return true
}

func (t *ActivityPurge) Timeout() time.Duration {
	return activityPurgeTimeout
}

func (t *ActivityPurge) Run(ctx context.Context) error {
	purged, err := t.activity.Purge(ctx)
	if err != nil {
		return err
	}

	// Only worth a line when it actually trimmed something, otherwise an hourly
	// no-op would fill the logs.
	if purged > 0 {
		t.ctx.Logger.Info("activity purge completed", "purged", purged)
	}
	return nil
}
