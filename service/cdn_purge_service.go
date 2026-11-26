package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/flectolab/flecto-manager/cdn"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"gorm.io/gorm"
)

// cdnAddressBatchSize bounds the IN clause reading the addresses a publish changes.
const cdnAddressBatchSize = 500

type CdnPurgeService interface {
	// Collect reads what a publish will invalidate. It runs before the transaction,
	// which deletes the rows saying what a removed entry used to serve.
	Collect(ctx context.Context, namespaceCode, projectCode string, changes model.CdnDraftChanges) (model.CdnPublishPlan, error)
	// Enqueue queues one job per configuration, inside the publish transaction.
	Enqueue(ctx context.Context, tx *gorm.DB, plan model.CdnPublishPlan, projectVersion int) error
	// PurgeURLs queues a manual purge of explicit URLs, outside of any publish.
	PurgeURLs(ctx context.Context, namespaceCode, projectCode, configName string, urls []string) (*model.ProjectCdnPurgeJob, error)
	// History returns the purge jobs of a project, newest first.
	History(ctx context.Context, namespaceCode, projectCode string, limit int) ([]model.ProjectCdnPurgeJob, error)
}

var (
	// ErrCdnConfigDisabled is returned when a manual purge targets a disabled configuration.
	ErrCdnConfigDisabled = errors.New("cdn configuration is disabled")
	// ErrCdnNothingToPurge is returned when none of the URLs can go through the configuration.
	ErrCdnNothingToPurge = errors.New("none of the given urls can be purged through this configuration")
	// ErrCdnPurgeDisabled is returned when no worker would ever send the job
	// (cdn.enabled off, or cdn.worker_interval at 0).
	ErrCdnPurgeDisabled = errors.New("cdn purge is disabled on this instance")
)

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
		entries = append(entries, cdn.Entry{
			Type:              address.Type,
			Source:            address.Source,
			IgnoreQueryString: address.Options.IsIgnoreQueryString(),
		})
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
	// An unknown type must not fail the publish: its job fails in the worker, in
	// the purge history, like any job whose provider cannot be built.
	var capabilities cdn.Capabilities
	if descriptor, errDescriptor := cdn.GetDescriptor(config.Type); errDescriptor == nil {
		capabilities = descriptor.Capabilities
	}
	purge := cdn.NewPlanner(config.Hosts()).WithCapabilities(capabilities).Build(publish.Entries)

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
		// Turned away here, the job never reaches the worker that journals the others.
		recordCdnPurgeOutcome(ctx, s.ctx, s.activity, tx, job)
	}

	s.ctx.Logger.Info("cdn purge job queued",
		"namespace", publish.NamespaceCode, "project", publish.ProjectCode, "config", config.Name,
		"version", projectVersion, "targets", job.Requested, "skipped", purge.SkippedTotal(),
		"status", job.Status)
	return nil
}

// merge folds a publish into a job still waiting for agents, and reports whether
// it did. It declines, leaving the publish its own job, when the union would
// exceed the cap (capping it would drop the earlier, valid purge) or when the
// job changed since it was read.
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

	// The agents must now serve the newer version; the wait starts over.
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

// applyCap skips a job larger than the configured cap.
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

// unionTargets keeps the order stable so batches are reproducible across a restart.
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

	cdn.SortTargets(merged)
	return merged
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
		// No publish, so no version for the agents to converge on: queued straight
		// as running. Its duration then includes the time spent in the queue.
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

// planManualTargets turns URLs into targets. A host the configuration does not
// serve belongs to another zone, and would fail the whole batch.
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

		// RequestURI keeps the query string and encoding: the cache keys on them.
		for _, path := range cdn.PathVariants(parsed.RequestURI()) {
			target := cdn.Target{Kind: cdn.KindURL, Host: host, Path: path}
			if _, ok := seen[target]; ok {
				continue
			}
			seen[target] = struct{}{}
			targets = append(targets, target)
		}
	}

	cdn.SortTargets(targets)
	return targets, skipped
}

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
