package cli

import (
	"fmt"
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/config"
	"github.com/flectolab/flecto-manager/context"
	"github.com/stretchr/testify/assert"
)

func Test_validateConfig(t *testing.T) {

	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			cfg: &config.Config{
				HTTP: config.HTTPConfig{Listen: "127.0.0.1:8080"},
				DB: config.DbConfig{
					Type:   "mysql",
					Config: map[string]interface{}{"dsn": "flecto:flecto@tcp(127.0.0.1:3306)/flecto"},
				},
				Auth: config.AuthConfig{
					JWT: config.JWTConfig{
						Secret:          "test-secret-key-for-jwt-min-32-chars!",
						AccessTokenTTL:  15 * time.Minute,
						RefreshTokenTTL: 7 * 24 * time.Hour,
						Issuer:          "flecto-manager-test",
						HeaderName:      "Authorization",
					},
					OpenID: config.OpenIDConfig{Enabled: false},
				},
				Page: config.PageConfig{
					SizeLimit:      1024,
					TotalSizeLimit: 2048,
				},
				Agent: config.AgentConfig{
					OfflineThreshold: 1 * time.Hour,
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "failedWithInvalidConfig",
			cfg: &config.Config{
				HTTP: config.HTTPConfig{Listen: ""},
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TestContext(nil)
			ctx.Config = tt.cfg
			tt.wantErr(t, validateConfig(ctx), fmt.Sprintf("validateConfig(%v)", ctx))
		})
	}
}
