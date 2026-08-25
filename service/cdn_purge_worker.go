package service

import (
	"context"
	"errors"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/flectolab/flecto-manager/cdn"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/metrics"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/types"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

const (
	// cdnMaxAttempts bounds how many runs a job may fail temporarily before it is
	// given up on. A rate limit or an outage clears in minutes; something failing
	// this many times is not going to clear on its own.
	cdnMaxAttempts = 5
	// cdnJobsPerRun bounds how many jobs one run looks at, so a backlog is worked
	// through steadily instead of in one long run that overruns its timeout.
	cdnJobsPerRun = 20
	// cdnErrorMaxLength bounds what is kept of a provider error, matching the column.
	cdnErrorMaxLength = 1000
	// cdnSaveTimeout bounds a save made after the run's own context ended.
	cdnSaveTimeout = 5 * time.Second
)

// saveContext returns a context that outlives the run's: the moment a run runs out
// of time is precisely when what it did has to be written down, and a cancelled
// context makes the database refuse the write.
func saveContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), cdnSaveTimeout)
}

// CdnPurgeWorker sends the queued invalidations.
//
// It is written to be interrupted: a run stops at the first sign that its context
// is done, having saved how far it got, and the next one picks the job back up
// from there. That is what lets a purge of fifty thousand URLs live inside a
// scheduler task whose runs are short.
type CdnPurgeWorker interface {
	// Run advances every job it can until the context ends, and reports how many it
	// finished.
	Run(ctx context.Context) (int, error)
	// PurgeExpiredJobs drops the finished jobs older than the configured retention.
	PurgeExpiredJobs(ctx context.Context) (int64, error)
}

type cdnPurgeWorker struct {
	ctx        *appContext.Context
	jobRepo    repository.ProjectCdnPurgeJobRepository
	configRepo repository.ProjectCdnConfigRepository
	agentRepo  repository.AgentRepository
	activity   ActivityService

	// limiters are kept per configuration, built from what its provider declares.
	// Two configurations pointing at the same CDN account share the real quota,
	// which this cannot see; jobs run one at a time, so the aggregate stays close
	// enough to the slowest of them.
	limitersMutex sync.Mutex
	limiters      map[int64]*rate.Limiter
}

func NewCdnPurgeWorker(
	ctx *appContext.Context,
	jobRepo repository.ProjectCdnPurgeJobRepository,
	configRepo repository.ProjectCdnConfigRepository,
	agentRepo repository.AgentRepository,
	activity ActivityService,
) CdnPurgeWorker {
	return &cdnPurgeWorker{
		ctx:        ctx,
		jobRepo:    jobRepo,
		configRepo: configRepo,
		agentRepo:  agentRepo,
		activity:   activity,
		limiters:   map[int64]*rate.Limiter{},
	}
}

func (w *cdnPurgeWorker) Run(ctx context.Context) (int, error) {
	if !w.ctx.Config.Cdn.Enabled {
		return 0, nil
	}

	jobs, err := w.jobRepo.FindActive(ctx, cdnJobsPerRun)
	if err != nil {
		if ctx.Err() != nil {
			// Cut short by a shutdown or by the run's own timeout. A big backlog hits
			// that on every run by design, so reporting it as a failure would make the
			// task's error metric say nothing.
			return 0, nil
		}
		return 0, err
	}

	finished := 0
	for i := range jobs {
		if ctx.Err() != nil {
			// Out of time. What is done is saved, the next run resumes.
			return finished, nil
		}

		done, errJob := w.advance(ctx, &jobs[i])
		if errJob != nil {
			// One job failing must not hold back the others: it is recorded on the
			// job itself and the run carries on.
			w.ctx.Logger.Error("cdn purge job failed",
				"namespace", jobs[i].NamespaceCode, "project", jobs[i].ProjectCode,
				"config", jobs[i].CdnConfigName, "job", jobs[i].ID, "error", errJob)
			continue
		}
		if done {
			finished++
		}
	}

	return finished, nil
}

