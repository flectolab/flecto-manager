package cli

import (
	"testing"

	"github.com/flectolab/flecto-manager/context"
	"github.com/stretchr/testify/assert"
)

func TestGetDBCmd(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetDBCmd(ctx)

	assert.Equal(t, "db", cmd.Use)
	assert.Equal(t, "db sub commands", cmd.Short)
}

func TestGetDBCmd_HasSubcommands(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetDBCmd(ctx)

	subcommands := cmd.Commands()
	assert.Len(t, subcommands, 3)

	// verify subcommand names
	names := make([]string, len(subcommands))
	for i, sub := range subcommands {
		names[i] = sub.Use
	}
	assert.Contains(t, names, "init")
	assert.Contains(t, names, "demo")
	assert.Contains(t, names, "migrate")
}

func TestGetDBCmd_InitSubcommand(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetDBCmd(ctx)

	initCmd, _, err := cmd.Find([]string{"init"})
	assert.NoError(t, err)
	assert.NotNil(t, initCmd)
	assert.Equal(t, "init", initCmd.Use)
}

func TestGetDBCmd_DemoSubcommand(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetDBCmd(ctx)

	demoCmd, _, err := cmd.Find([]string{"demo"})
	assert.NoError(t, err)
	assert.NotNil(t, demoCmd)
	assert.Equal(t, "demo", demoCmd.Use)
}

func TestGetDBCmd_MigrateSubcommand(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetDBCmd(ctx)

	migrateCmd, _, err := cmd.Find([]string{"migrate"})
	assert.NoError(t, err)
	assert.NotNil(t, migrateCmd)
	assert.Equal(t, "migrate", migrateCmd.Use)
}
