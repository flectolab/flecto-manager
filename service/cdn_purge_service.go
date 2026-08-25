package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/flectolab/flecto-manager/cdn"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"gorm.io/gorm"
)

// cdnAddressBatchSize bounds an IN clause reading the addresses a publish is
// about to change, for the same reason the publish batches its own statements.
const cdnAddressBatchSize = 500

type CdnPurgeService interface {
	// Collect reads what a publish is about to invalidate, and through which
	// configurations. It runs before the transaction, because a publish deletes the
	// rows that say what a removed entry used to serve.
	//
	// A project with no enabled configuration costs one query and nothing else.
	Collect(ctx context.Context, namespaceCode, projectCode string, changes model.CdnDraftChanges) (model.CdnPublishPlan, error)
	// Enqueue records the purges of a publish, one job per configuration. It runs
	// inside the publish transaction: a purge must not outlive a rolled back
	// publish, nor be lost by one that went through.
	Enqueue(ctx context.Context, tx *gorm.DB, plan model.CdnPublishPlan, projectVersion int) error
	// PurgeURLs queues a purge of explicit URLs, outside of any publish. It is the
	// way out when a purge was skipped or failed: the addresses are given as they
	// are served, and go through the same worker.
	PurgeURLs(ctx context.Context, namespaceCode, projectCode, configName string, urls []string) (*model.ProjectCdnPurgeJob, error)
	// History returns the purge jobs of a project, newest first.
	History(ctx context.Context, namespaceCode, projectCode string, limit int) ([]model.ProjectCdnPurgeJob, error)
}

var (
	// ErrCdnConfigDisabled is returned when a manual purge targets a configuration
	// that is turned off. Queueing a job the worker would immediately skip would
	// look like it worked.
	ErrCdnConfigDisabled = errors.New("cdn configuration is disabled")
	// ErrCdnNothingToPurge is returned when none of the submitted URLs can be sent
	// through that configuration.
	ErrCdnNothingToPurge = errors.New("none of the given urls can be purged through this configuration")
	// ErrCdnPurgeDisabled is returned when the instance sends no purge at all
	// (cdn.enabled off, or cdn.worker_interval at 0). The job would sit in the queue
	// with nothing to ever send it, while the caller is told it was queued.
	ErrCdnPurgeDisabled = errors.New("cdn purge is disabled on this instance")
)

// cdnHistoryDefaultLimit bounds the purge history returned by default, which is
// read by a page showing the last runs and not by anything paginating.
const cdnHistoryDefaultLimit = 50

type cdnPurgeService struct {
	ctx          *appContext.Context
	configRepo   repository.ProjectCdnConfigRepository
	jobRepo      repository.ProjectCdnPurgeJobRepository
	redirectRepo repository.RedirectRepository
	pageRepo     repository.PageRepository
	activity     ActivityService
}

func NewCdnPurgeService(
	ctx *appContext.Context,
	configRepo repository.ProjectCdnConfigRepository,
	jobRepo repository.ProjectCdnPurgeJobRepository,
	redirectRepo repository.RedirectRepository,
	pageRepo repository.PageRepository,
	activity ActivityService,
) CdnPurgeService {
	return &cdnPurgeService{
		ctx:          ctx,
		configRepo:   configRepo,
		jobRepo:      jobRepo,
		redirectRepo: redirectRepo,
		pageRepo:     pageRepo,
		activity:     activity,
	}
}