// advance moves one job forward and reports whether it reached a terminal state.
func (w *cdnPurgeWorker) advance(ctx context.Context, job *model.ProjectCdnPurgeJob) (bool, error) {
	config, err := w.configRepo.FindByName(ctx, job.NamespaceCode, job.ProjectCode, job.CdnConfigName)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}

	// A configuration disabled or removed between the publish and the purge is an
	// explicit answer, not a failure: someone turned it off on purpose.
	if config == nil || !config.IsEnabled() {
		return true, w.finish(ctx, job, model.CdnPurgeStatusSkipped, model.CdnPurgeSkipReasonConfigDisabled, nil)
	}

	if job.Status == model.CdnPurgeStatusWaitingAgents {
		ready, waited, errReady := w.readyToPurge(ctx, job)
		if errReady != nil {
			return false, errReady
		}
		if !ready {
			return false, nil
		}

		// Conditional on the job being as it was read: a publish merging into it in
		// the meantime brought targets this copy does not have, and a version the
		// agents were not checked against. The next run looks at it again.
		started, errStart := w.jobRepo.MarkRunning(ctx, job, time.Now(), waited)
		if errStart != nil || !started {
			return false, errStart
		}
	}

	provider, err := cdn.New(config.Type, config.Settings, config.Secrets, cdnProviderOptions(w.ctx))
	if err != nil {
		return true, w.finish(ctx, job, model.CdnPurgeStatusFailed, "", err)
	}

	return w.send(ctx, job, provider)
}

// readyToPurge reports whether the agents of the project are serving the version
// the job was queued for, and whether that was reached by waiting or by giving up.
//
// Purging before they catch up is worse than not purging at all: the CDN goes
// straight back to an agent that still answers with the old redirect and caches it
// again, which looks exactly like a purge that silently did nothing.
func (w *cdnPurgeWorker) readyToPurge(ctx context.Context, job *model.ProjectCdnPurgeJob) (ready, waited bool, err error) {
	agents, err := w.agentRepo.FindByProject(ctx, job.NamespaceCode, job.ProjectCode)
	if err != nil {
		return false, false, err
	}

	onlineSince := time.Now().Add(-w.ctx.Config.Agent.OfflineThreshold)
	behind := 0
	online := 0

	for _, agent := range agents {
		if agent.LastHitAt.Before(onlineSince) {
			continue
		}
		online++
		if agent.Version < job.ProjectVersion {
			behind++
		}
	}

	// No agent to wait for: nothing is going to serve the old response back.
	if online == 0 || behind == 0 {
		return true, true, nil
	}

	if time.Now().Before(job.DeadlineAt) {
		return false, false, nil
	}

	w.ctx.Logger.Warn("cdn purge starting before agents caught up, the cache may be refilled with the previous version",
		"namespace", job.NamespaceCode, "project", job.ProjectCode, "config", job.CdnConfigName,
		"job", job.ID, "version", job.ProjectVersion, "agentsBehind", behind, "agentsOnline", online)

	return true, false, nil
}

// send pushes the remaining batches until the job is done or the run is out of
// time.
func (w *cdnPurgeWorker) send(ctx context.Context, job *model.ProjectCdnPurgeJob, provider cdn.Provider) (bool, error) {
	limits := provider.Limits()
	limiter := w.limiter(job.CdnConfigID, limits)

	for _, batch := range cdn.Batch(job.Remaining(), limits.MaxTargetsPerRequest) {
		if ctx.Err() != nil {
			// Out of time. Progress is saved after every batch, so the next run
			// resumes from here.
			return false, nil
		}

		// WaitN blocks for the time the quota needs, and returns as soon as the
		// context ends rather than sleeping through a shutdown.
		if err := limiter.WaitN(ctx, len(batch)); err != nil {
			return false, nil
		}

		err := provider.Purge(ctx, batch)
		if err == nil {
			job.Cursor += len(batch)
			job.Purged += len(batch)
			// cdnMaxAttempts bounds a streak of failures, not the failures of a whole
			// job: a long purge meeting a few rate limits far apart is healthy.
			job.Attempts = 0
			job.Error = ""
			// Saved batch by batch: a run cut short, or a process killed, then
			// resends at most one batch instead of the whole job, and the history
			// shows the purge moving.
			if errSave := w.persistProgress(ctx, job); errSave != nil {
				return false, errSave
			}
			continue
		}

		return w.handlePurgeError(ctx, job, err)
	}

	status := model.CdnPurgeStatusSucceeded
	if job.Failed > 0 {
		status = model.CdnPurgeStatusPartial
	}
	return true, w.finish(ctx, job, status, "", nil)
}

