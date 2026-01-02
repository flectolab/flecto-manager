package cli

import (
	"testing"

	"github.com/flectolab/flecto-manager/context"
	"github.com/stretchr/testify/assert"
)

func TestGetUserCmd(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetUserCmd(ctx)

	assert.Equal(t, "user", cmd.Use)
	assert.Equal(t, "user sub commands", cmd.Short)
}

func TestGetUserCmd_HasSubcommands(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetUserCmd(ctx)

	subcommands := cmd.Commands()
	assert.Len(t, subcommands, 1)

	// verify subcommand names
	names := make([]string, len(subcommands))
	for i, sub := range subcommands {
		names[i] = sub.Use
	}
	assert.Contains(t, names, "change-password")
}

func TestGetUserCmd_ChangePasswordSubcommand(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetUserCmd(ctx)

	changePasswordCmd, _, err := cmd.Find([]string{"change-password"})
	assert.NoError(t, err)
	assert.NotNil(t, changePasswordCmd)
	assert.Equal(t, "change-password", changePasswordCmd.Use)
}
