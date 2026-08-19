package scheduler

import (
	"bytes"
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// fakeTask is a Task whose behaviour each test tunes.
type fakeTask struct {
	name       string
	interval   time.Duration
	runs       atomic.Int32
	err        error
	panicOnRun bool
	runOnStart bool
	timeout    time.Duration
	// block makes Run wait until released, to observe a run in flight.
	block   chan struct{}
	sawCtx  atomic.Bool
	ctxDone atomic.Bool
	// waitForCtx makes Run block until its context ends, to observe the timeout.
	waitForCtx bool
	// runFor makes Run last, to observe what happens to ticks meanwhile.
	runFor        time.Duration
	concurrent    atomic.Int32
	maxConcurrent atomic.Int32
}

func (t *fakeTask) Name() string            { return t.name }
func (t *fakeTask) Interval() time.Duration { return t.interval }
func (t *fakeTask) RunOnStart() bool        { return t.runOnStart }
func (t *fakeTask) Timeout() time.Duration  { return t.timeout }

func (t *fakeTask) Run(ctx context.Context) error {
	t.runs.Add(1)
	t.sawCtx.Store(ctx != nil)

	inFlight := t.concurrent.Add(1)
	for {
		seen := t.maxConcurrent.Load()
		if inFlight <= seen || t.maxConcurrent.CompareAndSwap(seen, inFlight) {
			break
		}
	}
	defer t.concurrent.Add(-1)

	if t.runFor > 0 {
		time.Sleep(t.runFor)
	}

	if t.panicOnRun {
		panic("boom")
	}
	if t.block != nil {
		<-t.block
	}
	if t.waitForCtx {
		// Long enough that the configured timeout must fire first
		select {
		case <-ctx.Done():
			t.ctxDone.Store(true)
		case <-time.After(5 * time.Second):
		}
	}
	return t.err
}

func eventually(t *testing.T, msg string, condition func() bool) {
	t.Helper()
	assert.Eventually(t, condition, 2*time.Second, 5*time.Millisecond, msg)
}

func TestNew(t *testing.T) {
	s := New(appContext.TestContext(nil))

	assert.NotNil(t, s)
	assert.Empty(t, s.tasks)
}

func TestSchedulerRegister(t *testing.T) {
	s := New(appContext.TestContext(nil))

	s.Register(&fakeTask{name: "a"}, &fakeTask{name: "b"})
	s.Register(&fakeTask{name: "c"})

	assert.Len(t, s.tasks, 3)
}

func TestSchedulerRunsOnEveryTick(t *testing.T) {
	appCtx := appContext.TestContext(nil)
	task := &fakeTask{name: "ticking", interval: 10 * time.Millisecond}

	s := New(appCtx)
	s.Register(task)
	s.Start()
	defer s.Stop()

	eventually(t, "the task should run repeatedly", func() bool { return task.runs.Load() >= 3 })
	assert.True(t, task.sawCtx.Load())
}

func TestSchedulerSkipsDisabledTask(t *testing.T) {
	appCtx := appContext.TestContext(nil)
	// A zero interval is how a feature turns its own background work off
	disabled := &fakeTask{name: "disabled", interval: 0}
	enabled := &fakeTask{name: "enabled", interval: 10 * time.Millisecond}

	s := New(appCtx)
	s.Register(disabled, enabled)
	s.Start()
	defer s.Stop()

	eventually(t, "the enabled task should run", func() bool { return enabled.runs.Load() >= 1 })
	assert.Zero(t, disabled.runs.Load(), "a task with a zero interval must never run")
}

func TestSchedulerRunOnStart(t *testing.T) {
	appCtx := appContext.TestContext(nil)
	// A long interval: only the boot run can happen within the test
	task := &fakeTask{name: "boot", interval: time.Hour, runOnStart: true}

	s := New(appCtx)
	s.Register(task)
	s.Start()
	defer s.Stop()

	eventually(t, "the task should run once at boot", func() bool { return task.runs.Load() == 1 })
}

func TestSchedulerWithoutRunOnStartWaitsForTick(t *testing.T) {
	appCtx := appContext.TestContext(nil)
	task := &fakeTask{name: "no-boot", interval: time.Hour, runOnStart: false}

	s := New(appCtx)
	s.Register(task)
	s.Start()
	defer s.Stop()

	time.Sleep(50 * time.Millisecond)
	assert.Zero(t, task.runs.Load(), "without RunOnStart the first run waits for the tick")
}

func TestSchedulerContainsPanic(t *testing.T) {
	appCtx := appContext.TestContext(nil)
	// A panicking task must not take the process down, and must keep being scheduled
	task := &fakeTask{name: "panicking", interval: 10 * time.Millisecond, panicOnRun: true}

	s := New(appCtx)
	s.Register(task)
	s.Start()
	defer s.Stop()

	eventually(t, "a panicking task should keep running on later ticks", func() bool {
		return task.runs.Load() >= 2
	})
}

func TestSchedulerKeepsRunningAfterError(t *testing.T) {
	appCtx := appContext.TestContext(nil)
	task := &fakeTask{name: "failing", interval: 10 * time.Millisecond, err: errors.New("nope")}

	s := New(appCtx)
	s.Register(task)
	s.Start()
	defer s.Stop()

	eventually(t, "a failing task should be retried on the next tick", func() bool {
		return task.runs.Load() >= 2
	})
}

func TestSchedulerAppliesTimeout(t *testing.T) {
	appCtx := appContext.TestContext(nil)
	task := &fakeTask{name: "slow", interval: time.Hour, runOnStart: true, timeout: 20 * time.Millisecond, waitForCtx: true}

	s := New(appCtx)
	s.Register(task)
	s.Start()
	defer s.Stop()

	eventually(t, "the run context should be cancelled by the timeout", func() bool {
		return task.ctxDone.Load()
	})
}

func TestSchedulerStopsOnAppShutdown(t *testing.T) {
	appCtx := appContext.TestContext(nil)
	task := &fakeTask{name: "stopping", interval: 10 * time.Millisecond}

	s := New(appCtx)
	s.Register(task)
	s.Start()

	eventually(t, "the task should be running", func() bool { return task.runs.Load() >= 1 })

	// Cancelling the application context is what shuts the tasks down
	appCtx.Cancel()
	s.Wait()

	runsAtStop := task.runs.Load()
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, runsAtStop, task.runs.Load(), "no run should start after shutdown")
}