// handlePurgeError decides what a rejected batch means for the job.
func (w *cdnPurgeWorker) handlePurgeError(
	ctx context.Context,
	job *model.ProjectCdnPurgeJob,
	err error,
) (bool, error) {
	if ctx.Err() != nil {
		// The run ended mid-request: the provider did nothing wrong, so it costs
		// no attempt. The batch is sent again next run, which is harmless since a
		// purge is idempotent.
		return false, nil
	}

	job.Error = truncateError(err.Error())

	if _, temporary := cdn.AsTemporary(err); temporary {
		job.Attempts++
		if job.Attempts < cdnMaxAttempts {
			// Leave the cursor where it is and come back next run: a purge is
			// idempotent, so resending the batch costs nothing but quota.
			w.ctx.Logger.Warn("cdn purge batch deferred",
				"namespace", job.NamespaceCode, "project", job.ProjectCode, "config", job.CdnConfigName,
				"job", job.ID, "attempt", job.Attempts, "error", err)
			return false, w.persistProgress(ctx, job)
		}
	}

	// Either it failed for good, or it kept failing. Both mean the rest of the job
	// would fail the same way - a bad credential does not fix itself between two
	// batches - so the remaining targets are counted as failed rather than retried
	// one by one.
	job.Failed += len(job.Remaining())
	job.Cursor = len(job.Targets)

	status := model.CdnPurgeStatusFailed
	if job.Purged > 0 {
		status = model.CdnPurgeStatusPartial
	}

	return true, w.finish(ctx, job, status, "", err)
}

// finish closes a job and records what happened. The activity entry is the only
// place a failed purge surfaces, so it is written even when the job failed to
// reach the provider at all.
func (w *cdnPurgeWorker) finish(
	ctx context.Context,
	job *model.ProjectCdnPurgeJob,
	status model.CdnPurgeStatus,
	skipReason string,
	cause error,
) error {
	now := time.Now()
	job.Status = status
	job.SkipReason = skipReason
	job.FinishedAt = &now
	if cause != nil {
		job.Error = truncateError(cause.Error())
	}

	// The addresses of a purge that went through are of no use to anyone, and a
	// publish of fifty thousand URLs would keep them forever.
	if status == model.CdnPurgeStatusSucceeded || status == model.CdnPurgeStatusSkipped {
		job.Targets = nil
	}

	// A job can finish on the very last instant of a run: its outcome is saved and
	// journaled all the same.
	ctx, cancel := saveContext(ctx)
	defer cancel()

	if err := w.jobRepo.Update(ctx, job); err != nil {
		return err
	}

	recordCdnPurgeOutcome(ctx, w.ctx, w.activity, nil, job)

	w.ctx.Logger.Info("cdn purge finished",
		"namespace", job.NamespaceCode, "project", job.ProjectCode, "config", job.CdnConfigName,
		"job", job.ID, "status", status, "purged", job.Purged, "failed", job.Failed)
	return nil
}

// persistProgress saves how far a job got, even when the run is out of time.
func (w *cdnPurgeWorker) persistProgress(ctx context.Context, job *model.ProjectCdnPurgeJob) error {
	saveCtx, cancel := saveContext(ctx)
	defer cancel()
	return w.jobRepo.UpdateProgress(saveCtx, job)
}

