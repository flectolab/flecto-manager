import { useState } from 'react'
import { useQuery, useMutation } from '@apollo/client/react'
import { useSearchParams, useNavigate, useParams } from 'react-router-dom'
import { useDocumentTitle } from '../hooks/useDocumentTitle'
import {
  GetProjectRedirectsDocument,
  GetProjectDocument,
  CreateRedirectDraftDocument,
  DeleteRedirectDraftDocument,
  PublishProjectDocument,
  RollbackRedirectDraftDocument,
  type RedirectType,
  type RedirectStatus,
  type Redirect,
  type DraftChangeType,
  type SortInput,
} from '../generated/graphql'
import { useCurrentProject } from '../hooks/useCurrentProject'
import { usePermissions, Action, ResourceType } from '../hooks/usePermissions'
import { RelativeTime } from '../components/RelativeTime'
import { SearchBar } from '../components/SearchBar'
import { ReloadButton } from '../components/ReloadButton'
import { SortableHeader, parseSortsFromUrl, serializeSortsToUrl } from '../components/SortableHeader'
import {
  RedirectStatusBadge,
  RedirectTypeBadge,
  DraftBadge,
  ConfirmModal,
  DiffModal,
  ImportModal,
} from '../components/redirects'
import { PathCell } from '../components/PathCell'

const PAGE_SIZE = 20

// Type alias for redirect (already includes redirectDraft)
type RedirectWithDraft = Redirect

// DraftChangeType includes PUBLISHED for filtering purposes
const draftStatusFilterLabels: Record<DraftChangeType, string> = {
  CREATE: 'New',
  UPDATE: 'Modified',
  DELETE: 'Deleted',
  PUBLISHED: 'Published',
}

const typeFilterLabels: Record<RedirectType, string> = {
  BASIC: 'Basic',
  BASIC_HOST: 'Host',
  REGEX: 'Regex',
  REGEX_HOST: 'Regex Host',
}

