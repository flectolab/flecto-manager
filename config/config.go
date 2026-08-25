package config

import (
	"time"
)

const DefaultRequestTimeout = 2 * time.Second

type Config struct {
	HTTP     HTTPConfig     `mapstructure:"http" validate:"required"`
	DB       DbConfig       `mapstructure:"db" validate:"required"`
	Auth     AuthConfig     `mapstructure:"auth" validate:"required"`
	Page     PageConfig     `mapstructure:"page" validate:"required"`
	Agent    AgentConfig    `mapstructure:"agent" validate:"required"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	Activity ActivityConfig `mapstructure:"activity"`
	Security SecurityConfig `mapstructure:"security"`
	Cdn      CdnConfig      `mapstructure:"cdn"`
}

// CdnConfig bounds the cache invalidation done on publish.
//
// The feature is inert until a project declares a CDN configuration, so the
// switch here is an operational kill switch rather than the way it is turned on.
type CdnConfig struct {
	// Enabled false stops every purge without touching the stored configurations.
	Enabled bool `mapstructure:"enabled"`
	// WorkerInterval is the delay between two runs of the purge task. Zero disables
	// the task, which leaves jobs queued rather than losing them.
	WorkerInterval time.Duration `mapstructure:"worker_interval" validate:"min=0"`
	// AgentConvergenceTimeout is how long a job waits for the agents of the project
	// to serve the published version before purging anyway. Purging earlier lets the
	// CDN cache the old response again, straight from an agent that has not caught
	// up; waiting forever would mean never purging when an agent is gone for good.
	AgentConvergenceTimeout time.Duration `mapstructure:"agent_convergence_timeout" validate:"min=0"`
	// MaxTargetsPerJob caps what a single publish may invalidate. Beyond it the job
	// is recorded as skipped rather than purged: the alternative would be purging a
	// whole hostname, which drops cached objects nobody changed. Zero lifts the cap.
	MaxTargetsPerJob int `mapstructure:"max_targets_per_job" validate:"min=0"`
	// HTTPTimeout bounds a single call to a CDN API.
	HTTPTimeout time.Duration `mapstructure:"http_timeout" validate:"min=0"`
	// JobRetention is how long finished jobs are kept for inspection. Zero keeps
	// them forever.
	JobRetention time.Duration `mapstructure:"job_retention" validate:"min=0"`
}

// SecurityConfig holds the material used to protect data at rest. It is
// deliberately not tied to a feature: any value that must not sit in clear text
// in the database is encrypted with this key.
//
// The key is optional. Without it the instance starts and serves everything that
// needs no encryption, and the features that do report themselves as unavailable
// rather than storing secrets in clear.
type SecurityConfig struct {
	// EncryptionKey is a 32 byte key, given as base64 (optionally prefixed with
	// "base64:") or as "hex:...". Generate one with "flecto-manager security
	// generate-key". Empty leaves encryption unconfigured.
	EncryptionKey string `mapstructure:"encryption_key" validate:"omitempty,encryption_key"`
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Listen  string `mapstructure:"listen"`
}

type HTTPConfig struct {
	Listen string `mapstructure:"listen" validate:"required"`
}
type PageConfig struct {
	SizeLimit      int `mapstructure:"size_limit" validate:"required,min=1"`
	TotalSizeLimit int `mapstructure:"total_size_limit" validate:"required,min=2,gtfield=SizeLimit"`
}

type AuthConfig struct {
	JWT    JWTConfig    `mapstructure:"jwt" validate:"required"`
	OpenID OpenIDConfig `mapstructure:"openid"`
}

type JWTConfig struct {
	Secret          string        `mapstructure:"secret" validate:"required,min=32"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl" validate:"required,min=1m"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl" validate:"required,min=1h"`
	Issuer          string        `mapstructure:"issuer" validate:"required"`
	HeaderName      string        `mapstructure:"header_name"`
}

type OpenIDConfig struct {
	Enabled      bool     `mapstructure:"enabled"`
	Name         string   `mapstructure:"name"`
	Icon         string   `mapstructure:"icon"`
	ProviderURL  string   `mapstructure:"provider_url" validate:"required_if=Enabled true,omitempty,url"`
	ClientID     string   `mapstructure:"client_id" validate:"required_if=Enabled true"`
	ClientSecret string   `mapstructure:"client_secret" validate:"required_if=Enabled true"`
	RedirectURL  string   `mapstructure:"redirect_url" validate:"required_if=Enabled true,omitempty,url"`
	Scopes       []string `mapstructure:"scopes"`
	RolesClaim   string   `mapstructure:"roles_claim"`
}

// DbLogLevel represents the database logging level
type DbLogLevel string

const (
	DbLogLevelSilent DbLogLevel = "silent"
	DbLogLevelError  DbLogLevel = "error"
	DbLogLevelWarn   DbLogLevel = "warn"
	DbLogLevelInfo   DbLogLevel = "info"
)

type DbConfig struct {
	Type     string                 `mapstructure:"type" validate:"required,excludesall=!@#$ "`
	LogLevel DbLogLevel             `mapstructure:"log_level"`
	Config   map[string]interface{} `mapstructure:"config"`
}

// ActivityConfig bounds the activity journal. MaxEventsPerProject is the only
// guard on its size: nb_projects * max_events_per_project rows at most. Zero
// disables the purge, which means accepting an unbounded table.
type ActivityConfig struct {
	MaxEventsPerProject int           `mapstructure:"max_events_per_project" validate:"min=0"`
	PurgeInterval       time.Duration `mapstructure:"purge_interval" validate:"min=0"`
}

type AgentConfig struct {
	OfflineThreshold time.Duration `mapstructure:"offline_threshold" validate:"required,min=1s"`
}

func DefaultConfig() *Config {
	return &Config{
		HTTP: HTTPConfig{Listen: "127.0.0.1:8080"},
		Page: PageConfig{SizeLimit: 1024 * 1024, TotalSizeLimit: 1024 * 1024 * 100},
		Agent: AgentConfig{
			OfflineThreshold: 6 * time.Hour,
		},
		Activity: ActivityConfig{
			MaxEventsPerProject: 1000,
			PurgeInterval:       time.Hour,
		},
		Auth: AuthConfig{
			JWT: JWTConfig{
				Secret:          "", // Must be set via config/env
				AccessTokenTTL:  15 * time.Minute,
				RefreshTokenTTL: 24 * time.Hour,
				Issuer:          "flecto-manager",
				HeaderName:      "Authorization",
			},
			OpenID: OpenIDConfig{
				Enabled: false,
			},
		},
		Metrics: MetricsConfig{
			Enabled: false,
		},
		Cdn: CdnConfig{
			Enabled:                 true,
			WorkerInterval:          30 * time.Second,
			AgentConvergenceTimeout: 30 * time.Minute,
			MaxTargetsPerJob:        5000,
			HTTPTimeout:             10 * time.Second,
			JobRetention:            30 * 24 * time.Hour,
		},
	}
}