func skippedTotal(skipped map[string]int) int {
	total := 0
	for _, count := range skipped {
		total += count
	}
	return total
}

// recordCdnPurgeOutcome counts a finished job in the metrics and journals it. It
// serves both the worker and a publish turning a job away on the spot, so that
// every job that ends shows up in the same two places, whoever ended it.
func recordCdnPurgeOutcome(
	ctx context.Context,
	appCtx *appContext.Context,
	activity ActivityService,
	tx *gorm.DB,
	job *model.ProjectCdnPurgeJob,
) {
	duration := time.Duration(0)
	if job.StartedAt != nil && job.FinishedAt != nil {
		duration = job.FinishedAt.Sub(*job.StartedAt)
	}
	metrics.RecordCdnPurgeJob(job.CdnConfigType, string(job.Status), job.Purged, job.Failed, skippedTotal(job.Skipped), duration)

	err := activity.Record(ctx, tx, model.ActivityInput{
		NamespaceCode: job.NamespaceCode,
		ProjectCode:   job.ProjectCode,
		Resource:      model.ActivityResourceCdn,
		Action:        model.ActivityActionPurge,
		ResourceID:    types.Ptr(job.ID),
		Data: model.ActivityCdnPurge{
			Config:          job.CdnConfigName,
			Type:            job.CdnConfigType,
			Status:          job.Status,
			Version:         job.ProjectVersion,
			Requested:       job.Requested,
			Purged:          job.Purged,
			Failed:          job.Failed,
			Skipped:         job.Skipped,
			SkipReason:      job.SkipReason,
			WaitedForAgents: job.WaitedForAgents,
			DurationMs:      duration.Milliseconds(),
			Error:           job.Error,
		},
	})
	if err != nil {
		// The purge already happened; failing to journal it must not undo anything.
		appCtx.Logger.Error("failed to record cdn purge activity",
			"namespace", job.NamespaceCode, "project", job.ProjectCode, "job", job.ID, "error", err)
	}
}

// limiter returns the rate limiter of a configuration, built from what its
// provider allows. Cloudflare counts its single-file purge quota in URLs per
// second rather than in requests, which is why the limiter is fed the size of the
// batch and not one token per call.
func (w *cdnPurgeWorker) limiter(configID int64, limits cdn.Limits) *rate.Limiter {
	w.limitersMutex.Lock()
	defer w.limitersMutex.Unlock()

	if existing, ok := w.limiters[configID]; ok {
		return existing
	}

	perSecond := limits.TargetsPerSecond
	if perSecond <= 0 {
		perSecond = limits.MaxTargetsPerRequest
	}
	burst := limits.Burst
	if burst <= 0 {
		burst = perSecond
	}
	// The burst has to hold a whole batch, otherwise WaitN can never be satisfied
	// and the job would block forever on its first send.
	if burst < limits.MaxTargetsPerRequest {
		burst = limits.MaxTargetsPerRequest
	}

	limiter := rate.NewLimiter(rate.Limit(perSecond), burst)
	w.limiters[configID] = limiter
	return limiter
}

func (w *cdnPurgeWorker) PurgeExpiredJobs(ctx context.Context) (int64, error) {
	retention := w.ctx.Config.Cdn.JobRetention
	if retention <= 0 {
		return 0, nil
	}

	return w.jobRepo.DeleteFinishedBefore(ctx, time.Now().Add(-retention), cdnJobRetentionBatchSize)
}

// cdnJobRetentionBatchSize bounds a single delete statement of the retention pass.
const cdnJobRetentionBatchSize = 500

// truncateError cuts a message to the column size, on a character boundary: half
// of a multi-byte character is invalid UTF-8, which a strict MySQL refuses to
// store, and the job could then never be saved as finished.
func truncateError(message string) string {
	if len(message) <= cdnErrorMaxLength {
		return message
	}
	end := cdnErrorMaxLength
	for end > 0 && !utf8.RuneStart(message[end]) {
		end--
	}
	return message[:end]
}
