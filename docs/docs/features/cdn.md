---
sidebar_position: 5
---

# CDN Cache Purge

When a project is served behind a CDN, publishing a change is not enough: the CDN
keeps serving the previous redirect or page until its cache expires. Flecto Manager
can invalidate that cache for you, on every publish, for exactly what the publish
changed.

## How it works

1. You declare one or more **CDN configurations** on a project (admin section,
   project edit page, **CDN** tab).
2. On every publish, Flecto Manager works out the URLs affected by the change and
   queues one **purge job** per enabled configuration. The job is created in the same
   transaction as the publish: a publish that fails queues nothing, and a publish
   that succeeds never loses its purge.
3. A background task sends the jobs to the CDN. It first waits for the project's
   agents to serve the published version (see [below](#waiting-for-the-agents)), then
   purges in batches, within the provider's rate limits.
4. Each finished job is shown in the purge history of the project and recorded in the
   [activity journal](./activity.md).

Both the old and the new address are purged: modifying the source of a redirect
from `/promo` to `/sale` purges `/promo` (which must stop redirecting) and `/sale`
(which must start).

## Prerequisites

CDN credentials are stored **encrypted** in the database. The instance needs an
encryption key, otherwise the CDN tab reports the feature as unavailable instead of
storing credentials in clear text.

```bash
flecto-manager security generate-key
```

```yaml title="manager.yaml"
security:
  encryption_key: "<the generated key>"
```

:::warning Keep the key safe
Losing the key, or changing it, makes every stored credential unreadable: each CDN
configuration then has to be given its credentials again.
:::

## Supported providers

### Cloudflare

Purges a Cloudflare zone **by URL** (single-file purge). Purging by prefix or by
hostname is never used, since it would drop cached objects nobody changed.

| Field | Description |
|-------|-------------|
| Zone ID | Found on the overview page of the domain in the Cloudflare dashboard |
| API token | A token with the **Zone › Cache Purge** permission on that zone. Stored encrypted and never displayed again |
| Hostnames | The hostnames this project serves through this zone (see [What gets purged](#what-gets-purged)) |
| Plan | The zone's plan, which sets the purge quota |

| Plan | URLs per request | URLs per second |
|------|------------------|-----------------|
| Free | 100 | 800 |
| Pro | 100 | 1,500 |
| Business | 100 | 1,500 |
| Enterprise | 500 | 3,000 |

Use the **Test** button after saving a configuration: it checks the credentials
against the Cloudflare API without purging anything.

## What gets purged

A CDN caches responses by URL, so a redirect or page can only be purged if it
resolves to concrete URLs.

| Entry | Purged |
|-------|--------|
| `BASIC` redirect or page (`/promo`) | One URL per hostname declared on the configuration |
| `BASIC_HOST` (`shop.example.com/promo`) | Only if that hostname is declared on the configuration |
| `REGEX`, `REGEX_HOST` | Never: a CDN cannot purge by pattern |

The query string is part of the purged URL: a CDN caches `/promo?utm=a` apart from
`/promo`. A path with encoded characters is purged in both of its forms
(`/caf%C3%A9` and `/café`), since which one a CDN keys its cache on depends on how
it normalises requests. A form that would not be a valid URL once decoded (an encoded
space, `%`, `?` or `#`) is only sent encoded.

A hostname entry starting with `*.` (for example `*.example.com`) matches any
subdomain, so it covers `BASIC_HOST` entries such as `fr.example.com/promo`. It does
not match the apex domain, and it cannot complete a `BASIC` entry, because
`*.example.com/promo` is not an address that can be purged.

What could not be purged is **counted per reason** and shown in the purge history and
the activity journal:

| Reason | Meaning |
|--------|---------|
| regex | A `REGEX` or `REGEX_HOST` entry |
| hostname not served | A `BASIC_HOST` entry whose hostname is not declared on this configuration |
| no hostname configured | A `BASIC` entry, but the configuration only declares `*.` hostnames |
| invalid | An address that could not be parsed |

## Waiting for the agents

A purge sent too early does nothing useful. Right after the purge, the CDN asks an
agent for the address again. If that agent has not synchronised yet, it answers with
the previous redirect, and the CDN caches it all over again.

A publish job therefore starts in the **Waiting for agents** status and is only sent
once every **online** agent of the project serves the published version. Agents that
have not been seen for `agent.offline_threshold` are ignored, and a project with no
online agent is purged straight away.

The wait is bounded by `cdn.agent_convergence_timeout` (30 minutes by default). Past
that deadline the job is sent anyway and flagged **Purged before the agents caught
up**: the cache may hold the previous version again, and a
[manual purge](#manual-purge) is worth running once the agents are back on track.

The purge history shows when a waiting job will be sent anyway. A job that sits in
this status for a while is not stuck: it is waiting for an agent, and the line under
its status says until when.

Publishing again while a job is still waiting folds the new changes into it rather
than queueing a second job. The wait then starts over from the new version. The new
publish gets a job of its own instead when:

- folding it in would take the job over `cdn.max_targets_per_job`: the waiting job
  keeps the earlier publish, which fit, and only the new one is weighed against the
  cap;
- the background task started sending the waiting job in the meantime.

## Purge history

The **CDN** tab of the project edit page lists the latest jobs, newest first, and
refreshes on its own.

| Status | Meaning |
|--------|---------|
| Waiting for agents | Queued, held back until the agents serve the published version |
| Running | Being sent. The `purged / requested` count moves as batches go through |
| Succeeded | Every URL was purged |
| Partial | Some batches went through and others did not: the cache is in a mixed state |
| Failed | Nothing could be purged, see the error |
| Skipped | Never sent, see the reason below |

A job is **skipped** when:

- its configuration was disabled or deleted before the job was sent, or
- the publish produced more URLs than `cdn.max_targets_per_job` (5,000 by default).
  Purging that many URLs one at a time is rarely worth it, and the alternatives
  (purging a prefix or a whole hostname) would drop cached objects nobody changed.
  The decision is left to you: purge the URLs that matter by hand, or purge the whole
  zone from the CDN's own dashboard. Such a job is recorded at publish time, in the
  history, the activity journal and the metrics, without ever reaching the background
  task.

### Failures and retries

A temporary failure (rate limit, network error, CDN outage) is retried on the next
run, from the batch that failed: what was already purged is not sent again. After 5
failed attempts, the job gives up.

A permanent failure, such as a rejected token, stops the job straight away: the
remaining batches would fail the same way. The job ends **Failed**, or **Partial** if
some batches had already gone through.

Large jobs are spread over several runs of the background task. Progress is saved
after every batch, so a restart of the instance resends at most one batch.

## Manual purge

**Purge URLs manually**, under the purge history, sends a list of URLs through one
configuration, outside of any publish. It is the way to catch up after a job was
skipped, failed, or purged before the agents caught up.

- URLs are given in full, as they are served: `https://www.example.com/promo`.
- Only URLs whose hostname the configuration declares are sent. The others are
  reported as skipped.
- A manual purge does not wait for the agents: it is sent on the next run of the
  background task.
- It is refused when the configuration is disabled, when CDN purging is turned off
  on the instance (`cdn.enabled: false` or `cdn.worker_interval: 0`), or when it
  exceeds `cdn.max_targets_per_job`.

It is also available through the GraphQL API, as the `purgeProjectCdnUrls` mutation.

## Configuration

```yaml title="manager.yaml"
cdn:
  enabled: true                    # Kill switch: stops every purge
  worker_interval: 30s             # How often queued jobs are sent (0 = disabled)
  agent_convergence_timeout: 30m   # How long a job waits for the agents
  max_targets_per_job: 5000        # Beyond it, a publish is recorded as skipped (0 = no cap)
  http_timeout: 10s                # Timeout of one request to the CDN API
  job_retention: 720h              # How long finished jobs are kept (0 = forever)
```

See [Configuration](../configuration.md#cdn-cache-purge) for details on each setting.

## Permissions

CDN configurations, the purge history and manual purges belong to the **Projects**
admin section:

| Action | Permission |
|--------|------------|
| View configurations and the purge history | `projects` read |
| Create, edit, delete or test a configuration, purge manually | `projects` write |

Project resource permissions are not enough: a user who can edit a project's
redirects cannot change where its cache is purged. CDN credentials are never
displayed back, whatever the permission.

## Monitoring

Every finished job is counted in Prometheus, by provider and outcome. See
[Monitoring](../monitoring.md#cdn-purges) for the metrics and an alert catching
publishes that never reached the CDN.
