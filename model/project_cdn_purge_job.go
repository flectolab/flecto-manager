package model

import (
	"time"

	"github.com/flectolab/flecto-manager/cdn"
)

// ResourceAddress is a redirect or page reduced to its address, so a large
// publish never loads page contents.
type ResourceAddress struct {
	ID     int64
	Type   string
	Source string
}

// CdnPurgeStatus is where a purge job stands.
type CdnPurgeStatus string

const (
	// CdnPurgeStatusWaitingAgents holds a job until the agents serve the published
	// version, so the CDN cannot re-cache a stale response. Only this status merges.
	CdnPurgeStatusWaitingAgents CdnPurgeStatus = "WAITING_AGENTS"
	CdnPurgeStatusRunning       CdnPurgeStatus = "RUNNING"
	CdnPurgeStatusSucceeded     CdnPurgeStatus = "SUCCEEDED"
	// CdnPurgeStatusPartial is a job where only some batches went through.
	CdnPurgeStatusPartial CdnPurgeStatus = "PARTIAL"
	CdnPurgeStatusFailed  CdnPurgeStatus = "FAILED"
	// CdnPurgeStatusSkipped is a job never sent (configuration disabled, or over the cap).
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

// CdnPurgeSkipReasonTooManyTargets is recorded on a job the cap turned away.
const CdnPurgeSkipReasonTooManyTargets = "TOO_MANY_TARGETS"

// CdnPurgeSkipReasonConfigDisabled is recorded when the configuration was
// disabled between the publish and the purge.
const CdnPurgeSkipReasonConfigDisabled = "CONFIG_DISABLED"

var ProjectCdnPurgeJobSortableColumns = map[string]string{
	"createdAt":  "created_at",
	"finishedAt": "finished_at",
	"status":     "status",
}

// ProjectCdnPurgeJob is one publish worth of cache invalidation for one CDN
// configuration. Targets are stored on the job because a publish deletes the rows.
type ProjectCdnPurgeJob struct {
	ID            int64    `json:"id" gorm:"primaryKey;autoIncrement"`
	NamespaceCode string   `json:"-" gorm:"size:50;index:idx_project_cdn_purge_jobs_ns_proj,priority:1"`
	ProjectCode   string   `json:"-" gorm:"size:50;index:idx_project_cdn_purge_jobs_ns_proj,priority:2"`
	Project       *Project `json:"project" gorm:"foreignKey:NamespaceCode,ProjectCode;references:NamespaceCode,ProjectCode;"`

	CdnConfigID int64             `json:"-" gorm:"not null;index:idx_project_cdn_purge_jobs_config"`
	CdnConfig   *ProjectCdnConfig `json:"cdnConfig" gorm:"foreignKey:CdnConfigID"`
	// CdnConfigName and CdnConfigType are snapshots that survive a rename.
	CdnConfigName string `json:"cdnConfigName" gorm:"size:100;not null"`
	CdnConfigType string `json:"cdnConfigType" gorm:"size:50;not null"`

	// ProjectVersion is the version agents must serve before the purge starts.
	ProjectVersion int `json:"projectVersion"`

	Status CdnPurgeStatus `json:"status" gorm:"size:20;not null;index:idx_project_cdn_purge_jobs_status"`

	// Targets is the remaining work, dropped once the job succeeds.
	Targets []cdn.Target `json:"-" gorm:"type:longtext;serializer:json"`
	// Cursor lets an interrupted run resume where it stopped.
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

	// WaitedForAgents is false when the job purged without the agents converging,
	// so the cache may hold the previous response again.
	WaitedForAgents bool `json:"waitedForAgents" gorm:"default:true;not null"`

	// DeadlineAt is when the job stops waiting for the agents.
	DeadlineAt time.Time  `json:"deadlineAt" gorm:"type:timestamp"`
	CreatedAt  time.Time  `json:"createdAt" gorm:"type:timestamp"`
	UpdatedAt  time.Time  `json:"updatedAt" gorm:"type:timestamp"`
	StartedAt  *time.Time `json:"startedAt,omitempty" gorm:"type:timestamp"`
	FinishedAt *time.Time `json:"finishedAt,omitempty" gorm:"type:timestamp"`
}

// CdnDraftChanges is what a publish moves: the entries it writes and the
// published rows it replaces or removes.
type CdnDraftChanges struct {
	// RedirectIDs and PageIDs are the published rows replaced or removed. Updates
	// appear here too (the old address may differ); creations do not.
	RedirectIDs []int64
	PageIDs     []int64
	// NewEntries are the addresses the publish writes.
	NewEntries []cdn.Entry
}

// CdnPublishPlan is what a publish has to invalidate, collected before the
// transaction deletes the published rows.
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

// Remaining returns the targets still to send.
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
