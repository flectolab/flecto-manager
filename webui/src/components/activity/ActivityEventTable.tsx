import type { ProjectActivityEventsQuery } from '../../generated/graphql'
import { RelativeTime } from '../RelativeTime'
import { ActivityActionBadge, ActivityActor, ActivityResourceBadge } from './ActivityBadges'
import { ActivityEventSummary } from './ActivityEventSummary'

/** Derived from the generated query so a schema change surfaces at compile time. */
export type ActivityEventRow = ProjectActivityEventsQuery['projectActivityEvents']['items'][number]

interface ActivityEventTableProps {
  events: readonly ActivityEventRow[]
  emptyLabel?: string
}

/**
 * Journal rows, shared by the activity page and the dashboard widget so both read the
 * same way. Resource and action sit in their own columns: as a single cell their
 * badges shifted with the resource name length and nothing lined up vertically.
 */
export function ActivityEventTable({ events, emptyLabel = 'No activity events found' }: ActivityEventTableProps) {
  return (
    <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 overflow-hidden">
      {/* Fixed layout on purpose: with auto layout the detail cell grows to fit its
          content, which defeats the truncation of the one-line summary. */}
      <table className="w-full table-fixed">
        <thead className="bg-slate-50 dark:bg-slate-700/50">
          <tr>
            <th className="w-40 px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              When
            </th>
            <th className="w-48 px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              Who
            </th>
            <th className="w-28 px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              Resource
            </th>
            <th className="w-32 px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              Action
            </th>
            <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              Detail
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
          {events.length === 0 ? (
            <tr>
              <td colSpan={5} className="px-6 py-8 text-center text-slate-500 dark:text-slate-400">
                {emptyLabel}
              </td>
            </tr>
          ) : (
            events.map((event) => (
              <tr key={event.id}>
                {/* whitespace-nowrap, not truncate: truncate brings overflow-hidden,
                    which clips the tooltip RelativeTime positions above the cell. The
                    column is wide enough for the date anyway. */}
                <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600 dark:text-slate-400">
                  <RelativeTime date={event.occurredAt} />
                </td>
                <td className="px-6 py-4 text-sm">
                  <ActivityActor actor={event.actor} authType={event.authType} />
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <ActivityResourceBadge resource={event.resource} />
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <ActivityActionBadge action={event.action} />
                </td>
                <td className="px-6 py-4 text-sm">
                  <ActivityEventSummary resource={event.resource} action={event.action} data={event.data} />
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  )
}
