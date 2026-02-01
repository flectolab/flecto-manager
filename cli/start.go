package cli

import (
	stdContext "context"
	"fmt"
	buildinHttp "net/http"

	"github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/http"
	"github.com/flectolab/flecto-manager/metrics"
	"github.com/spf13/cobra"
)

func GetStartCmd(ctx *context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "start server",
		RunE:  GetStartRunFn(ctx),
	}
}

func GetStartRunFn(ctx *context.Context) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		e, err := http.CreateServerHTTP(ctx)
		if err != nil {
			return err
		}

		// Start separate metrics server if configured
		var metricsServer *buildinHttp.Server
		if ctx.Config.Metrics.Enabled && ctx.Config.Metrics.Listen != "" {
			metricsServer = metrics.StartServer(ctx, ctx.Config.Metrics.Listen)
		}

		httpConfig := ctx.Config.HTTP
		go func() {
			for {
				select {
				case sig := <-ctx.Signal():
					ctx.Logger.Info(fmt.Sprintf("%s signal received, exiting...", sig.String()))
					ctx.Cancel()
					if metricsServer != nil {
						_ = metricsServer.Shutdown(stdContext.Background())
					}
					_ = e.Shutdown(stdContext.Background())
					ctx.Logger.Info("graceful shutdown completed")
				}
			}
		}()

		ctx.Logger.Info(fmt.Sprintf("starting server on %s", httpConfig.Listen))
		errStart := e.Start(httpConfig.Listen)
		if errStart != nil && errStart != buildinHttp.ErrServerClosed {
			panic(errStart)
		}

		return nil
	}
}
