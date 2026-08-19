import type { ActivityAction, ActivityResource } from '../generated/graphql'

/**
 * Payload types mirroring model/activity_payload.go.
 *
 * The `data` field of an activity event is an opaque JSON scalar, so the compiler
 * cannot check it against the server. These types are the single place where that
 * contract is written down: keep them in sync with the Go structs, and never widen
 * a renderer to read a field the Go side does not send.
 */

/** Key an event is rendered by. Mirrors the (resource, action) pair server-side. */
export type ActivityEventKey = `${ActivityResource}_${ActivityAction}`

export function activityEventKey(resource: ActivityResource, action: ActivityAction): ActivityEventKey {
  return `${resource}_${action}`
}

export interface RedirectSnapshot {
  type: string
  source: string
  target: string
  status: string
}

/** A page snapshot never carries the page content, only its size. */
export interface PageSnapshot {
  type: string
  path: string
  contentType: string
  contentSize: number
}

/** before is absent on a creation, after is absent on a deletion. */
export interface ActivityChange<T> {
  before?: T
  after?: T
}

export type ActivityRollbackScope = 'SINGLE' | 'PROJECT'

export type DraftChangeType = 'CREATE' | 'UPDATE' | 'DELETE' | 'PUBLISHED'

export interface ActivityDraftCounts {
  create: number
  update: number
  delete: number
}

/** SINGLE carries changeType and entry, PROJECT carries discarded. */
export interface ActivityRollback<T> {
  scope: ActivityRollbackScope
  changeType?: DraftChangeType
  entry?: T
  discarded?: ActivityDraftCounts
}

export interface ActivityImportError {
  line: number
  source?: string
  reason: string
}

export interface ActivityImport {
  filename?: string
  overwrite: boolean
  totalLines: number
  imported: number
  skipped: number
  errorCount: number
  /** Capped server-side: errorCount > errorSample.length means it was truncated. */
  errorSample?: ActivityImportError[]
}

export interface ActivityPublishCounts {
  created: number
  updated: number
  deleted: number
}

export interface ActivityPublish {
  version: number
  redirects: ActivityPublishCounts
  pages: ActivityPublishCounts
}

/**
 * Which payload each event key carries is expressed where it is consumed, in
 * `components/activity/describeActivityEvent.ts`. A key that file does not know falls
 * back to the raw payload, which is what lets this webui keep working against a
 * server recording event types it has never heard of.
 */

/** Wiping a resource of a project. `version` is the project version it produced. */
export interface ActivityTruncate {
  published: number
  drafts: number
  version?: number
}