func (s *cdnPurgeService) Collect(
	ctx context.Context,
	namespaceCode, projectCode string,
	changes model.CdnDraftChanges,
) (model.CdnPublishPlan, error) {
	plan := model.CdnPublishPlan{NamespaceCode: namespaceCode, ProjectCode: projectCode}

	if !s.ctx.Config.Cdn.Enabled {
		return plan, nil
	}

	configs, err := s.configRepo.FindEnabledByProject(ctx, namespaceCode, projectCode)
	if err != nil {
		return plan, err
	}
	if len(configs) == 0 {
		return plan, nil
	}
	plan.Configs = configs

	entries := make([]cdn.Entry, 0, len(changes.NewEntries)+len(changes.RedirectIDs)+len(changes.PageIDs))
	entries = append(entries, changes.NewEntries...)

	redirects, err := s.redirectRepo.FindAddressesByIDs(ctx, nil, changes.RedirectIDs, cdnAddressBatchSize)
	if err != nil {
		return plan, err
	}
	pages, err := s.pageRepo.FindAddressesByIDs(ctx, nil, changes.PageIDs, cdnAddressBatchSize)
	if err != nil {
		return plan, err
	}

	for _, address := range append(redirects, pages...) {
		entries = append(entries, cdn.Entry{Type: address.Type, Source: address.Source})
	}

	plan.Entries = entries
	return plan, nil
}

func (s *cdnPurgeService) Enqueue(ctx context.Context, tx *gorm.DB, plan model.CdnPublishPlan, projectVersion int) error {
	if plan.IsEmpty() {
		return nil
	}

	deadline := time.Now().Add(s.ctx.Config.Cdn.AgentConvergenceTimeout)

	for i := range plan.Configs {
		if err := s.enqueueOne(ctx, tx, &plan.Configs[i], plan, projectVersion, deadline); err != nil {
			return err
		}
	}

	return nil
}

