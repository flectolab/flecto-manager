package context

import (
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/flectolab/flecto-manager/config"
	"github.com/stretchr/testify/assert"
)

func TestDefaultContext_Success(t *testing.T) {

	level := &slog.LevelVar{}
	level.Set(slog.LevelInfo)
	opts := &slog.HandlerOptions{AddSource: false, Level: level}
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	want := &Context{
		Logger:   logger,
		LogLevel: level,
		Config:   config.DefaultConfig(),
	}
	got := DefaultContext()

	got.done = nil
	got.sigs = nil
	assert.NotNil(t, got.Validator)
	got.Validator = nil
	assert.Equal(t, want, got)
}

func TestTestContext(t *testing.T) {

	level := &slog.LevelVar{}
	level.Set(slog.LevelInfo)
	opts := &slog.HandlerOptions{AddSource: false, Level: level}
	logger := slog.New(slog.NewTextHandler(io.Discard, opts))
	want := &Context{
		Logger:   logger,
		LogLevel: level,
		Config:   config.DefaultConfig(),
	}
	got := TestContext(nil)

	got.done = nil
	got.sigs = nil
	assert.NotNil(t, got.Validator)
	got.Validator = nil
	assert.Equal(t, want, got)
}

func TestTestContext_WithLogBuffer(t *testing.T) {

	level := &slog.LevelVar{}
	level.Set(slog.LevelInfo)
	opts := &slog.HandlerOptions{AddSource: false, Level: level}
	logger := slog.New(slog.NewTextHandler(io.Discard, opts))
	want := &Context{
		Logger:   logger,
		LogLevel: level,
		Config:   config.DefaultConfig(),
	}
	got := TestContext(io.Discard)
	got.done = nil
	got.sigs = nil
	assert.NotNil(t, got.Validator)
	got.Validator = nil
	assert.Equal(t, want, got)
}

func TestContext_Cancel(t *testing.T) {
	ctx := &Context{}
	ctx.done = make(chan bool)
	running := true
	go func() {
		select {
		case <-ctx.done:
			running = false
			return
		}
	}()
	ctx.Cancel()
	time.Sleep(time.Millisecond * 100)
	assert.Equal(t, false, running)
}

func TestContext_Done(t *testing.T) {
	ctx := &Context{}
	ctx.done = make(chan bool)
	running := true
	go func() {
		select {
		case <-ctx.Done():
			running = false
			return
		}
	}()
	close(ctx.done)
	time.Sleep(time.Millisecond * 100)
	assert.Equal(t, false, running)
}

func TestContext_Signal(t *testing.T) {
	ctx := &Context{}
	ctx.sigs = make(chan os.Signal, 1)
	signal.Notify(ctx.sigs, syscall.SIGINT, syscall.SIGTERM)
	running := true
	go func() {
		select {
		case <-ctx.Signal():
			running = false
		}
	}()
	ctx.Signal() <- syscall.SIGINT
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, false, running)
}

func TestContext_GetLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{AddSource: false, Level: &slog.LevelVar{}}))
	c := &Context{
		Logger: logger,
	}
	assert.Equalf(t, logger, c.GetLogger(), "GetLogger()")
}

func TestContext_GetLogLevel(t *testing.T) {
	logLevel := &slog.LevelVar{}
	logLevel.Set(slog.LevelInfo)
	c := &Context{
		LogLevel: logLevel,
	}
	assert.Equalf(t, logLevel, c.GetLogLevel(), "GetLogLevel()")
}
