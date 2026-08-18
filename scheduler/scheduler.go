// Package scheduler runs recurring background work.
//
// It owns only the plumbing every background job needs — its own goroutine and
// ticker, panic recovery, an optional per-run timeout, uniform logging and a clean
// shutdown — so a task itself is just the work to do.
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/metrics"
)

// Task is a unit of recurring background work.
//
// Scheduling policy and work sit in the same interface on purpose: the zero value
// of each knob already means "no option" — a zero interval disables the task, a
// zero timeout leaves the run unbounded, RunOnStart false waits for the first tick
// — so splitting them into optional interfaces would only have bought type
// assertions in the runner and awkward composition at every call site.
type Task interface {
	// Name identifies the task in logs.
	Name() string
	// Interval is the delay between two runs. Zero disables the task.
	Interval() time.Duration
	// RunOnStart reports whether the first run happens at boot instead of waiting
	// for the first tick.
	RunOnStart() bool
	// Timeout bounds a single run. Zero leaves it unbounded.
	Timeout() time.Duration
	// Run does the work. It must be idempotent: a task can run again after a failed
	// or interrupted attempt.
	Run(ctx context.Context) error
}

type Scheduler struct {
	appCtx *appContext.Context
	tasks  []Task

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New(appCtx *appContext.Context) *Scheduler {
	return &Scheduler{appCtx: appCtx}
}

// Register adds tasks to run. It must be called before Start.
func (s *Scheduler) Register(tasks ...Task) {
	s.tasks = append(s.tasks, tasks...)
}

// Start launches one goroutine per enabled task. A task with a zero interval is
// skipped, which is how a feature disables its own background work through config.
func (s *Scheduler) Start() {
	// The application context exposes a done channel rather than a context.Context,
	// so the bridge to one lives here instead of changing that contract.
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	go func() {
		<-s.appCtx.Done()
		cancel()
	}()

	for _, task := range s.tasks {
		interval := task.Interval()
		if interval <= 0 {
			s.appCtx.Logger.Info("scheduler task disabled", "task", task.Name())
			continue
		}

		// Two runs of the same task never overlap, and ticks that fire during a run
		// are dropped rather than queued, so a slow task cannot pile up. What it can
		// do is occupy its slot back to back with no pause, which a timeout at least
		// as long as the interval makes likely.
		if timeout := task.Timeout(); timeout > 0 && timeout >= interval {
			s.appCtx.Logger.Warn("scheduler task timeout is not shorter than its interval, runs may follow each other with no pause",
				"task", task.Name(), "interval", interval, "timeout", timeout)
		}

		metrics.InitSchedulerTask(task.Name())

		s.wg.Add(1)
		go s.loop(ctx, task, interval)
		s.appCtx.Logger.Info("scheduler task started", "task", task.Name(), "interval", interval)
	}
}

// Stop cancels the running tasks and waits for the current runs to finish.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

// Wait blocks until every task goroutine has returned.
func (s *Scheduler) Wait() {
	s.wg.Wait()
}

// loop drives one task. One goroutine and one ticker per task means a slow task
// never delays another, and two runs of the same task never overlap.
func (s *Scheduler) loop(ctx context.Context, task Task, interval time.Duration) {
	defer s.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	if task.RunOnStart() {
		s.run(ctx, task)
	}

	for {
		select {
		case <-ctx.Done():
			s.appCtx.Logger.Info("scheduler task stopped", "task", task.Name())
			return
		case <-ticker.C:
			s.run(ctx, task)
		}
	}
}

// run executes one attempt, and is the reason a task can stay this simple: a panic
// is contained here rather than taking the process down, and every task is logged
// and measured the same way.
func (s *Scheduler) run(ctx context.Context, task Task) {
	start := time.Now()

	// err is set by the run below and read back by the deferred reporter, which also
	// turns a panic into a failure: a task crashing on every run must not read as
	// healthy in the metrics.
	var err error
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}

		duration := time.Since(start)
		metrics.RecordSchedulerRun(task.Name(), duration, err)

		if err != nil {
			s.appCtx.Logger.Error("scheduler task failed", "task", task.Name(), "duration", duration, "error", err)
			return
		}
		s.appCtx.Logger.Debug("scheduler task completed", "task", task.Name(), "duration", duration)
	}()

	runCtx := ctx
	if timeout := task.Timeout(); timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	err = task.Run(runCtx)
}
