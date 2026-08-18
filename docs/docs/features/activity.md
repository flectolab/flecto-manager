---
sidebar_position: 4
---

# Activity Journal

Every project keeps a journal of who changed what: redirects and pages created,
modified or deleted, imports, discarded changes and publishes.

:::info Informational, not an audit trail
The journal is meant to answer "what happened to this project recently". It is
deliberately **not** an audit log: entries are purged automatically, payloads are
capped, and page contents are never stored. Do not rely on it for compliance or for
restoring a previous state.
:::

## What is recorded

| Action | Recorded as | Detail kept |
|--------|-------------|-------------|
| Create a redirect or page | `CREATE` | The new version |
| Modify one | `UPDATE` | Only the fields that changed, before and after |
| Delete one | `DELETE` | The version that was removed |
| Discard a pending change | `ROLLBACK` | What was cancelled |
| Discard all pending changes | `ROLLBACK` | How many creations, updates and deletions were dropped |
| Import redirects | `IMPORT` | File name, counts, and a sample of the errors |
| Publish the project | `PUBLISH` | The new version and what it moved |
| Truncate all redirects or pages | `TRUNCATE` | How many published entries and drafts were removed, and the version it published |
| Truncate the journal | `TRUNCATE` | How many entries were removed |

Bulk operations produce **one** entry, not one per item: importing 200,000 redirects
adds a single row summarising the import, and publishing 50,000 changes adds a single
row with the counts.

Page contents are never recorded, only the path, type and size. A journal entry tells
you a page changed and how big it became, not what was inside it.

## Who acted

Each entry keeps the name that acted: a username for a signed-in user, or the token
name when the change came through an API token.

The name is stored as a snapshot, so the journal stays readable after a user is
renamed or deleted. Deleting a user only removes the link to their account — their
name remains next to what they did.

## Retention

The journal is bounded by a **maximum number of entries per project**, not by age.
Older entries beyond that cap are removed by a background task.

```yaml title="manager.yaml"
activity:
  max_events_per_project: 1000  # 0 = unlimited (unbounded table)
  purge_interval: 1h            # 0 = disable the purge task
```

A cap gives a guaranteed maximum size — `projects × max_events_per_project` — which a
retention period cannot: a busy project can write far more entries in 90 days than a
quiet one does in years. The trade-off is that history depth varies: an active project
may only keep a few days, a quiet one several months.

The task also runs once at startup, so an instance that was down while entries piled
up trims immediately instead of waiting a full interval.

To trim right away, for example after lowering the cap:

```bash
flecto-manager db activity-purge -c /etc/flecto/manager.yaml
```

The purge is monitored like any background task, see
[Monitoring](../monitoring.md#background-tasks).

## Permissions

Read access to **any** resource of a project grants access to its journal, since the
journal spans redirects, pages and the project itself. No dedicated permission is
needed.

Note the consequence: a user with only `redirect:read` also sees page entries — their
paths and sizes, never their contents.

## Clearing the journal

An administrator can clear a project's journal from the admin project page. A single
entry is kept afterwards, recording who cleared it and how many entries went — so an
empty journal explains itself instead of looking like a project where nothing ever
happened.

## Deleting a project

A project's journal is deleted with the project. This is intentional: without it, a
project recreated under the same code would inherit the previous one's history.
