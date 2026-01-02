package cli

import (
	"fmt"
	"io"
	"testing"

	"github.com/flectolab/flecto-manager/config"
	"github.com/flectolab/flecto-manager/context"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func Test_initConfig_Success(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	fs := afero.NewMemMapFs()
	viper.Reset()
	viper.SetFs(fs)
	path := "/app"
	_ = fs.Mkdir(path, 0775)
	_ = afero.WriteFile(fs, fmt.Sprintf("%s/config.yml", path), []byte("accept_type_files: []"), 0644)
	want := config.DefaultConfig()
	initConfig(ctx, cmd)
	assert.Equal(t, want, ctx.Config)
}

func Test_initConfig_SuccessWithConfigFlag(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	fs := afero.NewMemMapFs()
	viper.Reset()
	viper.SetFs(fs)
	path := "/foo"
	_ = fs.Mkdir(path, 0775)
	_ = afero.WriteFile(fs, fmt.Sprintf("%s/foo.yml", path), []byte("accept_type_files: []"), 0644)
	want := config.DefaultConfig()
	viper.Set(ConfigName, fmt.Sprintf("%s/foo.yml", path))
	initConfig(ctx, cmd)
	assert.Equal(t, want, ctx.Config)
}

func Test_initConfig_FailReadConfig(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	fs := afero.NewMemMapFs()
	viper.Reset()
	viper.SetFs(fs)

	want := config.DefaultConfig()
	initConfig(ctx, cmd)
	assert.Equal(t, want, ctx.Config)
}

func Test_initConfig_FailUnmarshal(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	fs := afero.NewMemMapFs()
	viper.Reset()
	viper.SetFs(fs)
	path := GetDefaultConfigPath()
	_ = fs.Mkdir(path, 0775)
	_ = afero.WriteFile(fs, fmt.Sprintf("%s/%s.yml", path, ConfigName), []byte("http: {listen: []}"), 0644)
	defer func() {
		if r := recover(); r != nil {
			assert.True(t, true)
		} else {
			t.Errorf("initConfig should have panicked")
		}
	}()
	initConfig(ctx, cmd)
}

func TestGetRootPreRunEFn_Success(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	path := GetDefaultConfigPath()
	fs := afero.NewMemMapFs()
	_ = fs.Mkdir(path, 0775)
	globalStr := `
db:
  type: mysql
  config:
    dsn: flecto:flecto@tcp(127.0.0.1:3306)/flecto
auth:
  jwt:
    secret: "test-secret-key-for-jwt-min-32-chars!"
    access_token_ttl: 15m
    refresh_token_ttl: 168h
    issuer: "flecto-manager-test"
    header_name: "Authorization"
  openid:
    enabled: false`
	_ = afero.WriteFile(fs, fmt.Sprintf("%s/%s.yml", path, ConfigName), []byte(globalStr+"\n"), 0644)
	viper.Reset()
	viper.SetFs(fs)
	err := GetRootPreRunEFn(ctx, true)(cmd, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "LevelVar(INFO)", ctx.LogLevel.String())
}

func TestGetRootPreRunEFn_SuccessLogLevelFlag(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	path := GetDefaultConfigPath()
	fs := afero.NewMemMapFs()
	_ = fs.Mkdir(path, 0775)
	globalStr := `auth:
  jwt:
    secret: "test-secret-key-for-jwt-min-32-chars!"
    access_token_ttl: 15m
    refresh_token_ttl: 168h
    issuer: "flecto-manager-test"
    header_name: "Authorization"
  openid:
    enabled: false`
	_ = afero.WriteFile(fs, fmt.Sprintf("%s/%s.yml", path, ConfigName), []byte(globalStr+"\n"), 0644)
	viper.Reset()
	viper.SetFs(fs)
	cmd.SetArgs([]string{
		"--" + LogLevel, "ERROR"},
	)
	_ = cmd.Execute()
	err := GetRootPreRunEFn(ctx, false)(cmd, []string{})
	assert.NoError(t, err)
	assert.Equal(t, "LevelVar(ERROR)", ctx.LogLevel.String())
}

func TestGetRootPreRunEFn_FailLogLevelFlagInvalid(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	path := GetDefaultConfigPath()
	fs := afero.NewMemMapFs()
	_ = fs.Mkdir(path, 0775)
	globalStr := `auth:
  jwt:
    secret: "test-secret-key-for-jwt-min-32-chars!"
    access_token_ttl: 15m
    refresh_token_ttl: 168h
    issuer: "flecto-manager-test"
    header_name: "Authorization"
  openid:
    enabled: false`
	_ = afero.WriteFile(fs, fmt.Sprintf("%s/%s.yml", path, ConfigName), []byte(globalStr), 0644)
	viper.Reset()
	viper.SetFs(fs)
	cmd.SetArgs([]string{
		"--" + LogLevel, "WRONG"},
	)
	_ = cmd.Execute()
	err := GetRootPreRunEFn(ctx, false)(cmd, []string{})
	assert.Error(t, err)
	assert.Equal(t, "LevelVar(INFO)", ctx.LogLevel.String())
}

func TestGetRootPreRunEFn_FailValidator(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	path := GetDefaultConfigPath()
	fs := afero.NewMemMapFs()
	_ = fs.Mkdir(path, 0775)
	globalStr := "http: {listen: ''}"
	_ = afero.WriteFile(fs, fmt.Sprintf("%s/%s.yml", path, ConfigName), []byte(globalStr+"\n"), 0644)
	viper.Reset()
	viper.SetFs(fs)
	err := GetRootPreRunEFn(ctx, true)(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "configuration file is not valid")
}
