package cli

import (
	stdContext "context"
	"fmt"
	buildinHttp "net/http"

	"github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/http"
	"github.com/flectolab/flecto-manager/metrics"
	"github.com/flectolab/flecto-manager/scheduler"
	"github.com/flectolab/flecto-manager/scheduler/task"
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
		e, services, err := http.CreateServerHTTP(ctx)
		if err != nil {
			return err
		}

		// Background work: one goroutine per task, independent of the HTTP server
		sched := scheduler.New(ctx)
		sched.Register(task.NewActivityPurge(ctx, services.Activity))
		sched.Start()

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
				}
			}
		}()

		ctx.Logger.Info(fmt.Sprintf("starting server on %s", httpConfig.Listen))
		errStart := e.Start(httpConfig.Listen)
		if errStart != nil && errStart != buildinHttp.ErrServerClosed {
			panic(errStart)
		}

		// e.Start returns once the server is shutting down. Waiting here rather than
		// in the signal goroutine is what makes it effective: the process exits
		// through this path, so a purge in flight is not cut mid-batch.
		sched.Stop()
		ctx.Logger.Info("graceful shutdown completed")

		return nil
	}
}
