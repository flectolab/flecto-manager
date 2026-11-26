// Package cloudflare purges a Cloudflare zone by URL, and by prefix for redirects
// ignoring the query string, whose cached variants cannot be listed. Hostname
// purges are not used: they would drop cached objects nobody changed.
package cloudflare

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/flectolab/flecto-manager/cdn"
)

// Type is the registered provider type.
const Type = "cloudflare"

const (
	fieldZoneID   = "zone_id"
	fieldAPIToken = "api_token"
	fieldPlan     = "plan"
)

// Plans: batch size and throughput depend on the zone's plan.
const (
	planFree       = "free"
	planPro        = "pro"
	planBusiness   = "business"
	planEnterprise = "enterprise"
)

// planLimits maps a plan to its single-file purge quota, counted in URLs, and to
// its prefix purge quota, counted in requests.
var planLimits = map[string]cdn.Limits{
	planFree: {MaxTargetsPerRequest: 100, TargetsPerSecond: 800, ByKind: map[cdn.TargetKind]cdn.Limits{
		cdn.KindPrefix: {MaxTargetsPerRequest: 100, RequestsPerSecond: 5.0 / 60, Burst: 25},
	}},
	planPro: {MaxTargetsPerRequest: 100, TargetsPerSecond: 1500, ByKind: map[cdn.TargetKind]cdn.Limits{
		cdn.KindPrefix: {MaxTargetsPerRequest: 100, RequestsPerSecond: 5, Burst: 25},
	}},
	planBusiness: {MaxTargetsPerRequest: 100, TargetsPerSecond: 1500, ByKind: map[cdn.TargetKind]cdn.Limits{
		cdn.KindPrefix: {MaxTargetsPerRequest: 100, RequestsPerSecond: 10, Burst: 50},
	}},
	planEnterprise: {MaxTargetsPerRequest: 500, TargetsPerSecond: 3000, ByKind: map[cdn.TargetKind]cdn.Limits{
		cdn.KindPrefix: {MaxTargetsPerRequest: 100, RequestsPerSecond: 50, Burst: 500},
	}},
}

// maxPrefixSeparators is the number of "/" Cloudflare accepts in a prefix.
const maxPrefixSeparators = 31

func init() {
	cdn.Register(Descriptor(), New)
}

// Descriptor describes how a Cloudflare zone is configured.
func Descriptor() cdn.Descriptor {
	return cdn.Descriptor{
		Type:  Type,
		Label: "Cloudflare",
		Doc: "Purges a Cloudflare zone by URL, and by prefix for redirects ignoring the query string. " +
			"The API token needs the Zone.Cache Purge permission on that zone.",
		Fields: []cdn.Field{
			{
				Name:        fieldZoneID,
				Label:       "Zone ID",
				Kind:        cdn.FieldKindText,
				Required:    true,
				Help:        "Found on the overview page of the domain in the Cloudflare dashboard.",
				Placeholder: "023e105f4ecef8ad9ca31a8372d0c353",
			},
			{
				Name:     fieldAPIToken,
				Label:    "API token",
				Kind:     cdn.FieldKindSecret,
				Required: true,
				Help:     "Token with the Zone.Cache Purge permission. Stored encrypted and never displayed again.",
			},
			{
				Name:     cdn.FieldHosts,
				Label:    "Hostnames",
				Kind:     cdn.FieldKindStringList,
				Required: true,
				Help: "The hostnames this project serves through this zone, one per entry. " +
					"A redirect or page written without a hostname (/promo) is purged once for each hostname " +
					"listed here. One written with its own (shop.example.com/soldes) is purged only if that " +
					"hostname is listed. An entry starting with \"*.\" matches any subdomain, but cannot be " +
					"used to complete an entry that has no hostname.",
				Placeholder: "www.example.com",
			},
			{
				Name:     fieldPlan,
				Label:    "Plan",
				Kind:     cdn.FieldKindSelect,
				Required: true,
				Default:  planBusiness,
				Help:     "Sets the purge quota: batch size and throughput depend on it.",
				Options: []cdn.Option{
					{Value: planFree, Label: "Free"},
					{Value: planPro, Label: "Pro"},
					{Value: planBusiness, Label: "Business"},
					{Value: planEnterprise, Label: "Enterprise"},
				},
			},
		},
		Capabilities: cdn.Capabilities{URL: true, Prefix: true},
	}
}

