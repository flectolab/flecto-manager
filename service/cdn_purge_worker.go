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
	// cdnMaxAttempts bounds consecutive temporary failures before a job is given up on.
	cdnMaxAttempts = 5
	// cdnJobsPerRun bounds the jobs of one run, so a backlog does not overrun its timeout.
	cdnJobsPerRun = 20
	// cdnErrorMaxLength matches the size of the error column.
	cdnErrorMaxLength = 1000
	// cdnSaveTimeout bounds a save made with a detached context.
	cdnSaveTimeout = 5 * time.Second
)

// saveContext detaches from the run's context: progress must still be saved once
// the run is out of time, and a cancelled context makes the database refuse it.
func saveContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), cdnSaveTimeout)
}

// CdnPurgeWorker sends the queued invalidations. A run stops when its context
// ends, having saved its progress, and the next run resumes from there.
type CdnPurgeWorker interface {
	// Run advances jobs until the context ends and returns how many it finished.
	Run(ctx context.Context) (int, error)
	PurgeExpiredJobs(ctx context.Context) (int64, error)
}

type cdnPurgeWorker struct {
	ctx        *appContext.Context
	jobRepo    repository.ProjectCdnPurgeJobRepository
	configRepo repository.ProjectCdnConfigRepository
	agentRepo  repository.AgentRepository
	activity   ActivityService

	// limiters are per configuration: two configurations on the same CDN account
	// share a quota this cannot see, acceptable since jobs run one at a time.
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
			// Cut short by shutdown or timeout: expected with a backlog, not a failure.
			return 0, nil
		}
		return 0, err
	}

	finished := 0
	for i := range jobs {
		if ctx.Err() != nil {
			return finished, nil
		}

		done, errJob := w.advance(ctx, &jobs[i])
		if errJob != nil {
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

// advance reports whether the job reached a terminal state.
func (w *cdnPurgeWorker) advance(ctx context.Context, job *model.ProjectCdnPurgeJob) (bool, error) {
	config, err := w.configRepo.FindByName(ctx, job.NamespaceCode, job.ProjectCode, job.CdnConfigName)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}

	// Disabled or removed since the publish: on purpose, so skipped rather than failed.
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

		// Conditional on the job being as read: a concurrent publish merge brought
		// targets and a version this copy lacks. The next run looks at it again.
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

// readyToPurge reports whether the agents serve the job's version, and whether that
// was reached by waiting or by giving up. Purging earlier lets the CDN re-cache the
// old response from a lagging agent.
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

func (w *cdnPurgeWorker) send(ctx context.Context, job *model.ProjectCdnPurgeJob, provider cdn.Provider) (bool, error) {
	limits := provider.Limits()
	limiter := w.limiter(job.CdnConfigID, limits)

	for _, batch := range cdn.Batch(job.Remaining(), limits.MaxTargetsPerRequest) {
		if ctx.Err() != nil {
			return false, nil
		}

		if err := limiter.WaitN(ctx, len(batch)); err != nil {
			return false, nil
		}

		err := provider.Purge(ctx, batch)
		if err == nil {
			job.Cursor += len(batch)
			job.Purged += len(batch)
			// cdnMaxAttempts bounds a streak of failures, not those of a whole job.
			job.Attempts = 0
			job.Error = ""
			// Saved per batch: an interrupted run resends at most one batch.
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

func (w *cdnPurgeWorker) handlePurgeError(
	ctx context.Context,
	job *model.ProjectCdnPurgeJob,
	err error,
) (bool, error) {
	if ctx.Err() != nil {
		// The run ended mid-request: no attempt is counted, and resending the batch
		// next run is harmless since a purge is idempotent.
		return false, nil
	}

	job.Error = truncateError(err.Error())

	if _, temporary := cdn.AsTemporary(err); temporary {
		job.Attempts++
		if job.Attempts < cdnMaxAttempts {
			// Cursor kept: the batch is resent next run, a purge being idempotent.
			w.ctx.Logger.Warn("cdn purge batch deferred",
				"namespace", job.NamespaceCode, "project", job.ProjectCode, "config", job.CdnConfigName,
				"job", job.ID, "attempt", job.Attempts, "error", err)
			return false, w.persistProgress(ctx, job)
		}
	}

	// Permanent or repeated failure: the rest would fail the same way, so the
	// remaining targets are counted as failed rather than retried.
	job.Failed += len(job.Remaining())
	job.Cursor = len(job.Targets)

	status := model.CdnPurgeStatusFailed
	if job.Purged > 0 {
		status = model.CdnPurgeStatusPartial
	}

	return true, w.finish(ctx, job, status, "", err)
}

// finish closes a job and journals it, even when the provider was never reached:
// the activity entry is the only place a failed purge surfaces.
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

	// Targets of a completed purge are useless and can be large.
	if status == model.CdnPurgeStatusSucceeded || status == model.CdnPurgeStatusSkipped {
		job.Targets = nil
	}

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

// recordCdnPurgeOutcome counts a finished job in the metrics and journals it,
// for both the worker and a publish that turns a job away.
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
		appCtx.Logger.Error("failed to record cdn purge activity",
			"namespace", job.NamespaceCode, "project", job.ProjectCode, "job", job.ID, "error", err)
	}
}

// limiter returns the rate limiter of a configuration. It counts targets, not
// requests, as Cloudflare's purge quota is in URLs per second.
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
	// The burst must hold a whole batch, or WaitN can never be satisfied.
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

const cdnJobRetentionBatchSize = 500

// truncateError cuts on a character boundary: invalid UTF-8 is refused by a
// strict MySQL, and the job could then never be saved.
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
