package main

import (
	"github.com/flectolab/flecto-manager/cli"
	"github.com/flectolab/flecto-manager/context"
)

func main() {
	ctx := context.DefaultContext()
	rootCmd := cli.GetRootCmd(ctx)

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}

}
