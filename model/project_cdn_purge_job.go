package model

import (
	"time"

	"github.com/flectolab/flecto-manager/cdn"
)

// ResourceAddress is the projection of a redirect or a page down to what a cache
// invalidation needs: what kind of address it is, and the address itself. Pages
// carry their content inline, so reading whole rows to get a path is not an
// option on a large publish.
type ResourceAddress struct {
	ID     int64
	Type   string
	Source string
}

// CdnPurgeStatus is where a purge job stands.
type CdnPurgeStatus string

const (
	// CdnPurgeStatusWaitingAgents is a job held back until the agents of the project
	// serve the published version. Purging before they do lets the CDN cache the old
	// response again, straight from an agent that has not caught up.
	//
	// It is also the only status a later publish can merge into: once a job is
	// running, its batches are already on their way.
	CdnPurgeStatusWaitingAgents CdnPurgeStatus = "WAITING_AGENTS"
	CdnPurgeStatusRunning       CdnPurgeStatus = "RUNNING"
	CdnPurgeStatusSucceeded     CdnPurgeStatus = "SUCCEEDED"
	// CdnPurgeStatusPartial is a job where some batches went through and others did
	// not. The cache is in a mixed state, which is worth saying rather than rounding
	// to success or failure.
	CdnPurgeStatusPartial CdnPurgeStatus = "PARTIAL"
	CdnPurgeStatusFailed  CdnPurgeStatus = "FAILED"
	// CdnPurgeStatusSkipped is a job that was never sent: the configuration was
	// disabled in the meantime, or the publish had more targets than the cap allows.
	CdnPurgeStatusSkipped CdnPurgeStatus = "SKIPPED"
)

// IsTerminal reports whether the job is done, whatever the outcome.
func (s CdnPurgeStatus) IsTerminal() bool {
	switch s {
	case CdnPurgeStatusSucceeded, CdnPurgeStatusPartial, CdnPurgeStatusFailed, CdnPurgeStatusSkipped:
		return true
	default:
		return false
	}
}

// CdnPurgeSkipReasonTooManyTargets is recorded on a job the cap turned away. It
// lives beside the cdn package reasons, which describe a single entry, because
// this one describes the whole job.
const CdnPurgeSkipReasonTooManyTargets = "TOO_MANY_TARGETS"

// CdnPurgeSkipReasonConfigDisabled is recorded when the configuration was turned
// off between the publish and the purge.
const CdnPurgeSkipReasonConfigDisabled = "CONFIG_DISABLED"

var ProjectCdnPurgeJobSortableColumns = map[string]string{
	"createdAt":  "created_at",
	"finishedAt": "finished_at",
	"status":     "status",
}

