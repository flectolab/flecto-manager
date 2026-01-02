package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	got := DefaultConfig()
	assert.Equal(t,
		&Config{
			HTTP: HTTPConfig{
				Listen: "127.0.0.1:8080",
			},
			Page: PageConfig{SizeLimit: 1024 * 1024, TotalSizeLimit: 1024 * 1024 * 100},
			Agent: AgentConfig{
				OfflineThreshold: 6 * time.Hour,
			},
			Auth: AuthConfig{
				JWT: JWTConfig{
					Secret:          "",
					AccessTokenTTL:  15 * time.Minute,
					RefreshTokenTTL: 24 * time.Hour,
					Issuer:          "flecto-manager",
					HeaderName:      "Authorization",
				},
				OpenID: OpenIDConfig{
					Enabled: false,
				},
			},
		},
		got,
	)
}
