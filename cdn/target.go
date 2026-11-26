package cdn

import (
	"net/url"
	"sort"
	"strings"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
)

// SkipReason says why an entry produced no target.
type SkipReason string

const (
	// SkipReasonPattern is a regex entry: a CDN matches cached objects by URL, not pattern.
	SkipReasonPattern SkipReason = "PATTERN"
	// SkipReasonNoHostMatch is an entry whose host is served by no configuration.
	SkipReasonNoHostMatch SkipReason = "NO_HOST_MATCH"
	// SkipReasonNoHost is a host-less entry with no concrete host to expand it with.
	SkipReasonNoHost SkipReason = "NO_HOST"
	// SkipReasonInvalid is an unparsable entry; published data should never hit it.
	SkipReasonInvalid SkipReason = "INVALID"
	// SkipReasonQueryVariants is an entry ignoring the query string purged by URL
	// only, the provider having no prefix purge: its cached variants stay.
	SkipReasonQueryVariants SkipReason = "QUERY_VARIANTS"
)

// Entry is a published redirect or page whose cached response must be invalidated.
type Entry struct {
	Type   string
	Source string
	// IgnoreQueryString entries are cached under every query string they were
	// requested with, which only a prefix purge reaches.
	IgnoreQueryString bool
}

// RedirectEntry projects a redirect on its source, where the response is cached.
func RedirectEntry(redirect *commonTypes.Redirect) Entry {
	if redirect == nil {
		return Entry{}
	}
	return Entry{
		Type:              string(redirect.Type),
		Source:            redirect.Source,
		IgnoreQueryString: redirect.Options.IsIgnoreQueryString(),
	}
}

// PageEntry projects a page.
func PageEntry(page *commonTypes.Page) Entry {
	if page == nil {
		return Entry{}
	}
	return Entry{Type: string(page.Type), Source: page.Path}
}

// Location is an entry resolved to a host and a path. Host is empty for a BASIC
// entry, whose hosts come from the configuration.
type Location struct {
	Host string
	Path string
}

// Resolve splits the entry into a host and a path, or says why it cannot be purged.
func (e Entry) Resolve() (Location, SkipReason) {
	switch e.Type {
	case string(commonTypes.RedirectTypeBasic):
		if !strings.HasPrefix(e.Source, "/") {
			return Location{}, SkipReasonInvalid
		}
		return Location{Path: e.Source}, ""

	case string(commonTypes.RedirectTypeBasicHost):
		// url.Parse only reads a scheme-less "host/path" as a host behind "//".
		parsed, err := url.Parse("//" + e.Source)
		if err != nil || parsed.Host == "" || parsed.Path == "" {
			return Location{}, SkipReasonInvalid
		}
		// RequestURI, not Path: Path is decoded and drops the query string, which a
		// CDN caches apart.
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

func (p Plan) IsEmpty() bool {
	return len(p.Targets) == 0
}

func (p Plan) SkippedTotal() int {
	total := 0
	for _, count := range p.Skipped {
		total += count
	}
	return total
}

// Planner turns changed entries into targets for one configuration.
type Planner struct {
	// hosts are concrete hostnames, used both to expand host-less entries and to match.
	hosts []string
	// patterns are "*." wildcards: they match a host but cannot expand a host-less
	// entry, "*.example.com" not being an address a purge can be sent to.
	patterns []string
	prefix   bool
}

// NewPlanner builds a planner from normalised, deduplicated and sorted hosts.
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

// WithCapabilities lets the planner use the prefix purge when the provider has one.
func (p *Planner) WithCapabilities(capabilities Capabilities) *Planner {
	p.prefix = capabilities.Supports(KindPrefix)
	return p
}

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
		// Any subdomain depth, but not the apex.
		if strings.HasSuffix(host, pattern[1:]) {
			return true
		}
	}

	return false
}

// Build turns entries into deduplicated targets, sorted so a job's batches are
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

		kind := KindURL
		if entry.IgnoreQueryString && p.prefix {
			kind = KindPrefix
		}

		targets, skip := p.expand(location, kind)
		if skip != "" {
			plan.Skipped[skip]++
			continue
		}
		if entry.IgnoreQueryString && !p.prefix {
			// Purged by URL anyway: the address without query string is still worth it,
			// but its other cached variants stay. Counted once the entry is known to be
			// purgeable, so it is never counted under two reasons.
			plan.Skipped[SkipReasonQueryVariants]++
		}

		for _, target := range targets {
			if _, ok := seen[target]; ok {
				continue
			}
			seen[target] = struct{}{}
			plan.Targets = append(plan.Targets, target)
		}
	}

	SortTargets(plan.Targets)
	return plan
}

// SortTargets groups targets by kind, so a batch is rarely cut short by a change
// of kind, then sorts them by host and path.
func SortTargets(targets []Target) {
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Kind != targets[j].Kind {
			return targets[i].Kind < targets[j].Kind
		}
		if targets[i].Host != targets[j].Host {
			return targets[i].Host < targets[j].Host
		}
		return targets[i].Path < targets[j].Path
	})
}

// expand turns a host-less location into one target per served host.
func (p *Planner) expand(location Location, kind TargetKind) ([]Target, SkipReason) {
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
			targets = append(targets, Target{Kind: kind, Host: host, Path: path})
		}
	}
	return targets, ""
}

// PathVariants returns the encoded path, plus the decoded one when it differs,
// since CDNs differ on which form they key their cache on. Decoding is skipped
// when it would yield a space, "%", "?" or "#": a different or rejected address
// that would fail the whole batch.
func PathVariants(path string) []string {
	// The placeholder host keeps a path starting with "//" from being read as a host.
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

func isSendablePath(path string) bool {
	for _, r := range path {
		if r <= ' ' || r == 0x7f || r == '%' || r == '?' || r == '#' {
			return false
		}
	}
	return true
}

// Batch cuts targets into slices of a single kind, each holding at most the
// MaxTargetsPerRequest of that kind; zero or less puts no bound on a batch.
func Batch(targets []Target, limits Limits) [][]Target {
	var batches [][]Target

	for start := 0; start < len(targets); {
		kind := targets[start].Kind
		size := limits.For(kind).MaxTargetsPerRequest

		end := start + 1
		for end < len(targets) && targets[end].Kind == kind && (size <= 0 || end-start < size) {
			end++
		}
		batches = append(batches, targets[start:end])
		start = end
	}

	return batches
}

// normalizeHost lowercases and trims a hostname. The port is kept: a service on a
// non standard port is not the one behind the CDN.
func normalizeHost(host string) string {
	return strings.ToLower(strings.TrimSpace(host))
}
