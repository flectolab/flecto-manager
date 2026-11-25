// Package cdn turns published changes into cache invalidations, independently of
// the CDN. A provider only sends one batch of targets to its API; purgeability,
// batching, rate limiting, retries and persistence live outside of it.
package cdn

import "context"

// TargetKind is the shape of a single invalidation. Only KindURL is produced:
// prefix or hostname purges would drop objects nobody changed.
type TargetKind string

const (
	KindURL      TargetKind = "URL"
	KindPrefix   TargetKind = "PREFIX"
	KindHostname TargetKind = "HOSTNAME"
)

// Target is one thing to invalidate, kept structured so each provider renders the
// form it needs (absolute URL, bare path...).
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

// Limits is what the provider's API allows; the worker sizes and paces batches from it.
type Limits struct {
	MaxTargetsPerRequest int
	// TargetsPerSecond is counted in targets, not requests, like Cloudflare's quota.
	TargetsPerSecond int
	// Burst is the token bucket size. Zero falls back to TargetsPerSecond.
	Burst int
}

// Provider sends invalidations to one CDN zone. It is stateless: the caller owns
// the job, its progress and its retries.
type Provider interface {
	Type() string
	Capabilities() Capabilities
	Limits() Limits
	// Verify checks the credentials and the zone without invalidating anything.
	Verify(ctx context.Context) error
	// Purge invalidates one batch. Errors cover the whole batch, which is
	// idempotent and retried as a unit.
	Purge(ctx context.Context, batch []Target) error
}
