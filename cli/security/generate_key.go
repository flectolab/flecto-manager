// Package security holds the commands managing the key material of an instance.
package security

import (
	"github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/crypt"
	"github.com/spf13/cobra"
)

const CmdGenerateKeyName = "generate-key"

func GetGenerateKeyCmd(ctx *context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   CmdGenerateKeyName,
		Short: "generate an encryption key for security.encryption_key",
		Long: `Generate a random 32 byte key and print it, base64 encoded.

Store it in the configuration under security.encryption_key. The key protects the
secrets kept in the database: losing it makes them unreadable, and changing it
makes the values written with the previous one unreadable too.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		// The whole point of this command is to run before a configuration exists,
		// so it skips the loading and validation the other commands go through.
		PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
		RunE:              GetGenerateKeyRunFn(ctx),
	}

	return cmd
}

func GetGenerateKeyRunFn(_ *context.Context) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		key, err := crypt.GenerateKey()
		if err != nil {
			return err
		}

		// Printed bare, with nothing around it, so it can be piped straight into a
		// vault or a secret manager.
		cmd.Println(crypt.EncodeKey(key))
		return nil
	}
}
