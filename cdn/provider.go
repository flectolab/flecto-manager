// Package cdn turns published changes into cache invalidations, independently of
// the CDN doing the work.
//
// A provider only knows how to send one batch of targets to its API. Everything
// else - deciding what is purgeable, expanding host-less entries, batching, rate
// limiting, retrying, persisting - lives outside of it, so adding a CDN is a
// matter of one HTTP client and one descriptor.
package cdn

import "context"

// TargetKind is the shape of a single invalidation. Only KindURL is produced
// today: purging a prefix or a whole hostname invalidates more than what changed,
// which is not something a publish should decide on its own.
//
// The other kinds stay in the type because a provider may only support some of
// them - CloudFront invalidates paths, not absolute URLs - and Capabilities is
// what lets the caller find out before sending anything.
type TargetKind string

const (
	KindURL      TargetKind = "URL"
	KindPrefix   TargetKind = "PREFIX"
	KindHostname TargetKind = "HOSTNAME"
)

// Target is one thing to invalidate, kept structured rather than rendered as a
// string: Cloudflare wants "https://host/path", CloudFront wants "/path" and
// derives the host from the distribution. Each provider renders what it needs.
type Target struct {
	Kind TargetKind `json:"kind"`
	Host string     `json:"host,omitempty"`
	Path string     `json:"path,omitempty"`
}

// Capabilities reports what a provider can invalidate.
type Capabilities struct {
	URL      bool `json:"url"`
	Prefix   bool `json:"prefix"`
	Hostname bool `json:"hostname"`
}

// Supports reports whether the provider handles this kind of target.
func (c Capabilities) Supports(kind TargetKind) bool {
	switch kind {
	case KindURL:
		return c.URL
	case KindPrefix:
		return c.Prefix
	case KindHostname:
		return c.Hostname
	default:
		return false
	}
}

// Limits is what the provider's API allows. The worker sizes its batches and
// paces itself from these values rather than from constants of its own, so a plan
// change is a change in one place.
type Limits struct {
	// MaxTargetsPerRequest is the most targets a single purge call accepts.
	MaxTargetsPerRequest int
	// TargetsPerSecond is the sustained throughput allowed, counted in targets and
	// not in requests: that is how Cloudflare expresses its single-file purge quota.
	TargetsPerSecond int
	// Burst is the size of the token bucket, which is how many targets can be sent
	// at once after an idle period. Zero falls back to TargetsPerSecond.
	Burst int
}

// Provider sends invalidations to one CDN account, zone or distribution.
//
// Implementations are stateless with respect to a purge: the caller owns the
// job, its progress and its retries.
type Provider interface {
	// Type is the registered provider type, for logs and activity payloads.
	Type() string
	// Capabilities reports what this provider can invalidate.
	Capabilities() Capabilities
	// Limits reports the constraints the caller must respect.
	Limits() Limits
	// Verify checks the credentials and the target zone without invalidating
	// anything, so a configuration can be tested from the admin UI.
	Verify(ctx context.Context) error
	// Purge invalidates one batch, already sized to Limits().MaxTargetsPerRequest.
	// It returns an error for the whole batch: providers do not report per-target
	// outcomes, and a purge is idempotent, so the caller retries the batch.
	Purge(ctx context.Context, batch []Target) error
}
