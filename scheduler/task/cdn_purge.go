package task

import (
	"context"
	"time"

	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/service"
)

// cdnPurgeTimeoutMargin keeps a run shorter than the interval that triggers it, so
// two runs never follow each other with no pause. A run that hits the timeout is
// not a failure: the job it was working on is saved mid-flight and the next run
// picks it up.
const cdnPurgeTimeoutMargin = 5 * time.Second

// CdnPurge sends the cache invalidations a publish queued.
//
// The work is deliberately spread over runs rather than done in one: a publish
// touching tens of thousands of URLs is paced by the CDN's own quota, which is
// minutes of sending, and a scheduler task that ran that long would block its own
// next tick.
type CdnPurge struct {
	ctx    *appContext.Context
	worker service.CdnPurgeWorker
}

func NewCdnPurge(ctx *appContext.Context, worker service.CdnPurgeWorker) *CdnPurge {
	return &CdnPurge{ctx: ctx, worker: worker}
}

func (t *CdnPurge) Name() string {
	return "cdn-purge"
}

// Interval is zero when the feature is off, which is how the scheduler skips the
// task entirely rather than waking up to do nothing.
func (t *CdnPurge) Interval() time.Duration {
	if !t.ctx.Config.Cdn.Enabled {
		return 0
	}
	return t.ctx.Config.Cdn.WorkerInterval
}

// RunOnStart sends once at boot: an instance that was down while a publish queued
// a purge should not leave the cache stale for another interval.
func (t *CdnPurge) RunOnStart() bool {
	return true
}

func (t *CdnPurge) Timeout() time.Duration {
	interval := t.Interval()
	if interval <= cdnPurgeTimeoutMargin {
		return interval
	}
	return interval - cdnPurgeTimeoutMargin
}

func (t *CdnPurge) Run(ctx context.Context) error {
	finished, err := t.worker.Run(ctx)
	if err != nil {
		return err
	}

	if finished > 0 {
		t.ctx.Logger.Info("cdn purge run completed", "finished", finished)
	}

	// Retention runs after the sending, and only with whatever time is left: dropping
	// old rows is never worth delaying an invalidation.
	purged, err := t.worker.PurgeExpiredJobs(ctx)
	if err != nil {
		return err
	}
	if purged > 0 {
		t.ctx.Logger.Info("cdn purge jobs removed", "purged", purged)
	}

	return nil
}