// ProjectCdnPurgeJob is one publish worth of cache invalidation for one CDN
// configuration.
//
// The targets are carried by the job rather than recomputed later on purpose: a
// publish deletes the rows it published, so what a deleted redirect used to serve
// is only knowable at that moment.
type ProjectCdnPurgeJob struct {
	ID            int64    `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceCode string   `json:"-" gorm:"size:50;index:idx_project_cdn_purge_jobs_ns_proj,priority:1"`
	ProjectCode   string   `json:"-" gorm:"size:50;index:idx_project_cdn_purge_jobs_ns_proj,priority:2"`
	Project       *Project `json:"project" gorm:"foreignKey:NamespaceCode,ProjectCode;references:NamespaceCode,ProjectCode;"`

	CdnConfigID int64             `json:"-" gorm:"not null;index:idx_project_cdn_purge_jobs_config"`
	CdnConfig   *ProjectCdnConfig `json:"cdnConfig" gorm:"foreignKey:CdnConfigID"`
	// CdnConfigName and CdnConfigType are snapshots, so a finished job still reads
	// as what it was even after the configuration is renamed.
	CdnConfigName string `json:"cdnConfigName" gorm:"size:100;not null"`
	CdnConfigType string `json:"cdnConfigType" gorm:"size:50;not null"`

	// ProjectVersion is the version the agents must be serving before the purge is
	// allowed to start.
	ProjectVersion int `json:"projectVersion"`

	Status CdnPurgeStatus `json:"status" gorm:"size:20;not null;index:idx_project_cdn_purge_jobs_status"`

	// Targets is the remaining work. It is dropped once the job succeeds: keeping
	// the addresses of a purge that went through buys nothing and grows the table.
	Targets []cdn.Target `json:"-" gorm:"type:longtext;serializer:json"`
	// Cursor is how far into Targets the worker got, so a run interrupted by a
	// restart resumes instead of purging everything again.
	Cursor int `json:"cursor" gorm:"default:0;not null"`

	Requested int `json:"requested" gorm:"default:0;not null"`
	Purged    int `json:"purged" gorm:"default:0;not null"`
	Failed    int `json:"failed" gorm:"default:0;not null"`
	// Skipped counts the entries that produced no target, by reason.
	Skipped map[string]int `json:"skipped" gorm:"type:longtext;serializer:json"`
	// SkipReason is set when the whole job was turned away.
	SkipReason string `json:"skipReason,omitempty" gorm:"size:50"`

	Attempts int    `json:"attempts" gorm:"default:0;not null"`
	Error    string `json:"error,omitempty" gorm:"size:1000"`

	// WaitedForAgents is false when the job gave up waiting for the agents to serve
	// the published version and purged anyway. The purge then races an agent that
	// still answers with the previous redirect, so the cache may hold it again.
	WaitedForAgents bool `json:"waitedForAgents" gorm:"default:true;not null"`

	// DeadlineAt is when the job stops waiting for the agents and purges anyway.
	DeadlineAt time.Time  `json:"deadlineAt" gorm:"type:timestamp"`
	CreatedAt  time.Time  `json:"createdAt" gorm:"type:timestamp"`
	UpdatedAt  time.Time  `json:"updatedAt" gorm:"type:timestamp"`
	StartedAt  *time.Time `json:"startedAt,omitempty" gorm:"type:timestamp"`
	FinishedAt *time.Time `json:"finishedAt,omitempty" gorm:"type:timestamp"`
}

// CdnDraftChanges is what a publish is moving, expressed the way the publish
// already holds it: the new state of the entries it writes, and the ids of the
// published rows it replaces or removes.
//
// Like ActivityInput, it lives in model rather than service so that the generated
// service mocks stay free of any import back into service.
type CdnDraftChanges struct {
	// RedirectIDs and PageIDs are the published rows a draft replaces or removes.
	// An update appears here as well as in NewEntries: its address may have changed,
	// and the old one has a cached response of its own.
	//
	// A creation does not: the row its draft points at is an empty placeholder with
	// no address yet.
	RedirectIDs []int64
	PageIDs     []int64
	// NewEntries are the addresses the publish writes.
	NewEntries []cdn.Entry
}

// CdnPublishPlan is what a publish has to invalidate, and through which
// configurations. It is collected before the transaction, because a publish
// deletes the rows that hold half of the answer.
type CdnPublishPlan struct {
	NamespaceCode string
	ProjectCode   string
	Entries       []cdn.Entry
	Configs       []ProjectCdnConfig
}

// IsEmpty reports whether there is nothing to enqueue.
func (p CdnPublishPlan) IsEmpty() bool {
	return len(p.Entries) == 0 || len(p.Configs) == 0
}

// Remaining is the slice of targets still to send.
func (j *ProjectCdnPurgeJob) Remaining() []cdn.Target {
	if j == nil || j.Cursor >= len(j.Targets) {
		return nil
	}
	return j.Targets[j.Cursor:]
}

// AddSkipped folds a plan's skip counters into the job's.
func (j *ProjectCdnPurgeJob) AddSkipped(skipped map[cdn.SkipReason]int) {
	if len(skipped) == 0 {
		return
	}
	if j.Skipped == nil {
		j.Skipped = map[string]int{}
	}
	for reason, count := range skipped {
		j.Skipped[string(reason)] += count
	}
}
