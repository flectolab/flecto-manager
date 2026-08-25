package security

import (
	"bytes"
	"strings"
	"testing"

	"github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/crypt"
	"github.com/stretchr/testify/assert"
)

func TestGetGenerateKeyCmd(t *testing.T) {
	cmd := GetGenerateKeyCmd(context.TestContext(nil))

	assert.Equal(t, CmdGenerateKeyName, cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestGetGenerateKeyRunFn(t *testing.T) {
	t.Run("successPrintsUsableKey", func(t *testing.T) {
		out := &bytes.Buffer{}
		cmd := GetGenerateKeyCmd(context.TestContext(nil))
		cmd.SetOut(out)

		err := cmd.Execute()

		assert.NoError(t, err)
		printed := strings.TrimSpace(out.String())

		key, errParse := crypt.ParseKey(printed)
		assert.NoError(t, errParse)
		assert.Len(t, key, crypt.KeySize)
	})

	t.Run("successPrintsDistinctKeys", func(t *testing.T) {
		first := &bytes.Buffer{}
		cmdFirst := GetGenerateKeyCmd(context.TestContext(nil))
		cmdFirst.SetOut(first)
		assert.NoError(t, cmdFirst.Execute())

		second := &bytes.Buffer{}
		cmdSecond := GetGenerateKeyCmd(context.TestContext(nil))
		cmdSecond.SetOut(second)
		assert.NoError(t, cmdSecond.Execute())

		assert.NotEqual(t, first.String(), second.String())
	})
}
