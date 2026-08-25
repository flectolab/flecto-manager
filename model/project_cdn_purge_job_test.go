package model

import (
	"testing"

	"github.com/flectolab/flecto-manager/cdn"
	"github.com/stretchr/testify/assert"
)

func TestCdnPurgeStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		name   string
		status CdnPurgeStatus
		want   bool
	}{
		{name: "waitingAgents", status: CdnPurgeStatusWaitingAgents, want: false},
		{name: "running", status: CdnPurgeStatusRunning, want: false},
		{name: "succeeded", status: CdnPurgeStatusSucceeded, want: true},
		{name: "partial", status: CdnPurgeStatusPartial, want: true},
		{name: "failed", status: CdnPurgeStatusFailed, want: true},
		{name: "skipped", status: CdnPurgeStatusSkipped, want: true},
		{name: "unknown", status: CdnPurgeStatus("SOMETHING"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.IsTerminal())
		})
	}
}

func TestProjectCdnPurgeJob_Remaining(t *testing.T) {
	targets := []cdn.Target{
		{Kind: cdn.KindURL, Host: "a.com", Path: "/1"},
		{Kind: cdn.KindURL, Host: "a.com", Path: "/2"},
	}

	tests := []struct {
		name string
		job  *ProjectCdnPurgeJob
		want []cdn.Target
	}{
		{
			name: "successFromTheStart",
			job:  &ProjectCdnPurgeJob{Targets: targets},
			want: targets,
		},
		{
			name: "successAfterProgress",
			job:  &ProjectCdnPurgeJob{Targets: targets, Cursor: 1},
			want: targets[1:],
		},
		{
			name: "successWhenDone",
			job:  &ProjectCdnPurgeJob{Targets: targets, Cursor: 2},
			want: nil,
		},
		{
			name: "successWithCursorPastTheEnd",
			job:  &ProjectCdnPurgeJob{Targets: targets, Cursor: 99},
			want: nil,
		},
		{
			name: "successWithNilJob",
			job:  nil,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.job.Remaining())
		})
	}
}

func TestProjectCdnPurgeJob_AddSkipped(t *testing.T) {
	t.Run("successOnAnEmptyJob", func(t *testing.T) {
		job := &ProjectCdnPurgeJob{}

		job.AddSkipped(map[cdn.SkipReason]int{cdn.SkipReasonPattern: 2})

		assert.Equal(t, map[string]int{string(cdn.SkipReasonPattern): 2}, job.Skipped)
	})

	t.Run("successAccumulatesAcrossMerges", func(t *testing.T) {
		job := &ProjectCdnPurgeJob{Skipped: map[string]int{string(cdn.SkipReasonPattern): 1}}

		job.AddSkipped(map[cdn.SkipReason]int{
			cdn.SkipReasonPattern:     2,
			cdn.SkipReasonNoHostMatch: 1,
		})

		assert.Equal(t, map[string]int{
			string(cdn.SkipReasonPattern):     3,
			string(cdn.SkipReasonNoHostMatch): 1,
		}, job.Skipped)
	})

	t.Run("successWithNothingSkipped", func(t *testing.T) {
		job := &ProjectCdnPurgeJob{}

		job.AddSkipped(nil)

		assert.Nil(t, job.Skipped)
	})
}

func TestCdnPublishPlan_IsEmpty(t *testing.T) {
	config := ProjectCdnConfig{Name: "prod"}
	entry := cdn.Entry{Type: "BASIC", Source: "/promo"}

	assert.True(t, CdnPublishPlan{}.IsEmpty())
	assert.True(t, CdnPublishPlan{Entries: []cdn.Entry{entry}}.IsEmpty())
	assert.True(t, CdnPublishPlan{Configs: []ProjectCdnConfig{config}}.IsEmpty())
	assert.False(t, CdnPublishPlan{Entries: []cdn.Entry{entry}, Configs: []ProjectCdnConfig{config}}.IsEmpty())
}
