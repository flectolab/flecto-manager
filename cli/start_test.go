package cli

import (
	"syscall"
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/config"
	"github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/database"
	"github.com/labstack/echo/v4"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetStartRunFn_SuccessOnlyListenHTTP(t *testing.T) {
	database.FactoryDialector[database.DbTypeSqlite] = database.CreateDialectorSqlite
	ctx := context.TestContext(nil)
	ctx.Config.DB = config.DbConfig{
		Type:   database.DbTypeSqlite,
		Config: map[string]interface{}{"dsn": ":memory:"},
	}
	ctx.Config.HTTP.Listen = "127.0.0.1:0"
	ctx.Config.Auth = config.AuthConfig{
		JWT: config.JWTConfig{
			Secret:          "test-secret-key-for-jwt-min-32-chars!",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 168 * time.Hour,
			Issuer:          "flecto-manager-test",
			HeaderName:      "Authorization",
		},
		OpenID: config.OpenIDConfig{Enabled: false},
	}
	viper.Reset()
	viper.SetFs(afero.NewMemMapFs())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cmd := GetStartCmd(ctx)
	go func() {
		err := GetStartRunFn(ctx)(cmd, []string{})
		assert.NoError(t, err)
	}()
	time.Sleep(time.Millisecond * 500)
	ctx.Signal() <- syscall.SIGINT
}

func TestGetStartRunFn_FailPortAlreadyBind(t *testing.T) {
	database.FactoryDialector[database.DbTypeSqlite] = database.CreateDialectorSqlite
	ctx := context.TestContext(nil)
	ctx.Config.DB = config.DbConfig{
		Type:   database.DbTypeSqlite,
		Config: map[string]interface{}{"dsn": ":memory:"},
	}
	ctx.Config.Auth = config.AuthConfig{
		JWT: config.JWTConfig{
			Secret:          "test-secret-key-for-jwt-min-32-chars!",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 168 * time.Hour,
			Issuer:          "flecto-manager-test",
			HeaderName:      "Authorization",
		},
		OpenID: config.OpenIDConfig{Enabled: false},
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	go func() {
		_ = e.Start("127.0.0.1:0")
	}()
	time.Sleep(time.Millisecond * 500)

	ctx.Config.HTTP.Listen = e.Listener.Addr().String()
	cmd := GetStartCmd(ctx)

	assert.Panics(t, func() {
		_ = GetStartRunFn(ctx)(cmd, []string{})
	})
	_ = e.Close()
}

func TestGetStartRunFn_WithMetricsEnabled(t *testing.T) {
	database.FactoryDialector[database.DbTypeSqlite] = database.CreateDialectorSqlite
	ctx := context.TestContext(nil)
	ctx.Config.DB = config.DbConfig{
		Type:   database.DbTypeSqlite,
		Config: map[string]interface{}{"dsn": ":memory:"},
	}
	ctx.Config.HTTP.Listen = "127.0.0.1:0"
	ctx.Config.Auth = config.AuthConfig{
		JWT: config.JWTConfig{
			Secret:          "test-secret-key-for-jwt-min-32-chars!",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 168 * time.Hour,
			Issuer:          "flecto-manager-test",
			HeaderName:      "Authorization",
		},
		OpenID: config.OpenIDConfig{Enabled: false},
	}
	ctx.Config.Metrics = config.MetricsConfig{
		Enabled: true,
		Listen:  "",
	}
	ctx.Config.Agent.OfflineThreshold = 6 * time.Hour
	viper.Reset()
	viper.SetFs(afero.NewMemMapFs())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cmd := GetStartCmd(ctx)
	go func() {
		err := GetStartRunFn(ctx)(cmd, []string{})
		assert.NoError(t, err)
	}()
	time.Sleep(time.Millisecond * 500)
	ctx.Signal() <- syscall.SIGINT
}

func TestGetStartRunFn_WithSeparateMetricsServer(t *testing.T) {
	database.FactoryDialector[database.DbTypeSqlite] = database.CreateDialectorSqlite
	ctx := context.TestContext(nil)
	ctx.Config.DB = config.DbConfig{
		Type:   database.DbTypeSqlite,
		Config: map[string]interface{}{"dsn": ":memory:"},
	}
	ctx.Config.HTTP.Listen = "127.0.0.1:0"
	ctx.Config.Auth = config.AuthConfig{
		JWT: config.JWTConfig{
			Secret:          "test-secret-key-for-jwt-min-32-chars!",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 168 * time.Hour,
			Issuer:          "flecto-manager-test",
			HeaderName:      "Authorization",
		},
		OpenID: config.OpenIDConfig{Enabled: false},
	}
	ctx.Config.Metrics = config.MetricsConfig{
		Enabled: true,
		Listen:  "127.0.0.1:0",
	}
	ctx.Config.Agent.OfflineThreshold = 6 * time.Hour
	viper.Reset()
	viper.SetFs(afero.NewMemMapFs())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cmd := GetStartCmd(ctx)
	go func() {
		err := GetStartRunFn(ctx)(cmd, []string{})
		assert.NoError(t, err)
	}()
	time.Sleep(time.Millisecond * 500)
	ctx.Signal() <- syscall.SIGINT
}
