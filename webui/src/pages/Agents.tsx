import { useState } from 'react'
import { useQuery } from '@apollo/client/react'
import { useSearchParams } from 'react-router-dom'
import { useDocumentTitle } from '../hooks/useDocumentTitle'
import {
  SearchAgentsDocument,
  type AgentType,
  type AgentStatus,
  type SortInput,
} from '../generated/graphql'
import { useCurrentProject } from '../hooks/useCurrentProject'
import { usePermissions, Action, ResourceType } from '../hooks/usePermissions'
import { SearchBar } from '../components/SearchBar'
import { ReloadButton } from '../components/ReloadButton'
import { SortableHeader, parseSortsFromUrl, serializeSortsToUrl } from '../components/SortableHeader'
import { AgentCard } from '../components/agents'

const PAGE_SIZE = 12

const typeFilterLabels: Record<AgentType, string> = {
  default: 'Default',
  traefik: 'Traefik',
}

const statusFilterLabels: Record<AgentStatus, string> = {
  success: 'Success',
  error: 'Error',
}

export function Agents() {
  const [searchParams, setSearchParams] = useSearchParams()
  const { namespaceCode, projectCode, project } = useCurrentProject()
  useDocumentTitle(namespaceCode && projectCode ? `Agents - ${namespaceCode}/${projectCode}` : 'Agents')
  const { canResource, loading: permissionsLoading } = usePermissions()

  const canRead = namespaceCode && projectCode ? canResource(namespaceCode, projectCode, ResourceType.Agent, Action.Read) : false

  const page = parseInt(searchParams.get('page') || '1', 10) - 1
  const searchFromUrl = searchParams.get('q') || ''
  const typeFilters: AgentType[] = (searchParams.get('types')?.split(',').filter(Boolean) as AgentType[]) || []
  const statusFilters: AgentStatus[] = (searchParams.get('status')?.split(',').filter(Boolean) as AgentStatus[]) || []
  const showOffline = searchParams.get('offline') === 'true'
  const defaultSort: SortInput[] = [{ column: 'lastHitAt', direction: 'DESC' }]
  const parsedSorts = parseSortsFromUrl(searchParams.get('sort'))
  const currentSorts = parsedSorts.length > 0 ? parsedSorts : defaultSort

  const [filtersOpen, setFiltersOpen] = useState(false)

  const updateParams = (updates: { page?: number; q?: string; types?: AgentType[]; status?: AgentStatus[]; sorts?: SortInput[]; offline?: boolean }) => {
    const newParams = new URLSearchParams(searchParams)

    if (updates.page !== undefined) {
      if (updates.page === 0) {
        newParams.delete('page')
      } else {
        newParams.set('page', String(updates.page + 1))
      }
    }

    if (updates.q !== undefined) {
      if (updates.q === '') {
        newParams.delete('q')
      } else {
        newParams.set('q', updates.q)
      }
    }

    if (updates.types !== undefined) {
      if (updates.types.length === 0) {
        newParams.delete('types')
      } else {
        newParams.set('types', updates.types.join(','))
      }
    }

    if (updates.status !== undefined) {
      if (updates.status.length === 0) {
        newParams.delete('status')
      } else {
        newParams.set('status', updates.status.join(','))
      }
    }

    if (updates.offline !== undefined) {
      if (!updates.offline) {
        newParams.delete('offline')
      } else {
        newParams.set('offline', 'true')
      }
    }

    if (updates.sorts !== undefined) {
      if (updates.sorts.length === 0) {
        newParams.delete('sort')
      } else {
        newParams.set('sort', serializeSortsToUrl(updates.sorts))
      }
    }

    setSearchParams(newParams)
  }

  const { data, loading, error, refetch } = useQuery(SearchAgentsDocument, {
    variables: {
      namespaceCode: namespaceCode ?? '',
      projectCode: projectCode ?? '',
      pagination: {
        limit: PAGE_SIZE,
        offset: page * PAGE_SIZE,
      },
      filter: {
        search: searchFromUrl || null,
        types: typeFilters.length > 0 ? typeFilters : null,
        status: statusFilters.length > 0 ? statusFilters : null,
        showOffline,
      },
      sort: currentSorts.length > 0 ? currentSorts : null,
    },
    skip: !namespaceCode || !projectCode,
  })

  const handleSearch = (value: string) => {
    updateParams({ q: value, page: 0 })
  }

  const toggleTypeFilter = (type: AgentType) => {
    const newTypes = typeFilters.includes(type)
      ? typeFilters.filter(t => t !== type)
      : [...typeFilters, type]
    updateParams({ types: newTypes, page: 0 })
  }

  const toggleStatusFilter = (status: AgentStatus) => {
    const newStatuses = statusFilters.includes(status)
      ? statusFilters.filter(s => s !== status)
      : [...statusFilters, status]
    updateParams({ status: newStatuses, page: 0 })
  }

  const handleClearFilters = () => {
    updateParams({ types: [], status: [], offline: false, page: 0 })
  }

  const handleSort = (column: string) => {
    const existingIndex = currentSorts.findIndex(s => s.column === column)

    if (existingIndex === -1) {
      updateParams({ sorts: [...currentSorts, { column, direction: 'ASC' }], page: 0 })
    } else {
      const existing = currentSorts[existingIndex]
      if (existing.direction === 'ASC') {
        const newSorts = [...currentSorts]
        newSorts[existingIndex] = { column, direction: 'DESC' }
        updateParams({ sorts: newSorts, page: 0 })
      } else {
        updateParams({ sorts: currentSorts.filter(s => s.column !== column), page: 0 })
      }
    }
  }

  const activeFiltersCount = typeFilters.length + statusFilters.length + (showOffline ? 1 : 0)

  const handlePageChange = (newPage: number) => {
    updateParams({ page: newPage })
  }

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
            <p className="text-amber-700 dark:text-amber-400">You don't have permission to view agents for this project.</p>
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4">
        <p className="text-red-700 dark:text-red-400">Error loading agents: {error.message}</p>
      </div>
    )
  }

  const agents = data?.searchAgents.items ?? []
  const total = data?.searchAgents.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)

  return (
    <div>
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-slate-900 dark:text-white">Agent Status</h2>
        <p className="mt-1 text-slate-600 dark:text-slate-400">Monitor your agents health and status</p>
      </div>

      <SearchBar
        placeholder="Search by agent name..."
        value={searchFromUrl}
        onSearch={handleSearch}
        leftSlot={<ReloadButton onReload={() => refetch()} loading={loading} />}
      >
        {/* Filter Toggle Button */}
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
          <svg
            className={`w-4 h-4 transition-transform ${filtersOpen ? 'rotate-180' : ''}`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </button>
      </SearchBar>

      {/* Collapsible Filters Panel */}
      {filtersOpen && (
        <div className="mb-4 p-4 rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50">
          <div className="flex flex-col sm:flex-row gap-6">
            {/* Type Filter */}
            <div className="flex-1">
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Type
              </label>
              <div className="flex flex-wrap gap-2">
                {(Object.keys(typeFilterLabels) as AgentType[]).map((type) => (
                  <button
                    key={type}
                    onClick={() => toggleTypeFilter(type)}
                    className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                      typeFilters.includes(type)
                        ? 'bg-brand-purple text-white'
                        : 'bg-white dark:bg-slate-700 text-slate-600 dark:text-slate-400 border border-slate-200 dark:border-slate-600 hover:bg-slate-100 dark:hover:bg-slate-600'
                    }`}
                  >
                    {typeFilterLabels[type]}
                  </button>
                ))}
              </div>
            </div>

            {/* Status Filter */}
            <div className="flex-1">
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Status
              </label>
              <div className="flex flex-wrap gap-2">
                {(Object.keys(statusFilterLabels) as AgentStatus[]).map((status) => (
                  <button
                    key={status}
                    onClick={() => toggleStatusFilter(status)}
                    className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                      statusFilters.includes(status)
                        ? status === 'success'
                          ? 'bg-green-500 text-white'
                          : 'bg-red-500 text-white'
                        : 'bg-white dark:bg-slate-700 text-slate-600 dark:text-slate-400 border border-slate-200 dark:border-slate-600 hover:bg-slate-100 dark:hover:bg-slate-600'
                    }`}
                  >
                    {statusFilterLabels[status]}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* Show Offline Toggle */}
          <div className="mt-4 pt-4 border-t border-slate-200 dark:border-slate-700">
            <label className="flex items-center gap-3 cursor-pointer">
              <button
                type="button"
                role="switch"
                aria-checked={showOffline}
                onClick={() => updateParams({ offline: !showOffline, page: 0 })}
                className={`relative inline-flex h-6 w-11 shrink-0 rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-brand-purple focus:ring-offset-2 ${
                  showOffline ? 'bg-brand-purple' : 'bg-slate-200 dark:bg-slate-600'
                }`}
              >
                <span
                  className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                    showOffline ? 'translate-x-5' : 'translate-x-0'
                  }`}
                />
              </button>
              <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
                Show offline agents
              </span>
            </label>
          </div>

          {/* Clear Filters */}
          {activeFiltersCount > 0 && (
            <div className="mt-4 pt-4 border-t border-slate-200 dark:border-slate-700">
              <button
                onClick={handleClearFilters}
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

      {/* Sort Controls */}
      <div className="mb-4 flex items-center justify-between">
        <span className="text-sm text-slate-600 dark:text-slate-400">
          {total} agent{total !== 1 ? 's' : ''} found
        </span>
        <div className="flex items-center gap-4">
          <span className="text-xs text-slate-500 dark:text-slate-400">Sort by:</span>
          <div className="flex gap-3">
            <SortableHeader label="Name" column="name" currentSorts={currentSorts} onSort={handleSort} />
            <SortableHeader label="Type" column="type" currentSorts={currentSorts} onSort={handleSort} />
            <SortableHeader label="Status" column="status" currentSorts={currentSorts} onSort={handleSort} />
            <SortableHeader label="Last Hit" column="lastHitAt" currentSorts={currentSorts} onSort={handleSort} />
          </div>
        </div>
      </div>

      {/* Agent Cards Grid */}
      {agents.length === 0 ? (
        <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-8 text-center">
          <svg className="w-12 h-12 mx-auto text-slate-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
          </svg>
          <p className="text-slate-500 dark:text-slate-400">No agents found</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {agents.map((agent) => (
            <AgentCard key={agent.name} agent={agent} projectVersion={project?.version ?? 0} />
          ))}
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="mt-6 flex items-center justify-between">
          <span className="text-sm text-slate-600 dark:text-slate-400">
            Page {page + 1} of {totalPages}
          </span>
          <div className="flex gap-2">
            <button
              onClick={() => handlePageChange(Math.max(0, page - 1))}
              disabled={page === 0}
              className="px-3 py-1.5 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Previous
            </button>
            <button
              onClick={() => handlePageChange(Math.min(totalPages - 1, page + 1))}
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