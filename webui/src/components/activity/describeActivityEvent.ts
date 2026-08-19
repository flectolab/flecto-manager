import { formatSize } from '../../utils/format'
import type { ActivityAction, ActivityResource } from '../../generated/graphql'
import type {
  ActivityChange,
  ActivityDraftCounts,
  ActivityImport,
  ActivityImportError,
  ActivityPublish,
  ActivityPublishCounts,
  ActivityRollback,
  ActivityTruncate,
  DraftChangeType,
  PageSnapshot,
  RedirectSnapshot,
} from '../../types/activity'
import { activityEventKey } from '../../types/activity'

/**
 * One field of an event. `before` set means the field changed and is rendered as
 * `before → after`; `before` absent means `after` is simply the value.
 */
export interface ActivityFieldDescriptor {
  label: string
  before?: string
  after: string
}

/**
 * Normalised description of an event: the single source of truth behind both
 * presentations, the one-line summary in the table and the modal. Adding an event
 * type means describing it here, not writing a renderer.
 */
export interface ActivityDescription {
  fields: ActivityFieldDescriptor[]
  /** Import only: the error sample kept by the server. */
  errors?: ActivityImportError[]
  /** Exact error count, which can exceed the sample above. */
  errorCount?: number
  /** Unknown event types: the payload, shown raw in the modal. */
  raw?: unknown
}

function value(label: string, after: string): ActivityFieldDescriptor {
  return { label, after }
}

function change(label: string, before: string, after: string): ActivityFieldDescriptor {
  return { label, before, after }
}

/** Always lists the three counts, even at zero: a predictable shape reads faster. */
function countsLabel(counts?: ActivityPublishCounts) {
  const c = counts ?? { created: 0, updated: 0, deleted: 0 }
  return `${c.created} created · ${c.updated} updated · ${c.deleted} deleted`
}

function draftCountsLabel(counts?: ActivityDraftCounts) {
  const c = counts ?? { create: 0, update: 0, delete: 0 }
  return `${c.create} created · ${c.update} updated · ${c.delete} deleted`
}

const changeTypeLabels: Record<DraftChangeType, string> = {
  CREATE: 'a pending creation',
  UPDATE: 'a pending update',
  DELETE: 'a pending deletion',
  PUBLISHED: 'a published change',
}

// --- Redirects ---------------------------------------------------------------

const REDIRECT_FIELDS = [
  { key: 'source', label: 'source' },
  { key: 'target', label: 'target' },
  { key: 'type', label: 'type' },
  { key: 'status', label: 'code' },
] as const

function redirectState(redirect: RedirectSnapshot): ActivityFieldDescriptor[] {
  return REDIRECT_FIELDS.map((f) => value(f.label, redirect[f.key]))
}

function describeRedirectChange(payload: ActivityChange<RedirectSnapshot>): ActivityDescription {
  const { before, after } = payload

  if (before && after) {
    const changed = REDIRECT_FIELDS.filter((f) => before[f.key] !== after[f.key])
    if (changed.length === 0) return { fields: redirectState(after) }

    return {
      fields: [
        // The source identifies the redirect, keep it even when unchanged
        ...(changed.some((f) => f.key === 'source') ? [] : [value('source', after.source)]),
        ...changed.map((f) => change(f.label, before[f.key], after[f.key])),
      ],
    }
  }

  const state = after ?? before
  return { fields: state ? redirectState(state) : [] }
}

// --- Pages ------------------------------------------------------------------

const PAGE_FIELDS = [
  { key: 'path', label: 'path' },
  { key: 'contentSize', label: 'size' },
  { key: 'type', label: 'type' },
  { key: 'contentType', label: 'content type' },
] as const

function pageFieldValue(page: PageSnapshot, key: keyof PageSnapshot): string {
  return key === 'contentSize' ? formatSize(page.contentSize) : String(page[key])
}

function pageState(page: PageSnapshot): ActivityFieldDescriptor[] {
  return PAGE_FIELDS.map((f) => value(f.label, pageFieldValue(page, f.key)))
}

function describePageChange(payload: ActivityChange<PageSnapshot>): ActivityDescription {
  const { before, after } = payload

  if (before && after) {
    const changed = PAGE_FIELDS.filter((f) => before[f.key] !== after[f.key])
    if (changed.length === 0) return { fields: pageState(after) }

    return {
      fields: [
        // The path identifies the page, keep it even when unchanged
        ...(changed.some((f) => f.key === 'path') ? [] : [value('path', after.path)]),
        ...changed.map((f) =>
          change(f.label, pageFieldValue(before, f.key), pageFieldValue(after, f.key))
        ),
      ],
    }
  }

  const state = after ?? before
  return { fields: state ? pageState(state) : [] }
}

