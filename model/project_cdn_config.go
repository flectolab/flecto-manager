package model

import (
	"time"

	"github.com/flectolab/flecto-manager/cdn"
	"github.com/flectolab/flecto-manager/crypt"
)

// The encrypted serializer must resolve whenever this model is parsed (tests and
// schema loader included); the configured cipher replaces this one at startup.
func init() {
	crypt.RegisterSerializer(crypt.NewUnavailable())
}

var ProjectCdnConfigSortableColumns = map[string]string{
	"name":      "name",
	"type":      "type",
	"enabled":   "enabled",
	"createdAt": "created_at",
	"updatedAt": "updated_at",
}

// ProjectCdnConfig is one CDN a project is served through, matched to purge
// targets by the hostnames it declares.
type ProjectCdnConfig struct {
	ID int64 `json:"id" gorm:"primaryKey;autoIncrement"`
	// The unique index also serves project-scoped lookups (leftmost columns).
	NamespaceCode string   `json:"-" gorm:"size:50;uniqueIndex:idx_project_cdn_configs_name,priority:1"`
	ProjectCode   string   `json:"-" gorm:"size:50;uniqueIndex:idx_project_cdn_configs_name,priority:2"`
	Project       *Project `json:"project" gorm:"foreignKey:NamespaceCode,ProjectCode;references:NamespaceCode,ProjectCode;"`
	Name          string   `json:"name" gorm:"size:100;not null;uniqueIndex:idx_project_cdn_configs_name,priority:3" validate:"required,code"`
	// Type is a provider type registered in the cdn package.
	Type string `json:"type" gorm:"size:50;not null" validate:"required"`
	// Enabled false keeps the configuration but stops its purge jobs.
	Enabled *bool `json:"enabled" gorm:"default:true;not null"`
	// Settings are stored in clear so the admin form never has to decrypt.
	Settings cdn.Values `json:"settings" gorm:"type:longtext;serializer:json"`
	// Secrets are encrypted at rest and never returned to a client.
	Secrets   cdn.Values `json:"-" gorm:"type:longtext;serializer:encrypted"`
	CreatedAt time.Time  `json:"createdAt" gorm:"type:timestamp"`
	UpdatedAt time.Time  `json:"updatedAt" gorm:"type:timestamp"`
}

// ProjectCdnConfigInput is a submitted configuration; Values holds secrets too,
// the descriptor splits them. Lives in model to avoid a mock import cycle.
type ProjectCdnConfigInput struct {
	Name    string
	Type    string
	Enabled *bool
	Values  cdn.Values
}

// IsEnabled reports whether the configuration takes part in a publish; nil means
// enabled.
func (c *ProjectCdnConfig) IsEnabled() bool {
	return c == nil || c.Enabled == nil || *c.Enabled
}

// Hosts are the hostnames this configuration serves.
func (c *ProjectCdnConfig) Hosts() []string {
	if c == nil {
		return nil
	}
	return c.Settings.StringList(cdn.FieldHosts)
}
