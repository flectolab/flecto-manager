package cli

import (
	"github.com/flectolab/flecto-manager/cli/security"
	"github.com/flectolab/flecto-manager/context"
	"github.com/spf13/cobra"
)

func GetSecurityCmd(ctx *context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "security",
		Short: "security sub commands",
	}
	cmd.AddCommand(security.GetGenerateKeyCmd(ctx))

	return cmd
}