func TestSchedulerWaitsForRunInFlight(t *testing.T) {
	appCtx := appContext.TestContext(nil)
	release := make(chan struct{})
	task := &fakeTask{name: "in-flight", interval: time.Hour, runOnStart: true, block: release}

	s := New(appCtx)
	s.Register(task)
	s.Start()

	eventually(t, "the run should have started", func() bool { return task.runs.Load() == 1 })

	waited := make(chan struct{})
	go func() {
		s.Stop()
		close(waited)
	}()

	// Stop must not return while the run is still in progress: a purge cut mid-batch
	// is what this guards against.
	select {
	case <-waited:
		t.Fatal("Stop returned while a run was still in flight")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	select {
	case <-waited:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not return after the run finished")
	}
}

func TestSchedulerRunsTasksIndependently(t *testing.T) {
	appCtx := appContext.TestContext(nil)
	release := make(chan struct{})
	// A task stuck in its run must not hold the other one back
	slow := &fakeTask{name: "slow", interval: 10 * time.Millisecond, block: release}
	fast := &fakeTask{name: "fast", interval: 10 * time.Millisecond}

	s := New(appCtx)
	s.Register(slow, fast)
	s.Start()

	eventually(t, "the fast task should keep ticking while the slow one blocks", func() bool {
		return fast.runs.Load() >= 3
	})
	assert.Equal(t, int32(1), slow.runs.Load(), "the blocked task should not have started a second run")

	close(release)
	s.Stop()
}

func TestSchedulerNeverOverlapsRunsOfSameTask(t *testing.T) {
	appCtx := appContext.TestContext(nil)
	// Runs six times longer than the interval: every tick but one lands while a run
	// is already in progress.
	task := &fakeTask{name: "slower-than-interval", interval: 10 * time.Millisecond, runFor: 60 * time.Millisecond}

	s := New(appCtx)
	s.Register(task)
	s.Start()

	time.Sleep(300 * time.Millisecond)
	s.Stop()

	// The loop calls Run synchronously, so a task can never race against itself.
	// This is what makes a task free to touch shared state without locking.
	assert.Equal(t, int32(1), task.maxConcurrent.Load(), "two runs of the same task must never overlap")

	// time.Ticker drops ticks for a slow receiver instead of queueing them, so a
	// slow task cannot build a backlog: roughly 300ms/60ms runs, not 300ms/10ms.
	runs := task.runs.Load()
	assert.Less(t, runs, int32(15), "ticks must be dropped during a run, not queued")
	assert.GreaterOrEqual(t, runs, int32(2), "the task should still run repeatedly")
}

func TestSchedulerWarnsWhenTimeoutNotShorterThanInterval(t *testing.T) {
	logs := &bytes.Buffer{}
	appCtx := appContext.TestContext(logs)
	// Timeout plus long que l'intervalle : configuration a signaler
	task := &fakeTask{name: "greedy", interval: 20 * time.Millisecond, timeout: time.Minute}

	s := New(appCtx)
	s.Register(task)
	s.Start()
	s.Stop()

	assert.Contains(t, logs.String(), "timeout is not shorter than its interval")
	assert.Contains(t, logs.String(), "task=greedy")
}

func TestSchedulerDoesNotWarnWhenTimeoutIsShorter(t *testing.T) {
	logs := &bytes.Buffer{}
	appCtx := appContext.TestContext(logs)
	task := &fakeTask{name: "sane", interval: time.Minute, timeout: 10 * time.Second}

	s := New(appCtx)
	s.Register(task)
	s.Start()
	s.Stop()

	assert.NotContains(t, logs.String(), "timeout is not shorter")
}

// taskMetric reads one scheduler metric back out of the default registry.
func taskMetric(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()

	families, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)

	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			matches := true
			for _, pair := range metric.GetLabel() {
				if want, ok := labels[pair.GetName()]; ok && want != pair.GetValue() {
					matches = false
				}
			}
			if !matches {
				continue
			}
			if counter := metric.GetCounter(); counter != nil {
				return counter.GetValue()
			}
			if gauge := metric.GetGauge(); gauge != nil {
				return gauge.GetValue()
			}
			if histogram := metric.GetHistogram(); histogram != nil {
				return float64(histogram.GetSampleCount())
			}
		}
	}
	return -1
}

