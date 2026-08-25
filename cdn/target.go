package cdn

import (
	"net/url"
	"sort"
	"strings"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
)

// SkipReason says why an entry produced no target. Every skipped entry is
// counted and reported in the activity trail: a purge that quietly covers half of
// what changed is worse than one that says what it left out.
type SkipReason string

const (
	// SkipReasonPattern is a regex based entry. A CDN matches cached objects by
	// URL, never by pattern, so there is nothing to ask it to purge.
	SkipReasonPattern SkipReason = "PATTERN"
	// SkipReasonNoHostMatch is an entry whose host is served by no configuration.
	SkipReasonNoHostMatch SkipReason = "NO_HOST_MATCH"
	// SkipReasonNoHost is a host-less entry met by a configuration declaring no
	// concrete host to expand it with.
	SkipReasonNoHost SkipReason = "NO_HOST"
	// SkipReasonInvalid is an entry that does not parse. It should not happen on
	// published data, which is validated on the way in, so it is kept apart from
	// the expected reasons.
	SkipReasonInvalid SkipReason = "INVALID"
)

// Entry is a published redirect or page whose cached response must be
// invalidated. Redirects and pages share the same shape here because they share
// the same addressing: a type telling whether the source carries a host, and the
// source itself.
type Entry struct {
	Type   string
	Source string
}

// RedirectEntry projects a redirect. The invalidated URL is its source: what a
// CDN caches is the redirect response served at that address, not its target.
func RedirectEntry(redirect *commonTypes.Redirect) Entry {
	if redirect == nil {
		return Entry{}
	}
	return Entry{Type: string(redirect.Type), Source: redirect.Source}
}

// PageEntry projects a page.
func PageEntry(page *commonTypes.Page) Entry {
	if page == nil {
		return Entry{}
	}
	return Entry{Type: string(page.Type), Source: page.Path}
}

// Location is an entry resolved to a host and a path. Host is empty when the
// entry does not carry one, which is the normal case for a BASIC redirect: it
// applies whatever the requested host, so the hosts come from the configuration.
type Location struct {
	Host string
	Path string
}

// Resolve splits the entry into a host and a path, or says why it cannot be
// purged.
func (e Entry) Resolve() (Location, SkipReason) {
	switch e.Type {
	case string(commonTypes.RedirectTypeBasic):
		if !strings.HasPrefix(e.Source, "/") {
			return Location{}, SkipReasonInvalid
		}
		return Location{Path: e.Source}, ""

	case string(commonTypes.RedirectTypeBasicHost):
		// The source is "host/path" with no scheme, which url.Parse only reads as a
		// host when it is introduced by "//" - the same trick the redirect validator
		// uses on the way in.
		parsed, err := url.Parse("//" + e.Source)
		if err != nil || parsed.Host == "" || parsed.Path == "" {
			return Location{}, SkipReasonInvalid
		}
		// RequestURI rather than Path: Path is decoded and drops the query string,
		// while a CDN caches "/promo?a=1" apart from "/promo".
		return Location{Host: normalizeHost(parsed.Host), Path: parsed.RequestURI()}, ""

	case string(commonTypes.RedirectTypeRegex), string(commonTypes.RedirectTypeRegexHost):
		return Location{}, SkipReasonPattern

	default:
		return Location{}, SkipReasonInvalid
	}
}

// Plan is what one configuration has to purge for a set of changed entries.
type Plan struct {
	Targets []Target
	Skipped map[SkipReason]int
}

// IsEmpty reports whether there is nothing to send.
func (p Plan) IsEmpty() bool {
	return len(p.Targets) == 0
}

// SkippedTotal counts the entries left out, all reasons together.
func (p Plan) SkippedTotal() int {
	total := 0
	for _, count := range p.Skipped {
		total += count
	}
	return total
}

// Planner turns changed entries into targets for one configuration.
//
// The hosts it holds serve two purposes: they supply the missing host of an entry
// that has none, and they decide whether an entry carrying a host belongs to this
// configuration at all.
type Planner struct {
	// hosts are the concrete hostnames served, used both to expand and to match.
	hosts []string
	// patterns are the wildcard entries. They can match an entry that carries its
	// own host, but they cannot expand a host-less one: "*.example.com" is not an
	// address a purge can be sent to.
	patterns []string
}

// NewPlanner builds a planner for the hosts of one configuration. Hosts are
// normalised, deduplicated and sorted, so two configurations listing the same
// hosts in a different order plan identically.
func NewPlanner(hosts []string) *Planner {
	planner := &Planner{}
	seen := make(map[string]struct{}, len(hosts))

	for _, host := range hosts {
		host = normalizeHost(host)
		if host == "" {
			continue
		}
		if _, ok := seen[host]; ok {
			continue
		}
		seen[host] = struct{}{}

		if strings.HasPrefix(host, "*.") {
			planner.patterns = append(planner.patterns, host)
			continue
		}
		planner.hosts = append(planner.hosts, host)
	}

	sort.Strings(planner.hosts)
	sort.Strings(planner.patterns)
	return planner
}