func (s *cdnPurgeService) enqueueOne(
	ctx context.Context,
	tx *gorm.DB,
	config *model.ProjectCdnConfig,
	publish model.CdnPublishPlan,
	projectVersion int,
	deadline time.Time,
) error {
	purge := cdn.NewPlanner(config.Hosts()).Build(publish.Entries)

	if purge.IsEmpty() && purge.SkippedTotal() == 0 {
		return nil
	}

	existing, err := s.jobRepo.FindMergeable(ctx, tx, config.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if existing != nil {
		merged, errMerge := s.merge(ctx, tx, existing, purge, projectVersion, deadline)
		if errMerge != nil || merged {
			return errMerge
		}
		// Not merged: this publish gets a job of its own, below.
	}

	job := &model.ProjectCdnPurgeJob{
		NamespaceCode:  publish.NamespaceCode,
		ProjectCode:    publish.ProjectCode,
		CdnConfigID:    config.ID,
		CdnConfigName:  config.Name,
		CdnConfigType:  config.Type,
		ProjectVersion: projectVersion,
		Status:         model.CdnPurgeStatusWaitingAgents,
		Targets:        purge.Targets,
		Requested:      len(purge.Targets),
		DeadlineAt:     deadline,
	}
	job.AddSkipped(purge.Skipped)

	s.applyCap(job)

	if err = tx.Create(job).Error; err != nil {
		return err
	}

	if job.Status.IsTerminal() {
		// A job turned away here never reaches the worker, which is what records
		// the others: without this, a publish too large to purge would leave no
		// trace outside of a log line. The journal entry is written in the publish
		// transaction, so it goes away with a publish that rolls back.
		recordCdnPurgeOutcome(ctx, s.ctx, s.activity, tx, job)
	}

	s.ctx.Logger.Info("cdn purge job queued",
		"namespace", publish.NamespaceCode, "project", publish.ProjectCode, "config", config.Name,
		"version", projectVersion, "targets", job.Requested, "skipped", purge.SkippedTotal(),
		"status", job.Status)
	return nil
}

// merge folds a new publish into a job that has not started, and reports whether
// it did. Waiting up to half an hour for the agents means several publishes can
// pile up on the same project, and two purges of overlapping URLs are one purge of
// their union.
//
// It declines in two cases, and the publish then gets a job of its own:
//   - the union would exceed the cap. Turning the merged job away would drop the
//     purge of the earlier publish, which was within bounds on its own;
//   - the job was started by a worker, or merged into by another publish, since it
//     was read. Adding to it then would either be lost or come too late.
func (s *cdnPurgeService) merge(
	ctx context.Context,
	tx *gorm.DB,
	job *model.ProjectCdnPurgeJob,
	purge cdn.Plan,
	projectVersion int,
	deadline time.Time,
) (bool, error) {
	targets := unionTargets(job.Targets, purge.Targets)
	if max := s.ctx.Config.Cdn.MaxTargetsPerJob; max > 0 && len(targets) > max {
		s.ctx.Logger.Info("cdn purge job not merged: the union would exceed the cap",
			"namespace", job.NamespaceCode, "project", job.ProjectCode, "config", job.CdnConfigName,
			"job", job.ID, "targets", len(targets), "max", max)
		return false, nil
	}

	readVersion := job.ProjectVersion
	job.Targets = targets
	job.Requested = len(job.Targets)
	job.AddSkipped(purge.Skipped)

	// The job now covers the newer publish, so that is the version the agents have
	// to be serving, and the wait starts over from it.
	job.ProjectVersion = projectVersion
	job.DeadlineAt = deadline

	saved, err := s.jobRepo.MergeWaiting(ctx, tx, job, readVersion)
	if err != nil {
		return false, err
	}
	if !saved {
		s.ctx.Logger.Info("cdn purge job not merged: it changed since it was read",
			"namespace", job.NamespaceCode, "project", job.ProjectCode, "config", job.CdnConfigName,
			"job", job.ID)
		return false, nil
	}

	s.ctx.Logger.Info("cdn purge job merged",
		"namespace", job.NamespaceCode, "project", job.ProjectCode, "config", job.CdnConfigName,
		"version", projectVersion, "targets", job.Requested, "status", job.Status)
	return true, nil
}

// applyCap turns away a job larger than the configured cap. Purging that much one
// URL at a time would take long enough to be pointless, and the alternatives -
// purging a prefix or a whole hostname - drop cached objects nobody changed. Saying
// so plainly leaves the decision to someone who can see the whole picture.
func (s *cdnPurgeService) applyCap(job *model.ProjectCdnPurgeJob) {
	max := s.ctx.Config.Cdn.MaxTargetsPerJob
	if max <= 0 || len(job.Targets) <= max {
		return
	}

	s.ctx.Logger.Warn("cdn purge job skipped: too many targets",
		"namespace", job.NamespaceCode, "project", job.ProjectCode, "config", job.CdnConfigName,
		"targets", len(job.Targets), "max", max)

	now := time.Now()
	job.Status = model.CdnPurgeStatusSkipped
	job.SkipReason = model.CdnPurgeSkipReasonTooManyTargets
	job.Targets = nil
	job.FinishedAt = &now
}

// unionTargets merges two target lists, keeping the order stable so a job's
// batches stay reproducible across a restart.
func unionTargets(current, incoming []cdn.Target) []cdn.Target {
	if len(incoming) == 0 {
		return current
	}

	seen := make(map[cdn.Target]struct{}, len(current)+len(incoming))
	merged := make([]cdn.Target, 0, len(current)+len(incoming))

	for _, target := range append(current, incoming...) {
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		merged = append(merged, target)
	}

	sortTargets(merged)
	return merged
}

func sortTargets(targets []cdn.Target) {
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Host != targets[j].Host {
			return targets[i].Host < targets[j].Host
		}
		return targets[i].Path < targets[j].Path
	})
}

func (s *cdnPurgeService) History(ctx context.Context, namespaceCode, projectCode string, limit int) ([]model.ProjectCdnPurgeJob, error) {
	if limit <= 0 {
		limit = cdnHistoryDefaultLimit
	}
	return s.jobRepo.FindByProject(ctx, namespaceCode, projectCode, limit)
}

