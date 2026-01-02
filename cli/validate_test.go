package cli

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/flectolab/flecto-manager/context"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func Test_GetValidateRun_Success(t *testing.T) {
	buffer := bytes.NewBufferString("")
	ctx := context.TestContext(buffer)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	path := "/app"
	fs := afero.NewMemMapFs()
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
	_ = afero.WriteFile(fs, fmt.Sprintf("%s/config.yml", path), []byte(globalStr+"\n"), 0644)
	viper.Reset()
	viper.SetFs(fs)

	cmd.SetArgs([]string{
		CmdValidateName,
		"--" + ConfigName, fmt.Sprintf("%s/config.yml", path),
	},
	)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buffer.String(), "configuration file is valid")
}

func Test_GetValidateRun_Fail(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetRootCmd(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	path := "/app"
	fs := afero.NewMemMapFs()
	_ = fs.Mkdir(path, 0775)
	globalStr := "http: {listen: ''}"
	_ = afero.WriteFile(fs, fmt.Sprintf("%s/config.yml", path), []byte(globalStr+"\n"), 0644)
	viper.Reset()
	viper.SetFs(fs)

	cmd.SetArgs([]string{
		CmdValidateName,
		"--" + ConfigName, fmt.Sprintf("%s/config.yml", path),
	},
	)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "configuration file is not valid")
}
