import { useState } from 'react'
import { useQuery } from '@apollo/client/react'
import { useSearchParams } from 'react-router-dom'
import { useDocumentTitle } from '../hooks/useDocumentTitle'
import {
  ProjectActivityEventsDocument,
  type ActivityAction,
  type ActivityResource,
} from '../generated/graphql'
import { useCurrentProject } from '../hooks/useCurrentProject'
import { usePermissions, Action, ResourceType } from '../hooks/usePermissions'
import { ReloadButton } from '../components/ReloadButton'
import { ActivityEventTable } from '../components/activity'

const PAGE_SIZE = 25

const resourceFilterLabels: Record<ActivityResource, string> = {
  REDIRECT: 'Redirects',
  PAGE: 'Pages',
  PROJECT: 'Project',
  ACTIVITY: 'Journal',
}

const actionFilterLabels: Record<ActivityAction, string> = {
  CREATE: 'Created',
  UPDATE: 'Updated',
  DELETE: 'Deleted',
  IMPORT: 'Imported',
  ROLLBACK: 'Discarded',
  PUBLISH: 'Published',
  TRUNCATE: 'Truncated',
}

export function Activity() {
  const [searchParams, setSearchParams] = useSearchParams()
  const { namespaceCode, projectCode } = useCurrentProject()
  useDocumentTitle(
    namespaceCode && projectCode ? `Activity - ${namespaceCode}/${projectCode}` : 'Activity'
  )
  const { canResource, loading: permissionsLoading } = usePermissions()

  // The journal spans redirects, pages and the project, so read access to any
  // resource of the project grants it. Mirrors the server-side check.
  const canRead =
    namespaceCode && projectCode
      ? canResource(namespaceCode, projectCode, ResourceType.Any, Action.Read)
      : false

  const page = parseInt(searchParams.get('page') || '1', 10) - 1
  const resourceFilter = (searchParams.get('resource') as ActivityResource | null) || null
  const actionFilter = (searchParams.get('action') as ActivityAction | null) || null
  const actorFilter = searchParams.get('actor') || ''

  const [filtersOpen, setFiltersOpen] = useState(false)

  const updateParams = (updates: {
    page?: number
    resource?: ActivityResource | null
    action?: ActivityAction | null
    actor?: string
  }) => {
    const newParams = new URLSearchParams(searchParams)

    if (updates.page !== undefined) {
      if (updates.page === 0) {
        newParams.delete('page')
      } else {
        newParams.set('page', String(updates.page + 1))
      }
    }

    if (updates.resource !== undefined) {
      if (updates.resource === null) {
        newParams.delete('resource')
      } else {
        newParams.set('resource', updates.resource)
      }
    }

    if (updates.action !== undefined) {
      if (updates.action === null) {
        newParams.delete('action')
      } else {
        newParams.set('action', updates.action)
      }
    }

    if (updates.actor !== undefined) {
      if (updates.actor === '') {
        newParams.delete('actor')
      } else {
        newParams.set('actor', updates.actor)
      }
    }

    setSearchParams(newParams)
  }

  const { data, loading, error, refetch } = useQuery(ProjectActivityEventsDocument, {
    variables: {
      namespaceCode: namespaceCode ?? '',
      projectCode: projectCode ?? '',
      pagination: {
        limit: PAGE_SIZE,
        offset: page * PAGE_SIZE,
      },
      filter: {
        resource: resourceFilter,
        action: actionFilter,
        actor: actorFilter || null,
      },
    },
    skip: !namespaceCode || !projectCode,
  })

  const toggleResource = (resource: ActivityResource) => {
    updateParams({ resource: resourceFilter === resource ? null : resource, page: 0 })
  }

  const toggleAction = (action: ActivityAction) => {
    updateParams({ action: actionFilter === action ? null : action, page: 0 })
  }

  const activeFiltersCount =
    (resourceFilter ? 1 : 0) + (actionFilter ? 1 : 0) + (actorFilter ? 1 : 0)

  if (loading || permissionsLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-purple border-t-transparent"></div>
      </div>
    )
  }

  if (!canRead) {
    return (
      <div className="rounded-xl bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 p-6">
        <div className="flex items-center gap-3">
          <svg className="w-6 h-6 text-amber-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div>
            <h3 className="font-semibold text-amber-800 dark:text-amber-300">Access Denied</h3>
            <p className="text-amber-700 dark:text-amber-400">
              You don't have permission to view the activity journal for this project.
            </p>
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4">
        <p className="text-red-700 dark:text-red-400">Error loading activity events: {error.message}</p>
      </div>
    )
  }

  const events = data?.projectActivityEvents.items ?? []
  const total = data?.projectActivityEvents.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)

  return (
    <div>
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-slate-900 dark:text-white">Activity</h2>
        <p className="mt-1 text-slate-600 dark:text-slate-400">
          Who changed what in this project. Older entries are purged automatically.
        </p>
      </div>

      {/* Toolbar */}
      <div className="mb-4 flex items-center gap-3 flex-wrap">
        <ReloadButton onReload={() => refetch()} loading={loading} />
        <button
          onClick={() => setFiltersOpen(!filtersOpen)}
          className={`flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg border transition-colors ${
            filtersOpen || activeFiltersCount > 0
              ? 'border-brand-purple bg-brand-purple/10 text-brand-purple'
              : 'border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700'
          }`}
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
          </svg>
          Filters
          {activeFiltersCount > 0 && (
            <span className="ml-1 px-1.5 py-0.5 text-xs font-semibold rounded-full bg-brand-purple text-white">
              {activeFiltersCount}
            </span>
          )}
        </button>
        <span className="text-sm text-slate-600 dark:text-slate-400">
          {total} event{total !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Collapsible filters */}
      {filtersOpen && (
        <div className="mb-4 p-4 rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50">
          <div className="flex flex-col sm:flex-row gap-6">
            <div className="flex-1">
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Resource
              </label>
              <div className="flex flex-wrap gap-2">
                {(Object.keys(resourceFilterLabels) as ActivityResource[]).map((resource) => (
                  <button
                    key={resource}
                    onClick={() => toggleResource(resource)}
                    className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                      resourceFilter === resource
                        ? 'bg-brand-purple text-white'
                        : 'bg-white dark:bg-slate-700 text-slate-600 dark:text-slate-400 border border-slate-200 dark:border-slate-600 hover:bg-slate-100 dark:hover:bg-slate-600'
                    }`}
                  >
                    {resourceFilterLabels[resource]}
                  </button>
                ))}
              </div>
            </div>

            <div className="flex-1">
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Action
              </label>
              <div className="flex flex-wrap gap-2">
                {(Object.keys(actionFilterLabels) as ActivityAction[]).map((action) => (
                  <button
                    key={action}
                    onClick={() => toggleAction(action)}
                    className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                      actionFilter === action
                        ? 'bg-brand-purple text-white'
                        : 'bg-white dark:bg-slate-700 text-slate-600 dark:text-slate-400 border border-slate-200 dark:border-slate-600 hover:bg-slate-100 dark:hover:bg-slate-600'
                    }`}
                  >
                    {actionFilterLabels[action]}
                  </button>
                ))}
              </div>
            </div>
          </div>

          <div className="mt-4">
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              Actor
            </label>
            <input
              type="text"
              defaultValue={actorFilter}
              placeholder="Exact username or token name"
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  updateParams({ actor: (e.target as HTMLInputElement).value, page: 0 })
                }
              }}
              className="w-full sm:w-72 px-3 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-700 text-slate-900 dark:text-white placeholder-slate-400"
            />
          </div>

          {activeFiltersCount > 0 && (
            <div className="mt-4 pt-4 border-t border-slate-200 dark:border-slate-700">
              <button
                onClick={() => updateParams({ resource: null, action: null, actor: '', page: 0 })}
                className="text-sm text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300 flex items-center gap-1"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M6 18L18 6M6 6l12 12" />
                </svg>
                Clear all filters
              </button>
            </div>
          )}
        </div>
      )}

      <ActivityEventTable events={events} />

      {totalPages > 1 && (
        <div className="mt-6 flex items-center justify-between">
          <span className="text-sm text-slate-600 dark:text-slate-400">
            Page {page + 1} of {totalPages}
          </span>
          <div className="flex gap-2">
            <button
              onClick={() => updateParams({ page: Math.max(0, page - 1) })}
              disabled={page === 0}
              className="px-3 py-1.5 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Previous
            </button>
            <button
              onClick={() => updateParams({ page: Math.min(totalPages - 1, page + 1) })}
              disabled={page >= totalPages - 1}
              className="px-3 py-1.5 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