// --- Rollbacks --------------------------------------------------------------

function describeRedirectRollback(payload: ActivityRollback<RedirectSnapshot>): ActivityDescription {
  if (payload.scope === 'PROJECT') {
    return { fields: [value('discarded', draftCountsLabel(payload.discarded))] }
  }

  const label = payload.changeType
    ? changeTypeLabels[payload.changeType] ?? payload.changeType.toLowerCase()
    : 'a pending change'

  return {
    fields: [
      value('cancelled', label),
      ...(payload.entry ? redirectState(payload.entry) : []),
    ],
  }
}

function describePageRollback(payload: ActivityRollback<PageSnapshot>): ActivityDescription {
  if (payload.scope === 'PROJECT') {
    return { fields: [value('discarded', draftCountsLabel(payload.discarded))] }
  }

  const label = payload.changeType
    ? changeTypeLabels[payload.changeType] ?? payload.changeType.toLowerCase()
    : 'a pending change'

  return {
    fields: [value('cancelled', label), ...(payload.entry ? pageState(payload.entry) : [])],
  }
}

// --- Import and publish -----------------------------------------------------

function describeImport(payload: ActivityImport): ActivityDescription {
  const outcome = [
    `${payload.imported} imported`,
    `${payload.skipped} skipped`,
    `${payload.errorCount} errors`,
  ].join(' · ')

  return {
    fields: [
      ...(payload.filename ? [value('file', payload.filename)] : []),
      value('result', `${outcome} of ${payload.totalLines} lines`),
      ...(payload.overwrite ? [value('mode', 'overwrite')] : []),
    ],
    errors: payload.errorSample,
    errorCount: payload.errorCount,
  }
}

function describeTruncate(payload: ActivityTruncate): ActivityDescription {
  return {
    fields: [
      value('removed', `${payload.published} published · ${payload.drafts} drafts`),
      ...(payload.version ? [value('version', `v${payload.version}`)] : []),
    ],
  }
}

function describePublish(payload: ActivityPublish): ActivityDescription {
  return {
    fields: [
      value('version', `v${payload.version}`),
      value('redirects', countsLabel(payload.redirects)),
      value('pages', countsLabel(payload.pages)),
    ],
  }
}

// --- Entry point ------------------------------------------------------------

type Describer = (data: unknown) => ActivityDescription

const describers: Partial<Record<string, Describer>> = {
  REDIRECT_CREATE: (d) => describeRedirectChange(d as ActivityChange<RedirectSnapshot>),
  REDIRECT_UPDATE: (d) => describeRedirectChange(d as ActivityChange<RedirectSnapshot>),
  REDIRECT_DELETE: (d) => describeRedirectChange(d as ActivityChange<RedirectSnapshot>),
  REDIRECT_ROLLBACK: (d) => describeRedirectRollback(d as ActivityRollback<RedirectSnapshot>),
  REDIRECT_IMPORT: (d) => describeImport(d as ActivityImport),
  PAGE_CREATE: (d) => describePageChange(d as ActivityChange<PageSnapshot>),
  PAGE_UPDATE: (d) => describePageChange(d as ActivityChange<PageSnapshot>),
  PAGE_DELETE: (d) => describePageChange(d as ActivityChange<PageSnapshot>),
  PAGE_ROLLBACK: (d) => describePageRollback(d as ActivityRollback<PageSnapshot>),
  PROJECT_PUBLISH: (d) => describePublish(d as ActivityPublish),
  REDIRECT_TRUNCATE: (d) => describeTruncate(d as ActivityTruncate),
  PAGE_TRUNCATE: (d) => describeTruncate(d as ActivityTruncate),
  ACTIVITY_TRUNCATE: (d) => describeTruncate(d as ActivityTruncate),
}

/**
 * Describes an event for display. An unknown (resource, action) pair falls back to
 * the raw payload rather than failing, which is what lets this webui keep working
 * against a server that records event types it has never heard of.
 */
export function describeActivityEvent(
  resource: ActivityResource,
  action: ActivityAction,
  data: unknown
): ActivityDescription {
  const key = activityEventKey(resource, action)
  const describe = describers[key]

  if (!describe || data === null || data === undefined) {
    return { fields: [value('event', key)], raw: data ?? undefined }
  }

  return describe(data)
}
