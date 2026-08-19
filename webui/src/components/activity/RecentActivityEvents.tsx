import { useQuery } from '@apollo/client/react'
import { Link } from 'react-router-dom'
import { ProjectActivityEventsDocument } from '../../generated/graphql'
import { usePermissions, Action, ResourceType } from '../../hooks/usePermissions'
import { ActivityEventTable } from './ActivityEventTable'

/** Enough to see what just happened, without turning the dashboard into a journal. */
const RECENT_COUNT = 15

interface RecentActivityEventsProps {
  namespaceCode: string
  projectCode: string
}

/**
 * The latest events of a project, unfiltered and unpaginated. Renders nothing when
 * the user cannot read the journal, so the dashboard degrades quietly instead of
 * showing a permission error next to the stats.
 */
export function RecentActivityEvents({ namespaceCode, projectCode }: RecentActivityEventsProps) {
  const { canResource, loading: permissionsLoading } = usePermissions()

  const canRead = canResource(namespaceCode, projectCode, ResourceType.Any, Action.Read)

  const { data, loading, error } = useQuery(ProjectActivityEventsDocument, {
    variables: {
      namespaceCode,
      projectCode,
      pagination: { limit: RECENT_COUNT, offset: 0 },
    },
    skip: !canRead || permissionsLoading,
  })

  if (!canRead || permissionsLoading) return null

  // A failing journal must not break the dashboard it sits on
  if (error) return null

  const events = data?.projectActivityEvents.items ?? []

  return (
    <div className="mt-6">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white">Recent activity</h3>
        <Link
          to={`/${namespaceCode}/${projectCode}/activity`}
          className="text-sm font-medium text-brand-purple hover:underline"
        >
          View full journal
        </Link>
      </div>

      {loading ? (
        <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-8 flex justify-center">
          <div className="h-6 w-6 animate-spin rounded-full border-4 border-brand-purple border-t-transparent"></div>
        </div>
      ) : (
        <ActivityEventTable events={events} emptyLabel="Nothing has happened in this project yet" />
      )}
    </div>
  )
}