export function Redirects() {
  const [searchParams, setSearchParams] = useSearchParams()
  const { namespace, project: projectParam } = useParams()
  const navigate = useNavigate()
  const { namespaceCode, projectCode, project, refetch: refetchProject } = useCurrentProject()
  useDocumentTitle(namespaceCode && projectCode ? `Redirects - ${namespaceCode}/${projectCode}` : 'Redirects')
  const { canResource, loading: permissionsLoading } = usePermissions()

  // Permission checks
  const canRead = namespaceCode && projectCode ? canResource(namespaceCode, projectCode, ResourceType.Redirect, Action.Read) : false
  const canWrite = namespaceCode && projectCode ? canResource(namespaceCode, projectCode, ResourceType.Redirect, Action.Write) : false

  const page = parseInt(searchParams.get('page') || '1', 10) - 1
  const searchFromUrl = searchParams.get('q') || ''
  // Parse comma-separated filter values from URL
  const typeFilters: RedirectType[] = (searchParams.get('types')?.split(',').filter(Boolean) as RedirectType[]) || []
  const draftStatusFilters: DraftChangeType[] = (searchParams.get('draftStatus')?.split(',').filter(Boolean) as DraftChangeType[]) || []
  // Parse sort from URL (default to updatedAt DESC)
  const defaultSort: SortInput[] = [{ column: 'updatedAt', direction: 'DESC' }]
  const parsedSorts = parseSortsFromUrl(searchParams.get('sort'))
  const currentSorts = parsedSorts.length > 0 ? parsedSorts : defaultSort

  const [filtersOpen, setFiltersOpen] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState<RedirectWithDraft | null>(null)
  const [rollbackConfirm, setRollbackConfirm] = useState<RedirectWithDraft | null>(null)
  const [publishConfirm, setPublishConfirm] = useState(false)
  const [rollbackAllConfirm, setRollbackAllConfirm] = useState(false)
  const [diffView, setDiffView] = useState<RedirectWithDraft | null>(null)
  const [importModalOpen, setImportModalOpen] = useState(false)

  const updateParams = (updates: { page?: number; q?: string; types?: RedirectType[]; draftStatus?: DraftChangeType[]; sorts?: SortInput[] }) => {
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

    if (updates.draftStatus !== undefined) {
      if (updates.draftStatus.length === 0) {
        newParams.delete('draftStatus')
      } else {
        newParams.set('draftStatus', updates.draftStatus.join(','))
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

  const { data, loading, error, refetch } = useQuery(GetProjectRedirectsDocument, {
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
        draftStatus: draftStatusFilters.length > 0 ? draftStatusFilters as unknown as DraftChangeType[] : null,
      },
      sort: currentSorts.length > 0 ? currentSorts : null,
    },
    skip: !namespaceCode || !projectCode,
  })

  const [createDraft] = useMutation(CreateRedirectDraftDocument, {
    refetchQueries: [GetProjectRedirectsDocument, GetProjectDocument],
  })

  const [deleteDraft] = useMutation(DeleteRedirectDraftDocument, {
    refetchQueries: [GetProjectRedirectsDocument, GetProjectDocument],
  })

  const [publishProject, { loading: publishLoading }] = useMutation(PublishProjectDocument, {
    refetchQueries: [GetProjectRedirectsDocument, GetProjectDocument],
  })

  const [rollbackAllDrafts, { loading: rollbackAllLoading }] = useMutation(RollbackRedirectDraftDocument, {
    refetchQueries: [GetProjectRedirectsDocument, GetProjectDocument],
  })

  const handleSearch = (value: string) => {
    updateParams({ q: value, page: 0 })
  }

  const toggleTypeFilter = (type: RedirectType) => {
    const newTypes = typeFilters.includes(type)
      ? typeFilters.filter(t => t !== type)
      : [...typeFilters, type]
    updateParams({ types: newTypes, page: 0 })
  }

  const toggleDraftStatusFilter = (status: DraftChangeType) => {
    const newStatuses = draftStatusFilters.includes(status)
      ? draftStatusFilters.filter(s => s !== status)
      : [...draftStatusFilters, status]
    updateParams({ draftStatus: newStatuses, page: 0 })
  }

  const handleClearFilters = () => {
    updateParams({ types: [], draftStatus: [], page: 0 })
  }

  const handleSort = (column: string) => {
    const existingIndex = currentSorts.findIndex(s => s.column === column)

    if (existingIndex === -1) {
      // Add new sort column
      updateParams({ sorts: [...currentSorts, { column, direction: 'ASC' }], page: 0 })
    } else {
      const existing = currentSorts[existingIndex]
      if (existing.direction === 'ASC') {
        // Toggle to DESC
        const newSorts = [...currentSorts]
        newSorts[existingIndex] = { column, direction: 'DESC' }
        updateParams({ sorts: newSorts, page: 0 })
      } else {
        // Remove sort
        updateParams({ sorts: currentSorts.filter(s => s.column !== column), page: 0 })
      }
    }
  }

  const activeFiltersCount = typeFilters.length + draftStatusFilters.length

  const handlePageChange = (newPage: number) => {
    updateParams({ page: newPage })
  }

  const handleAddRedirect = () => {
    navigate(`/${namespace}/${projectParam}/redirects/add`)
  }

  const handleEditRedirect = (redirect: RedirectWithDraft) => {
    navigate(`/${namespace}/${projectParam}/redirects/edit/${redirect.id}`)
  }

  const handleDeleteRedirect = (redirect: RedirectWithDraft) => {
    setDeleteConfirm(redirect)
  }

  const handleRollback = (redirect: RedirectWithDraft) => {
    setRollbackConfirm(redirect)
  }

  const handleDeleteConfirm = async () => {
    if (!namespaceCode || !projectCode || !deleteConfirm) return

    try {
      const changeType = deleteConfirm.redirectDraft?.changeType
      if (changeType === 'CREATE' && deleteConfirm.redirectDraft) {
        // Deleting a CREATE draft - just delete the draft directly
        await deleteDraft({
          variables: {
            namespaceCode,
            projectCode,
            redirectDraftID: deleteConfirm.redirectDraft.id,
          },
        })
      } else if (!changeType) {
        // Deleting an existing published redirect - create a DELETE draft
        await createDraft({
          variables: {
            namespaceCode,
            projectCode,
            input: {
              oldRedirectID: deleteConfirm.id,
              newRedirect: null,
            },
          },
        })
      }
      setDeleteConfirm(null)
      refetch()
      refetchProject()
    } catch (err) {
      console.error('Failed to delete redirect:', err)
    }
  }

  const handleRollbackConfirm = async () => {
    if (!namespaceCode || !projectCode || !rollbackConfirm?.redirectDraft) return

    try {
      await deleteDraft({
        variables: {
          namespaceCode,
          projectCode,
          redirectDraftID: rollbackConfirm.redirectDraft.id,
        },
      })
      setRollbackConfirm(null)
      refetch()
      refetchProject()
    } catch (err) {
      console.error('Failed to rollback draft:', err)
    }
  }

  const handlePublish = () => {
    setPublishConfirm(true)
  }

  const handlePublishConfirm = async () => {
    if (!namespaceCode || !projectCode) return

    try {
      await publishProject({
        variables: {
          namespaceCode,
          projectCode,
        },
      })
      setPublishConfirm(false)
      refetch()
      refetchProject()
    } catch (err) {
      console.error('Failed to publish:', err)
    }
  }

  const handleRollbackAllConfirm = async () => {
    if (!namespaceCode || !projectCode) return

    try {
      await rollbackAllDrafts({
        variables: {
          namespaceCode,
          projectCode,
        },
      })
      setRollbackAllConfirm(false)
      refetch()
      refetchProject()
    } catch (err) {
      console.error('Failed to rollback all drafts:', err)
    }
  }

  const getRowClassName = (changeType: DraftChangeType | undefined): string => {
    const baseClass = 'hover:bg-slate-50 dark:hover:bg-slate-700/30'

    switch (changeType) {
      case 'CREATE':
        return `${baseClass} bg-green-50/50 dark:bg-green-900/10`
      case 'UPDATE':
        return `${baseClass} bg-amber-50/50 dark:bg-amber-900/10`
      case 'DELETE':
        return `${baseClass} bg-red-50/50 dark:bg-red-900/10 opacity-60`
      default:
        return baseClass
    }
  }

  const getDisplayData = (redirect: RedirectWithDraft): { type: RedirectType; source: string; target: string; status: RedirectStatus } => {
    const changeType = redirect.redirectDraft?.changeType

    // For CREATE and UPDATE drafts, show the new values from the draft
    if ((changeType === 'CREATE' || changeType === 'UPDATE') && redirect.redirectDraft?.newRedirect) {
      return redirect.redirectDraft.newRedirect
    }

    // For everything else (published, DELETE), show the current values
    return {
      type: redirect.type,
      source: redirect.source ?? '',
      target: redirect.target ?? '',
      status: redirect.status,
    }
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
            <p className="text-amber-700 dark:text-amber-400">You don't have permission to view redirects for this project.</p>
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4">
        <p className="text-red-700 dark:text-red-400">Error loading redirects: {error.message}</p>
      </div>
    )
  }

  const redirects = (data?.projectsRedirects.items ?? []) as RedirectWithDraft[]
  const total = data?.projectsRedirects.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)
  const draftCount = project?.countRedirectDrafts ?? 0

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-slate-900 dark:text-white">Redirects</h2>
          <p className="mt-1 text-slate-600 dark:text-slate-400">Manage your redirect rules</p>
        </div>
        <div className="flex items-center gap-3">
          {canWrite && draftCount > 0 && (
            <button
              onClick={() => setRollbackAllConfirm(true)}
              disabled={rollbackAllLoading}
              className="flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg border border-red-200 dark:border-red-800 bg-white dark:bg-slate-800 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors disabled:opacity-50"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1.5}
                  d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6"
                />
              </svg>
              Rollback All
            </button>
          )}
          {canWrite && draftCount > 0 && (
            <button
              onClick={handlePublish}
              disabled={publishLoading}
              className="flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg bg-amber-500 text-white hover:bg-amber-600 transition-colors disabled:opacity-50"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1.5}
                  d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
                />
              </svg>
              Publish ({draftCount})
            </button>
          )}
          {canWrite && (
            <button
              onClick={() => setImportModalOpen(true)}
              className="flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
              </svg>
              Import
            </button>
          )}
          {canWrite && (
            <button
              onClick={handleAddRedirect}
              className="flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg bg-gradient-to-r from-brand-purple to-brand-indigo text-white hover:opacity-90 transition-opacity"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 4v16m8-8H4" />
              </svg>
              Add Redirect
            </button>
          )}
        </div>
      </div>

      <SearchBar
        placeholder="Search by source or target..."
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
            {/* Type Filter (multi-select) */}
            <div className="flex-1">
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Type
              </label>
              <div className="flex flex-wrap gap-2">
                {(Object.keys(typeFilterLabels) as RedirectType[]).map((type) => (
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

            {/* Draft Status Filter (multi-select) */}
            <div className="flex-1">
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Draft Status
              </label>
              <div className="flex flex-wrap gap-2">
                {(Object.keys(draftStatusFilterLabels) as DraftChangeType[]).map((status) => (
                  <button
                    key={status}
                    onClick={() => toggleDraftStatusFilter(status)}
                    className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                      draftStatusFilters.includes(status)
                        ? status === 'CREATE'
                          ? 'bg-green-500 text-white'
                          : status === 'UPDATE'
                          ? 'bg-amber-500 text-white'
                          : status === 'DELETE'
                          ? 'bg-red-500 text-white'
                          : status === 'PUBLISHED'
                          ? 'bg-slate-500 text-white'
                          : 'bg-brand-purple text-white'
                        : 'bg-white dark:bg-slate-700 text-slate-600 dark:text-slate-400 border border-slate-200 dark:border-slate-600 hover:bg-slate-100 dark:hover:bg-slate-600'
                    }`}
                  >
                    {draftStatusFilterLabels[status]}
                  </button>
                ))}
              </div>
            </div>
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

      {/* Results count */}
      <div className="mb-4">
        <span className="text-sm text-slate-600 dark:text-slate-400">
          {total} redirect{total !== 1 ? 's' : ''} found
        </span>
      </div>

      <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 overflow-hidden">
        <table className="w-full">
          <thead className="bg-slate-50 dark:bg-slate-700/50">
            <tr>
              <th className="px-6 py-3 text-left">
                <span className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  Status
                </span>
              </th>
              <th className="px-6 py-3 text-left">
                <SortableHeader label="Type" column="type" currentSorts={currentSorts} onSort={handleSort} />
              </th>
              <th className="px-6 py-3 text-left">
                <SortableHeader label="Source" column="source" currentSorts={currentSorts} onSort={handleSort} />
              </th>
              <th className="px-6 py-3 text-left">
                <SortableHeader label="Target" column="target" currentSorts={currentSorts} onSort={handleSort} />
              </th>
              <th className="px-6 py-3 text-left">
                <SortableHeader label="Code" column="status" currentSorts={currentSorts} onSort={handleSort} />
              </th>
              <th className="px-6 py-3 text-left">
                <SortableHeader label="Updated" column="updatedAt" currentSorts={currentSorts} onSort={handleSort} />
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
            {redirects.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-6 py-8 text-center text-slate-500 dark:text-slate-400">
                  No redirects found
                </td>
              </tr>
            ) : (
              redirects.map((redirect) => {
                const changeType = redirect.redirectDraft?.changeType
                const displayData = getDisplayData(redirect)

                return (
                  <tr key={redirect.id} className={getRowClassName(changeType)}>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <DraftBadge changeType={changeType} />
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <RedirectTypeBadge type={displayData.type} />
                    </td>
                    <td className="px-6 py-4">
                      <PathCell value={displayData.source} className="text-slate-900 dark:text-white" />
                    </td>
                    <td className="px-6 py-4">
                      <PathCell value={displayData.target} className="text-slate-600 dark:text-slate-400" />
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <RedirectStatusBadge status={displayData.status} />
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-slate-600 dark:text-slate-400">
                      <RelativeTime date={redirect.updatedAt} />
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right">
                      <div className="flex items-center justify-end gap-2">
                        {/* Diff (only for updates) */}
                        {changeType === 'UPDATE' && (
                          <button
                            onClick={() => setDiffView(redirect)}
                            className="p-2 rounded-lg transition-colors text-slate-600 hover:text-cyan-600 hover:bg-cyan-50 dark:text-slate-400 dark:hover:text-cyan-400 dark:hover:bg-cyan-900/20"
                            title="View changes"
                          >
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={1.5}
                                d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                              />
                            </svg>
                          </button>
                        )}

                        {/* Rollback (only for drafts, requires write permission) */}
                        {canWrite && changeType && (
                          <button
                            onClick={() => handleRollback(redirect)}
                            className="p-2 rounded-lg transition-colors text-slate-600 hover:text-amber-600 hover:bg-amber-50 dark:text-slate-400 dark:hover:text-amber-400 dark:hover:bg-amber-900/20"
                            title="Rollback change"
                          >
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={1.5}
                                d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6"
                              />
                            </svg>
                          </button>
                        )}

                        {/* Edit (requires write permission) */}
                        {canWrite && changeType !== 'DELETE' && (
                          <button
                            onClick={() => handleEditRedirect(redirect)}
                            className="p-2 rounded-lg transition-colors text-slate-600 hover:text-brand-purple hover:bg-brand-purple/10 dark:text-slate-400 dark:hover:text-brand-purple"
                            title="Edit redirect"
                          >
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={1.5}
                                d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                              />
                            </svg>
                          </button>
                        )}

                        {/* Delete (requires write permission, only for published redirects without draft) */}
                        {canWrite && !changeType && (
                          <button
                            onClick={() => handleDeleteRedirect(redirect)}
                            className="p-2 rounded-lg transition-colors text-slate-600 hover:text-red-600 hover:bg-red-50 dark:text-slate-400 dark:hover:text-red-400 dark:hover:bg-red-900/20"
                            title="Delete redirect"
                          >
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={1.5}
                                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                              />
                            </svg>
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                )
              })
            )}
          </tbody>
        </table>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="px-6 py-4 border-t border-slate-200 dark:border-slate-700 flex items-center justify-between">
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

      {/* Delete Confirmation Modal */}
      {deleteConfirm && (
        <ConfirmModal
          title="Delete Redirect"
          message={
            deleteConfirm.redirectDraft?.changeType === 'CREATE' ? (
              <p>
                Are you sure you want to delete this new redirect from{' '}
                <strong className="font-mono">{deleteConfirm.source}</strong>? This will remove the
                draft immediately.
              </p>
            ) : (
              <p>
                Are you sure you want to delete the redirect from{' '}
                <strong className="font-mono">{deleteConfirm.source}</strong>? This will create a draft
                that must be published to take effect.
              </p>
            )
          }
          confirmLabel="Delete"
          variant="danger"
          onConfirm={handleDeleteConfirm}
          onCancel={() => setDeleteConfirm(null)}
        />
      )}

      {/* Rollback Confirmation Modal */}
      {rollbackConfirm && (
        <ConfirmModal
          title="Rollback Change"
          message={
            rollbackConfirm.redirectDraft?.changeType === 'CREATE' ? (
              <p>
                Are you sure you want to cancel this new redirect? It will be removed from the draft
                list.
              </p>
            ) : (
              <p>
                Are you sure you want to rollback this change? The redirect will be restored to its
                previous state.
              </p>
            )
          }
          confirmLabel="Rollback"
          variant="warning"
          onConfirm={handleRollbackConfirm}
          onCancel={() => setRollbackConfirm(null)}
        />
      )}

      {/* Publish Confirmation Modal */}
      {publishConfirm && (
        <ConfirmModal
          title="Publish Changes"
          message={
            <p>
              Are you sure you want to publish <strong>{draftCount}</strong> change
              {draftCount > 1 ? 's' : ''}? This will apply all pending changes to the live
              configuration.
            </p>
          }
          confirmLabel="Publish"
          variant="info"
          onConfirm={handlePublishConfirm}
          onCancel={() => setPublishConfirm(false)}
        />
      )}

      {/* Rollback All Confirmation Modal */}
      {rollbackAllConfirm && (
        <ConfirmModal
          title="Rollback All Changes"
          message={
            <p>
              Are you sure you want to rollback <strong>{draftCount}</strong> pending change
              {draftCount > 1 ? 's' : ''}? This will discard all draft changes and restore the
              configuration to its last published state.
            </p>
          }
          confirmLabel="Rollback All"
          variant="danger"
          onConfirm={handleRollbackAllConfirm}
          onCancel={() => setRollbackAllConfirm(false)}
        />
      )}

      {/* Diff View Modal */}
      {diffView && diffView.redirectDraft?.newRedirect && (
        <DiffModal
          oldData={{
            type: diffView.type,
            source: diffView.source ?? '',
            target: diffView.target ?? '',
            status: diffView.status,
          }}
          newData={diffView.redirectDraft.newRedirect}
          onClose={() => setDiffView(null)}
        />
      )}

      {/* Import Modal */}
      {importModalOpen && namespaceCode && projectCode && (
        <ImportModal
          namespaceCode={namespaceCode}
          projectCode={projectCode}
          onClose={() => setImportModalOpen(false)}
          onSuccess={() => {
            refetch()
            refetchProject()
          }}
        />
      )}
    </div>
  )
}
