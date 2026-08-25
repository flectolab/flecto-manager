package cli

import (
	"testing"

	"github.com/flectolab/flecto-manager/context"
	"github.com/stretchr/testify/assert"
)

func TestGetSecurityCmd(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetSecurityCmd(ctx)

	assert.Equal(t, "security", cmd.Use)
	assert.Equal(t, "security sub commands", cmd.Short)
}

func TestGetSecurityCmd_HasSubcommands(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetSecurityCmd(ctx)

	subcommands := cmd.Commands()
	assert.Len(t, subcommands, 1)

	names := make([]string, len(subcommands))
	for i, sub := range subcommands {
		names[i] = sub.Use
	}
	assert.Contains(t, names, "generate-key")
}

func TestGetSecurityCmd_GenerateKeySubcommand(t *testing.T) {
	ctx := context.TestContext(nil)
	cmd := GetSecurityCmd(ctx)

	generateKeyCmd, _, err := cmd.Find([]string{"generate-key"})
	assert.NoError(t, err)
	assert.NotNil(t, generateKeyCmd)
	assert.Equal(t, "generate-key", generateKeyCmd.Use)
}
