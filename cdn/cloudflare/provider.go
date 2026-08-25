// Package cloudflare purges a Cloudflare zone by URL.
//
// Only single-file purge is used. Purging by prefix or by hostname would drop
// cached objects nobody changed, which is not something a publish gets to decide.
package cloudflare

import (
	"context"
	"fmt"
	"net/http"

	"github.com/flectolab/flecto-manager/cdn"
)

// Type is the registered provider type.
const Type = "cloudflare"

const (
	fieldZoneID   = "zone_id"
	fieldAPIToken = "api_token"
	fieldPlan     = "plan"
)

// Plans and their purge quotas, as published by Cloudflare. They are read from
// the configuration rather than hard-coded to one value because both the batch
// size and the throughput depend on the zone's plan.
const (
	planFree       = "free"
	planPro        = "pro"
	planBusiness   = "business"
	planEnterprise = "enterprise"
)

// planLimits maps a plan to what its single-file purge allows: how many URLs a
// request accepts, and how many URLs per second the account may sustain.
var planLimits = map[string]cdn.Limits{
	planFree:       {MaxTargetsPerRequest: 100, TargetsPerSecond: 800},
	planPro:        {MaxTargetsPerRequest: 100, TargetsPerSecond: 1500},
	planBusiness:   {MaxTargetsPerRequest: 100, TargetsPerSecond: 1500},
	planEnterprise: {MaxTargetsPerRequest: 500, TargetsPerSecond: 3000},
}

func init() {
	cdn.Register(Descriptor(), New)
}

// Descriptor describes how a Cloudflare zone is configured.
func Descriptor() cdn.Descriptor {
	return cdn.Descriptor{
		Type:  Type,
		Label: "Cloudflare",
		Doc:   "Purges a Cloudflare zone by URL. The API token needs the Zone.Cache Purge permission on that zone.",
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
		Capabilities: cdn.Capabilities{URL: true},
	}
}

// Provider purges one Cloudflare zone.
type Provider struct {
	zoneID string
	limits cdn.Limits
	client *client
}

// New builds a provider from a configuration already validated against the
// descriptor.
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

// Capabilities reports URL only. Cloudflare also purges by prefix and by
// hostname on every plan, but both invalidate more than what was published.
func (p *Provider) Capabilities() cdn.Capabilities {
	return cdn.Capabilities{URL: true}
}

func (p *Provider) Limits() cdn.Limits {
	return p.limits
}

// Verify checks the credentials. It reads the zone first, which also confirms the
// zone id, and falls back to verifying the token alone when the token is scoped
// narrowly enough to purge but not to read the zone - the recommended scope.
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

// purgeRequest is the body of a single-file purge. Files are plain URL strings:
// the object form exists to name the headers of a custom cache key, which
// nothing here produces.
type purgeRequest struct {
	Files []string `json:"files"`
}

func (p *Provider) Purge(ctx context.Context, batch []cdn.Target) error {
	if len(batch) == 0 {
		return nil
	}

	files := make([]string, 0, len(batch))
	for _, target := range batch {
		url, err := renderURL(target)
		if err != nil {
			return err
		}
		files = append(files, url)
	}

	return p.client.do(ctx, http.MethodPost, "/zones/"+p.zoneID+"/purge_cache", purgeRequest{Files: files})
}

// renderURL turns a target into the absolute URL Cloudflare purges. Only https
// is emitted: a zone redirecting http to https answers those at the edge, so
// purging them would spend quota on responses that were never cached from the
// origin.
func renderURL(target cdn.Target) (string, error) {
	if target.Kind != cdn.KindURL {
		return "", fmt.Errorf("cloudflare provider cannot purge a %s target", target.Kind)
	}
	if target.Host == "" || target.Path == "" {
		return "", fmt.Errorf("cloudflare provider needs a host and a path, got %q%q", target.Host, target.Path)
	}
	return "https://" + target.Host + target.Path, nil
}