func TestSchedulerPublishesMetrics(t *testing.T) {
	t.Run("a task is published before its first run", func(t *testing.T) {
		appCtx := appContext.TestContext(nil)
		// A long interval and no run on start: nothing has run yet
		task := &fakeTask{name: "metrics-never-run", interval: time.Hour}

		s := New(appCtx)
		s.Register(task)
		s.Start()
		defer s.Stop()

		// An absent series would make an alerting rule match nothing, which looks
		// exactly like a healthy task
		assert.Equal(t, float64(0), taskMetric(t, "flecto_scheduler_task_runs_total",
			map[string]string{"task": "metrics-never-run", "status": "error"}))
		assert.Equal(t, float64(0), taskMetric(t, "flecto_scheduler_task_runs_total",
			map[string]string{"task": "metrics-never-run", "status": "success"}))
	})

	t.Run("a successful run counts and stamps the last success", func(t *testing.T) {
		appCtx := appContext.TestContext(nil)
		task := &fakeTask{name: "metrics-success", interval: time.Hour, runOnStart: true}

		s := New(appCtx)
		s.Register(task)
		s.Start()
		defer s.Stop()

		eventually(t, "the run should be counted", func() bool {
			return taskMetric(t, "flecto_scheduler_task_runs_total",
				map[string]string{"task": "metrics-success", "status": "success"}) == 1
		})
		assert.Zero(t, taskMetric(t, "flecto_scheduler_task_runs_total",
			map[string]string{"task": "metrics-success", "status": "error"}))
		assert.Positive(t, taskMetric(t, "flecto_scheduler_task_last_success_timestamp_seconds",
			map[string]string{"task": "metrics-success"}))
		assert.Equal(t, float64(1), taskMetric(t, "flecto_scheduler_task_duration_seconds",
			map[string]string{"task": "metrics-success"}))
	})

	t.Run("a failed run counts as an error and leaves the last success untouched", func(t *testing.T) {
		appCtx := appContext.TestContext(nil)
		task := &fakeTask{name: "metrics-failure", interval: time.Hour, runOnStart: true, err: errors.New("nope")}

		s := New(appCtx)
		s.Register(task)
		s.Start()
		defer s.Stop()

		eventually(t, "the failure should be counted", func() bool {
			return taskMetric(t, "flecto_scheduler_task_runs_total",
				map[string]string{"task": "metrics-failure", "status": "error"}) == 1
		})
		assert.Zero(t, taskMetric(t, "flecto_scheduler_task_runs_total",
			map[string]string{"task": "metrics-failure", "status": "success"}))
		// Never succeeded, so the timestamp series must not exist
		assert.Equal(t, float64(-1), taskMetric(t, "flecto_scheduler_task_last_success_timestamp_seconds",
			map[string]string{"task": "metrics-failure"}))
	})

	t.Run("a panic counts as an error", func(t *testing.T) {
		appCtx := appContext.TestContext(nil)
		// A task crashing on every run must not read as healthy
		task := &fakeTask{name: "metrics-panic", interval: time.Hour, runOnStart: true, panicOnRun: true}

		s := New(appCtx)
		s.Register(task)
		s.Start()
		defer s.Stop()

		eventually(t, "the panic should be counted as an error", func() bool {
			return taskMetric(t, "flecto_scheduler_task_runs_total",
				map[string]string{"task": "metrics-panic", "status": "error"}) == 1
		})
	})
}
