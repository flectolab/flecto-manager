package cli

import (
	"github.com/flectolab/flecto-manager/context"
	"github.com/spf13/cobra"
)

const CmdValidateName = "validate"

func GetValidateCmd(ctx *context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:               CmdValidateName,
		Short:             "validate config",
		SilenceUsage:      true,
		SilenceErrors:     true,
		PersistentPreRunE: GetRootPreRunEFn(ctx, false),
		RunE:              GetValidateRunFn(ctx),
	}

	return cmd
}

func GetValidateRunFn(ctx *context.Context) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := validateConfig(ctx)
		if err != nil {
			return err
		}
		ctx.Logger.Info("configuration file is valid")

		return nil
	}
}
