package cli

import (
	"github.com/flectolab/flecto-manager/cli/db"
	"github.com/flectolab/flecto-manager/context"
	"github.com/spf13/cobra"
)

func GetDBCmd(ctx *context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "db sub commands",
	}
	cmd.AddCommand(db.GetInitCmd(ctx))
	cmd.AddCommand(db.GetMigrateCmd(ctx))

	return cmd
}