// Hosts returns the concrete hosts of the planner.
func (p *Planner) Hosts() []string {
	return p.hosts
}

// Serves reports whether a host belongs to this configuration.
func (p *Planner) Serves(host string) bool {
	host = normalizeHost(host)

	for _, candidate := range p.hosts {
		if candidate == host {
			return true
		}
	}
	for _, pattern := range p.patterns {
		// "*.example.com" covers any subdomain, at any depth, but not the apex: a
		// suffix match on ".example.com" is exactly that.
		if strings.HasSuffix(host, pattern[1:]) {
			return true
		}
	}

	return false
}

// Build turns entries into the targets this configuration has to purge.
//
// Duplicates are removed and the result is sorted: a publish touching the same
// path twice must not purge it twice, and a stable order keeps a job's batches
// reproducible across restarts.
func (p *Planner) Build(entries []Entry) Plan {
	plan := Plan{Skipped: map[SkipReason]int{}}
	seen := map[Target]struct{}{}

	for _, entry := range entries {
		location, skip := entry.Resolve()
		if skip != "" {
			plan.Skipped[skip]++
			continue
		}

		targets, skip := p.expand(location)
		if skip != "" {
			plan.Skipped[skip]++
			continue
		}

		for _, target := range targets {
			if _, ok := seen[target]; ok {
				continue
			}
			seen[target] = struct{}{}
			plan.Targets = append(plan.Targets, target)
		}
	}

	sort.Slice(plan.Targets, func(i, j int) bool {
		if plan.Targets[i].Host != plan.Targets[j].Host {
			return plan.Targets[i].Host < plan.Targets[j].Host
		}
		return plan.Targets[i].Path < plan.Targets[j].Path
	})

	return plan
}

// expand turns one location into the targets to purge for this configuration. A
// host-less location becomes one target per served host, which is why a project
// serving three domains purges three URLs for a single path-only redirect.
func (p *Planner) expand(location Location) ([]Target, SkipReason) {
	hosts := p.hosts
	if location.Host != "" {
		if !p.Serves(location.Host) {
			return nil, SkipReasonNoHostMatch
		}
		hosts = []string{location.Host}
	}

	if len(hosts) == 0 {
		return nil, SkipReasonNoHost
	}

	paths := PathVariants(location.Path)
	targets := make([]Target, 0, len(hosts)*len(paths))
	for _, host := range hosts {
		for _, path := range paths {
			targets = append(targets, Target{Kind: KindURL, Host: host, Path: path})
		}
	}
	return targets, ""
}

// PathVariants returns the forms under which a CDN may hold a path in cache. A
// path is written either percent-encoded (/caf%C3%A9) or decoded (/café), and
// which of the two a CDN keys its cache on depends on how it normalises a request.
// Purging both costs one URL of quota and settles it.
//
// The encoded form always comes first and always comes back: it is what a client
// sends. The decoded form is added only when it differs and stays a valid URL
// path - decoding a space, "%", "?" or "#" yields an address that either means
// something else or that the CDN rejects, failing the whole batch with it. The
// query string is kept as written in both.
func PathVariants(path string) []string {
	// The placeholder host keeps a path starting with "//" from being read as one.
	parsed, err := url.Parse("http://placeholder" + path)
	if err != nil {
		return []string{path}
	}

	query := ""
	if parsed.RawQuery != "" {
		query = "?" + parsed.RawQuery
	}

	escaped := parsed.EscapedPath() + query
	variants := []string{escaped}
	if decoded := parsed.Path + query; decoded != escaped && isSendablePath(parsed.Path) {
		variants = append(variants, decoded)
	}
	return variants
}

// isSendablePath reports whether a decoded path can be sent as is.
func isSendablePath(path string) bool {
	for _, r := range path {
		if r <= ' ' || r == 0x7f || r == '%' || r == '?' || r == '#' {
			return false
		}
	}
	return true
}

// Batch cuts targets into slices of at most size, which is how a provider's
// MaxTargetsPerRequest is applied. A size of zero or less yields a single batch:
// a provider that declares no limit is asking for everything at once.
func Batch(targets []Target, size int) [][]Target {
	if len(targets) == 0 {
		return nil
	}
	if size <= 0 {
		return [][]Target{targets}
	}

	batches := make([][]Target, 0, (len(targets)+size-1)/size)
	for start := 0; start < len(targets); start += size {
		end := start + size
		if end > len(targets) {
			end = len(targets)
		}
		batches = append(batches, targets[start:end])
	}

	return batches
}

// normalizeHost lowercases a hostname and trims the spacing an operator may have
// left when pasting a list. Hosts are compared as written otherwise, port
// included: a service exposed on a non standard port is not the one behind the
// CDN.
func normalizeHost(host string) string {
	return strings.ToLower(strings.TrimSpace(host))
}