func (s *cdnPurgeService) PurgeURLs(
	ctx context.Context,
	namespaceCode, projectCode, configName string,
	urls []string,
) (*model.ProjectCdnPurgeJob, error) {
	if !s.ctx.Config.Cdn.Enabled || s.ctx.Config.Cdn.WorkerInterval <= 0 {
		return nil, ErrCdnPurgeDisabled
	}

	config, err := s.configRepo.FindByName(ctx, namespaceCode, projectCode, configName)
	if err != nil {
		return nil, err
	}
	if !config.IsEnabled() {
		return nil, ErrCdnConfigDisabled
	}

	targets, skipped := planManualTargets(cdn.NewPlanner(config.Hosts()), urls)
	if len(targets) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrCdnNothingToPurge, describeSkipped(skipped))
	}

	if max := s.ctx.Config.Cdn.MaxTargetsPerJob; max > 0 && len(targets) > max {
		return nil, fmt.Errorf("%d urls exceeds the %d allowed in one purge", len(targets), max)
	}

	now := time.Now()
	job := &model.ProjectCdnPurgeJob{
		NamespaceCode: namespaceCode,
		ProjectCode:   projectCode,
		CdnConfigID:   config.ID,
		CdnConfigName: config.Name,
		CdnConfigType: config.Type,
		// A manual purge answers to no publish, so there is no version for the agents
		// to converge on. It is queued straight as running: the worker only checks
		// the agents of a waiting job, and the history does not claim it waits for
		// anyone. Its duration then counts the time spent in the queue.
		ProjectVersion:  0,
		Status:          model.CdnPurgeStatusRunning,
		Targets:         targets,
		Requested:       len(targets),
		WaitedForAgents: true,
		DeadlineAt:      now,
		StartedAt:       &now,
	}
	job.AddSkipped(skipped)

	if err = s.jobRepo.Create(ctx, job); err != nil {
		return nil, err
	}

	s.ctx.Logger.Info("cdn manual purge queued",
		"namespace", namespaceCode, "project", projectCode, "config", config.Name,
		"targets", job.Requested, "skipped", len(skipped))
	return job, nil
}

// planManualTargets turns submitted URLs into targets, applying the same routing
// rule as a publish: a URL whose host this configuration does not serve belongs to
// another zone, and sending it there fails the whole batch it lands in.
func planManualTargets(planner *cdn.Planner, urls []string) ([]cdn.Target, map[cdn.SkipReason]int) {
	skipped := map[cdn.SkipReason]int{}
	seen := map[cdn.Target]struct{}{}
	targets := make([]cdn.Target, 0, len(urls))

	for _, raw := range urls {
		parsed, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.Path == "" {
			skipped[cdn.SkipReasonInvalid]++
			continue
		}

		host := strings.ToLower(parsed.Host)
		if !planner.Serves(host) {
			skipped[cdn.SkipReasonNoHostMatch]++
			continue
		}

		// RequestURI keeps the query string and the encoding as submitted: the
		// cache holds "/promo?a=1" apart from "/promo".
		for _, path := range cdn.PathVariants(parsed.RequestURI()) {
			target := cdn.Target{Kind: cdn.KindURL, Host: host, Path: path}
			if _, ok := seen[target]; ok {
				continue
			}
			seen[target] = struct{}{}
			targets = append(targets, target)
		}
	}

	sortTargets(targets)
	return targets, skipped
}

// describeSkipped renders why nothing could be purged, so the answer is actionable
// rather than a bare refusal.
func describeSkipped(skipped map[cdn.SkipReason]int) string {
	reasons := make([]string, 0, len(skipped))
	for _, reason := range []cdn.SkipReason{cdn.SkipReasonInvalid, cdn.SkipReasonNoHostMatch} {
		if count := skipped[reason]; count > 0 {
			reasons = append(reasons, fmt.Sprintf("%d %s", count, strings.ToLower(string(reason))))
		}
	}
	if len(reasons) == 0 {
		return "no url given"
	}
	return strings.Join(reasons, ", ")
}
