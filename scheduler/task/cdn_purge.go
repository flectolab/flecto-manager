package task

import (
	"context"
	"time"

	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/service"
)

// cdnPurgeTimeoutMargin keeps a run shorter than its interval. Hitting the timeout
// is not a failure: progress is saved and the next run resumes.
const cdnPurgeTimeoutMargin = 5 * time.Second

// CdnPurge sends the queued cache invalidations, spread over short runs since
// the CDN quota paces a large purge over minutes.
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

// Interval is zero when the feature is off, so the scheduler skips the task.
func (t *CdnPurge) Interval() time.Duration {
	if !t.ctx.Config.Cdn.Enabled {
		return 0
	}
	return t.ctx.Config.Cdn.WorkerInterval
}

// RunOnStart avoids leaving the cache stale for an interval after a restart.
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

	// Retention comes last: it is never worth delaying an invalidation.
	purged, err := t.worker.PurgeExpiredJobs(ctx)
	if err != nil {
		return err
	}
	if purged > 0 {
		t.ctx.Logger.Info("cdn purge jobs removed", "purged", purged)
	}

	return nil
}
