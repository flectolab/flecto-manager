---
sidebar_position: 5
---

# Monitoring

Flecto Manager exposes Prometheus metrics covering HTTP traffic, agent health and
background tasks.

Metrics are disabled by default. See [Configuration](./configuration.md#metrics) to
enable them and to choose between the main server and a dedicated port.

## Scraping

```yaml title="prometheus.yml"
scrape_configs:
  - job_name: 'flecto-manager'
    static_configs:
      - targets: ['localhost:8080']  # or the address set in metrics.listen
```

## Available metrics

### HTTP

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `flecto_http_requests_total` | Counter | `method`, `path`, `status` | Total number of HTTP requests |
| `flecto_http_request_duration_seconds` | Histogram | `method`, `path` | HTTP request duration in seconds |

### Agents

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `flecto_agent_online_total` | Gauge | `namespace`, `project` | Number of online agents |
| `flecto_agent_errors_total` | Gauge | `namespace`, `project` | Number of agents in error status (excluding offline agents) |

### Background tasks

Flecto Manager runs recurring background work, such as the
[activity journal](./features/activity.md) purge. Every task reports the same three
metrics, identified by a `task` label.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `flecto_scheduler_task_runs_total` | Counter | `task`, `status` | Task runs by outcome, `success` or `error` |
| `flecto_scheduler_task_duration_seconds` | Histogram | `task` | Task run duration in seconds |
| `flecto_scheduler_task_last_success_timestamp_seconds` | Gauge | `task` | Unix timestamp of the last successful run |

A task that panics is counted as an `error`, so a task failing on every run never
reads as healthy.

## Alerting

The examples below are starting points. Adjust every threshold and window to your own
traffic and configuration — the values shown are not defaults, and a rule copied
unchanged will either be noisy or silent.

### Agents

```yaml
# No agent is serving a project any more.
- alert: FlectoNoAgentOnline
  expr: flecto_agent_online_total == 0
  for: 5m

# Agents are reporting sync errors.
- alert: FlectoAgentErrors
  expr: flecto_agent_errors_total > 0
  for: 5m
```

Both work on gauges, which return to zero once the situation clears, so the alert
resolves by itself.

### HTTP

```yaml
# Server-side error ratio above 1% of traffic.
- alert: FlectoHighErrorRate
  expr: |
    sum(rate(flecto_http_requests_total{status=~"5.."}[5m]))
      / sum(rate(flecto_http_requests_total[5m])) > 0.01
  for: 10m

# 95th percentile latency degraded.
- alert: FlectoSlowRequests
  expr: |
    histogram_quantile(0.95,
      sum(rate(flecto_http_request_duration_seconds_bucket[5m])) by (le)
    ) > 1
  for: 10m
```

### Background tasks

:::warning The window must be longer than the task interval
A background task only runs once per interval. With an hourly task, a rule looking at
a 5 minute window reads zero even while the task is failing every single run, because
no run happened inside the window. Size the window to a few intervals.
:::

```yaml
# A task has failed at least once recently.
# Replace 2h with a few times the task interval (for example activity.purge_interval).
- alert: FlectoBackgroundTaskFailing
  expr: increase(flecto_scheduler_task_runs_total{status="error"}[2h]) > 0
  for: 5m
```

The rule above only catches a task that *runs and fails*. A task that stops running
altogether emits no error at all — it simply stops emitting — so it needs its own
rule based on how stale its last success is:

```yaml
# A task has not succeeded for far longer than its interval.
# Threshold: a small multiple of the interval, here 2h for an hourly task.
- alert: FlectoBackgroundTaskStale
  expr: time() - flecto_scheduler_task_last_success_timestamp_seconds > 7200
  for: 15m
```

This second rule is the one that catches a misconfigured interval, a task stuck in a
run that never returns, or an instance that stopped scheduling. Of the two, it is the
one worth having if you only keep one.

```yaml
# A task is drifting towards its timeout: runs are taking far longer than they used to.
- alert: FlectoBackgroundTaskSlow
  expr: |
    histogram_quantile(0.95,
      sum(rate(flecto_scheduler_task_duration_seconds_bucket[1h])) by (le, task)
    ) > 300
  for: 30m
```

## Dashboards

Useful panels to build from these metrics:

- Request rate and latency percentiles, broken down by path
- Online and erroring agents per namespace and project
- Background task success rate, `rate(flecto_scheduler_task_runs_total[1h])` by `task` and `status`
- Time since each task last succeeded, `time() - flecto_scheduler_task_last_success_timestamp_seconds`
