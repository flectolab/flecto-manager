package model

import (
	"time"

	"github.com/flectolab/flecto-manager/cdn"
	"github.com/flectolab/flecto-manager/crypt"
)

// The encrypted serializer has to resolve as soon as this model is parsed, which
// happens in every binary that opens the database, tests and the schema loader
// included. Registering the unavailable cipher here makes the name always valid;
// reading the configuration replaces it with the real one before anything is
// written, and a write with no key configured fails rather than storing clear
// text.
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

// ProjectCdnConfig is one CDN a project is served through. A project can have
// several: one per zone or per distribution, each declaring the hostnames it
// covers, which is how a purge target finds the configuration it belongs to.
//
// It is administered from the admin section, under project administration rights:
// it carries the credentials of a third-party account.
type ProjectCdnConfig struct {
	ID int64 `json:"id" gorm:"primaryKey;autoIncrement"`
	// The unique index on (namespace, project, name) doubles as the lookup index:
	// every query of this table starts by narrowing to a project, and its leftmost
	// columns are exactly that.
	NamespaceCode string   `json:"-" gorm:"size:50;uniqueIndex:idx_project_cdn_configs_name,priority:1"`
	ProjectCode   string   `json:"-" gorm:"size:50;uniqueIndex:idx_project_cdn_configs_name,priority:2"`
	Project       *Project `json:"project" gorm:"foreignKey:NamespaceCode,ProjectCode;references:NamespaceCode,ProjectCode;"`
	Name          string   `json:"name" gorm:"size:100;not null;uniqueIndex:idx_project_cdn_configs_name,priority:3" validate:"required,code"`
	// Type is a provider type registered in the cdn package.
	Type string `json:"type" gorm:"size:50;not null" validate:"required"`
	// Enabled turns the configuration off without losing it: a disabled
	// configuration keeps its zone and its credentials, it simply stops receiving
	// purge jobs.
	Enabled *bool `json:"enabled" gorm:"default:true;not null"`
	// Settings are the provider values stored in clear. They are shown in the admin
	// form, so keeping them apart from the secrets means rendering the form never
	// has to decrypt anything.
	Settings cdn.Values `json:"settings" gorm:"type:longtext;serializer:json"`
	// Secrets are the provider values stored encrypted, using the key of
	// security.encryption_key. They are never returned to a client.
	Secrets   cdn.Values `json:"-" gorm:"type:longtext;serializer:encrypted"`
	CreatedAt time.Time  `json:"createdAt" gorm:"type:timestamp"`
	UpdatedAt time.Time  `json:"updatedAt" gorm:"type:timestamp"`
}

// ProjectCdnConfigInput is a submitted configuration. Values holds every provider
// field, secrets included: the descriptor is what splits them apart.
//
// Like ActivityInput, it lives in model rather than service so that the generated
// service mocks stay free of any import back into service, which would make the
// service tests cycle.
type ProjectCdnConfigInput struct {
	Name    string
	Type    string
	Enabled *bool
	Values  cdn.Values
}

// IsEnabled reports whether the configuration takes part in a publish, nil
// meaning enabled: a row written before the column existed is not a reason to
// stop purging.
func (c *ProjectCdnConfig) IsEnabled() bool {
	return c == nil || c.Enabled == nil || *c.Enabled
}

// Hosts are the hostnames this configuration serves, read from the conventional
// provider field.
func (c *ProjectCdnConfig) Hosts() []string {
	if c == nil {
		return nil
	}
	return c.Settings.StringList(cdn.FieldHosts)
}
