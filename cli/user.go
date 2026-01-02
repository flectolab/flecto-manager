package cli

import (
	"github.com/flectolab/flecto-manager/cli/user"
	"github.com/flectolab/flecto-manager/context"
	"github.com/spf13/cobra"
)

func GetUserCmd(ctx *context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "user sub commands",
	}
	cmd.AddCommand(user.GetChangePasswordCmd(ctx))

	return cmd
}
