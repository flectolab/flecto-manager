package context

import (
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/flectolab/flecto-manager/config"
	flectoValidator "github.com/flectolab/flecto-manager/validator"
	"github.com/go-playground/validator/v10"
)

type Context struct {
	Logger   *slog.Logger
	LogLevel *slog.LevelVar

	sigs chan os.Signal
	done chan bool

	Config    *config.Config
	Validator *validator.Validate
}

func (c *Context) GetLogger() *slog.Logger {
	return c.Logger
}

func (c *Context) GetLogLevel() *slog.LevelVar {
	return c.LogLevel
}

func (c *Context) Cancel() {
	close(c.done)
}

func (c *Context) Done() <-chan bool {
	return c.done
}

func (c *Context) Signal() chan os.Signal {
	return c.sigs
}

func DefaultContext() *Context {
	level := &slog.LevelVar{}
	level.Set(slog.LevelInfo)
	opts := &slog.HandlerOptions{AddSource: false, Level: level}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	return &Context{
		Logger:    slog.New(slog.NewTextHandler(os.Stdout, opts)),
		LogLevel:  level,
		done:      make(chan bool),
		sigs:      sigs,
		Config:    config.DefaultConfig(),
		Validator: flectoValidator.New(),
	}
}

func TestContext(logBuffer io.Writer) *Context {
	if logBuffer == nil {
		logBuffer = io.Discard
	}
	level := &slog.LevelVar{}
	level.Set(slog.LevelInfo)
	opts := &slog.HandlerOptions{AddSource: false, Level: level}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	return &Context{
		Logger:    slog.New(slog.NewTextHandler(logBuffer, opts)),
		LogLevel:  level,
		done:      make(chan bool),
		sigs:      sigs,
		Config:    config.DefaultConfig(),
		Validator: flectoValidator.New(),
	}
}