// Provider purges one Cloudflare zone.
type Provider struct {
	zoneID string
	limits cdn.Limits
	client *client
}

// New builds a provider from a validated configuration.
func New(settings, secrets cdn.Values, options cdn.Options) (cdn.Provider, error) {
	limits, ok := planLimits[settings.String(fieldPlan)]
	if !ok {
		return nil, fmt.Errorf("unknown cloudflare plan %q", settings.String(fieldPlan))
	}

	return &Provider{
		zoneID: settings.String(fieldZoneID),
		limits: limits,
		client: newClient(secrets.String(fieldAPIToken), options.HTTPTimeout),
	}, nil
}

func (p *Provider) Type() string {
	return Type
}

// Capabilities leaves hostname purges out: they invalidate more than what was
// published. Prefix purges are only planned for redirects ignoring the query string.
func (p *Provider) Capabilities() cdn.Capabilities {
	return Descriptor().Capabilities
}

func (p *Provider) Limits() cdn.Limits {
	return p.limits
}

// Verify reads the zone, falling back to verifying the token alone when it is
// scoped to purge but not to read the zone (the recommended scope).
func (p *Provider) Verify(ctx context.Context) error {
	err := p.client.do(ctx, http.MethodGet, "/zones/"+p.zoneID, nil)
	if err == nil {
		return nil
	}

	if !isForbidden(err) {
		return err
	}
	return p.client.do(ctx, http.MethodGet, "/user/tokens/verify", nil)
}

// purgeRequest is the body of a single-file purge.
type purgeRequest struct {
	Files []string `json:"files"`
}

// prefixPurgeRequest is the body of a purge by prefix. Cloudflare takes a single
// kind of purge per request.
type prefixPurgeRequest struct {
	Prefixes []string `json:"prefixes"`
}

// Purge sends a batch of a single kind, as cdn.Batch builds them.
func (p *Provider) Purge(ctx context.Context, batch []cdn.Target) error {
	if len(batch) == 0 {
		return nil
	}

	kind := batch[0].Kind
	render := renderURL
	if kind == cdn.KindPrefix {
		render = renderPrefix
	}

	values := make([]string, 0, len(batch))
	for _, target := range batch {
		if target.Kind != kind {
			return fmt.Errorf("cloudflare provider cannot mix %s and %s targets in a batch", kind, target.Kind)
		}
		value, err := render(target)
		if err != nil {
			return err
		}
		values = append(values, value)
	}

	var body any = purgeRequest{Files: values}
	if kind == cdn.KindPrefix {
		body = prefixPurgeRequest{Prefixes: values}
	}
	return p.client.do(ctx, http.MethodPost, "/zones/"+p.zoneID+"/purge_cache", body)
}

// renderURL emits https only: http requests are redirected at the edge and never
// cached, so purging them would waste quota.
func renderURL(target cdn.Target) (string, error) {
	if target.Kind != cdn.KindURL {
		return "", fmt.Errorf("cloudflare provider cannot purge a %s target", target.Kind)
	}
	if target.Host == "" || target.Path == "" {
		return "", fmt.Errorf("cloudflare provider needs a host and a path, got %q%q", target.Host, target.Path)
	}
	return "https://" + target.Host + target.Path, nil
}

// renderPrefix emits host/path: Cloudflare refuses a scheme, a query string or a
// fragment in a prefix.
func renderPrefix(target cdn.Target) (string, error) {
	if target.Kind != cdn.KindPrefix {
		return "", fmt.Errorf("cloudflare provider cannot purge a %s target by prefix", target.Kind)
	}
	if target.Host == "" || target.Path == "" {
		return "", fmt.Errorf("cloudflare provider needs a host and a path, got %q%q", target.Host, target.Path)
	}
	if strings.ContainsAny(target.Path, "?#") {
		return "", fmt.Errorf("cloudflare provider cannot purge a prefix with a query string or a fragment: %q", target.Path)
	}
	if strings.Count(target.Path, "/") > maxPrefixSeparators {
		return "", fmt.Errorf("cloudflare provider cannot purge a prefix with more than %d path separators: %q", maxPrefixSeparators, target.Path)
	}
	return target.Host + target.Path, nil
}
