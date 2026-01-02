package cli

import (
	"os"

	"github.com/flectolab/flecto-manager/version"
	"github.com/spf13/cobra"
)

func GetVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version info",
		Run:   GetVersionRunFn(),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	cmd.SetOut(os.Stdout)
	return cmd
}

func GetVersionRunFn() func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		cmd.Println(version.GetFormattedVersion())
	}
}
