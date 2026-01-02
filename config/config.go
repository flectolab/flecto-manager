package config

import (
	"time"
)

const DefaultRequestTimeout = 2 * time.Second

type Config struct {
	HTTP HTTPConfig `mapstructure:"http" validate:"required"`
	DB   DbConfig   `mapstructure:"db" validate:"required"`
	Auth AuthConfig `mapstructure:"auth" validate:"required"`
	Page PageConfig `mapstructure:"page" validate:"required"`
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
	Enabled      bool   `mapstructure:"enabled"`
	ProviderURL  string `mapstructure:"provider_url" validate:"required_if=Enabled true,omitempty,url"`
	ClientID     string `mapstructure:"client_id" validate:"required_if=Enabled true"`
	ClientSecret string `mapstructure:"client_secret" validate:"required_if=Enabled true"`
	RedirectURL  string `mapstructure:"redirect_url" validate:"required_if=Enabled true,omitempty,url"`
}

type DbConfig struct {
	Type   string                 `mapstructure:"type" validate:"required,excludesall=!@#$ "`
	Config map[string]interface{} `mapstructure:"config"`
}

func DefaultConfig() *Config {
	return &Config{
		HTTP: HTTPConfig{Listen: "127.0.0.1:8080"},
		Page: PageConfig{SizeLimit: 1024 * 1024, TotalSizeLimit: 1024 * 1024 * 100},
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
	}
}
