import { useState } from 'react'
import { useQuery, useMutation } from '@apollo/client/react'
import { useSearchParams, useNavigate, useParams } from 'react-router-dom'
import { useDocumentTitle } from '../hooks/useDocumentTitle'
import {
  GetProjectPagesDocument,
  GetProjectDocument,
  CreatePageDraftDocument,
  DeletePageDraftDocument,
  PublishProjectDocument,
  RollbackPageDraftDocument,
  type PageType,
  type PageContentType,
  type Page,
  type DraftChangeType,
  type SortInput,
} from '../generated/graphql'
import { useCurrentProject } from '../hooks/useCurrentProject'
import { usePermissions, Action, ResourceType } from '../hooks/usePermissions'
import { RelativeTime } from '../components/RelativeTime'
import { SearchBar } from '../components/SearchBar'
import { ReloadButton } from '../components/ReloadButton'
import { SortableHeader, parseSortsFromUrl, serializeSortsToUrl } from '../components/SortableHeader'
import { DraftBadge, ConfirmModal } from '../components/redirects'
import { PathCell } from '../components/PathCell'

const PAGE_SIZE = 20

type PageWithDraft = Page

const draftStatusFilterLabels: Record<DraftChangeType, string> = {
  CREATE: 'New',
  UPDATE: 'Modified',
  DELETE: 'Deleted',
  PUBLISHED: 'Published',
}

const typeFilterLabels: Record<PageType, string> = {
  BASIC: 'Basic',
  BASIC_HOST: 'Host',
}

const contentTypeLabels: Record<PageContentType, string> = {
  TEXT_PLAIN: 'Text',
  XML: 'XML',
}

function PageTypeBadge({ type }: { type: PageType }) {
  return (
    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-700 dark:bg-slate-700 dark:text-slate-300">
      {typeFilterLabels[type]}
    </span>
  )
}

function ContentTypeBadge({ contentType }: { contentType: PageContentType }) {
  const colors: Record<PageContentType, string> = {
    TEXT_PLAIN: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
    XML: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
  }

  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${colors[contentType]}`}>
      {contentTypeLabels[contentType]}
    </span>
  )
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

export function Pages() {
  const [searchParams, setSearchParams] = useSearchParams()
  const { namespace, project: projectParam } = useParams()
  const navigate = useNavigate()
  const { namespaceCode, projectCode, project, refetch: refetchProject } = useCurrentProject()
  useDocumentTitle(namespaceCode && projectCode ? `Pages - ${namespaceCode}/${projectCode}` : 'Pages')
  const { canResource, loading: permissionsLoading } = usePermissions()

  const canRead = namespaceCode && projectCode ? canResource(namespaceCode, projectCode, ResourceType.Page, Action.Read) : false
  const canWrite = namespaceCode && projectCode ? canResource(namespaceCode, projectCode, ResourceType.Page, Action.Write) : false

  const page = parseInt(searchParams.get('page') || '1', 10) - 1
  const searchFromUrl = searchParams.get('q') || ''
  const typeFilters: PageType[] = (searchParams.get('types')?.split(',').filter(Boolean) as PageType[]) || []
  const contentTypeFilters: PageContentType[] = (searchParams.get('contentTypes')?.split(',').filter(Boolean) as PageContentType[]) || []
  const draftStatusFilters: DraftChangeType[] = (searchParams.get('draftStatus')?.split(',').filter(Boolean) as DraftChangeType[]) || []
  const defaultSort: SortInput[] = [{ column: 'updatedAt', direction: 'DESC' }]
  const parsedSorts = parseSortsFromUrl(searchParams.get('sort'))
  const currentSorts = parsedSorts.length > 0 ? parsedSorts : defaultSort

  const [filtersOpen, setFiltersOpen] = useState(false)
  const [deleteConfirm, setDeleteConfirm] = useState<PageWithDraft | null>(null)
  const [rollbackConfirm, setRollbackConfirm] = useState<PageWithDraft | null>(null)
  const [publishConfirm, setPublishConfirm] = useState(false)
  const [rollbackAllConfirm, setRollbackAllConfirm] = useState(false)

  const updateParams = (updates: { page?: number; q?: string; types?: PageType[]; contentTypes?: PageContentType[]; draftStatus?: DraftChangeType[]; sorts?: SortInput[] }) => {
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

    if (updates.contentTypes !== undefined) {
      if (updates.contentTypes.length === 0) {
        newParams.delete('contentTypes')
      } else {
        newParams.set('contentTypes', updates.contentTypes.join(','))
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

  const { data, loading, error, refetch } = useQuery(GetProjectPagesDocument, {
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
        contentTypes: contentTypeFilters.length > 0 ? contentTypeFilters : null,
        draftStatus: draftStatusFilters.length > 0 ? draftStatusFilters as unknown as DraftChangeType[] : null,
      },
      sort: currentSorts.length > 0 ? currentSorts : null,
    },
    skip: !namespaceCode || !projectCode,
  })

  const [createDraft] = useMutation(CreatePageDraftDocument, {
    refetchQueries: [GetProjectPagesDocument, GetProjectDocument],
  })

  const [deleteDraft] = useMutation(DeletePageDraftDocument, {
    refetchQueries: [GetProjectPagesDocument, GetProjectDocument],
  })

  const [publishProject, { loading: publishLoading }] = useMutation(PublishProjectDocument, {
    refetchQueries: [GetProjectPagesDocument, GetProjectDocument],
  })

  const [rollbackAllDrafts, { loading: rollbackAllLoading }] = useMutation(RollbackPageDraftDocument, {
    refetchQueries: [GetProjectPagesDocument, GetProjectDocument],
  })

  const handleSearch = (value: string) => {
    updateParams({ q: value, page: 0 })
  }

  const toggleTypeFilter = (type: PageType) => {
    const newTypes = typeFilters.includes(type)
      ? typeFilters.filter(t => t !== type)
      : [...typeFilters, type]
    updateParams({ types: newTypes, page: 0 })
  }

  const toggleContentTypeFilter = (contentType: PageContentType) => {
    const newTypes = contentTypeFilters.includes(contentType)
      ? contentTypeFilters.filter(t => t !== contentType)
      : [...contentTypeFilters, contentType]
    updateParams({ contentTypes: newTypes, page: 0 })
  }

  const toggleDraftStatusFilter = (status: DraftChangeType) => {
    const newStatuses = draftStatusFilters.includes(status)
      ? draftStatusFilters.filter(s => s !== status)
      : [...draftStatusFilters, status]
    updateParams({ draftStatus: newStatuses, page: 0 })
  }

  const handleClearFilters = () => {
    updateParams({ types: [], contentTypes: [], draftStatus: [], page: 0 })
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

  const activeFiltersCount = typeFilters.length + contentTypeFilters.length + draftStatusFilters.length

  const handlePageChange = (newPage: number) => {
    updateParams({ page: newPage })
  }

  const handleAddPage = () => {
    navigate(`/${namespace}/${projectParam}/pages/add`)
  }

  const handleEditPage = (pageItem: PageWithDraft) => {
    navigate(`/${namespace}/${projectParam}/pages/edit/${pageItem.id}`)
  }

  const handleDeletePage = (pageItem: PageWithDraft) => {
    setDeleteConfirm(pageItem)
  }

  const handleRollback = (pageItem: PageWithDraft) => {
    setRollbackConfirm(pageItem)
  }

  const handleDeleteConfirm = async () => {
    if (!namespaceCode || !projectCode || !deleteConfirm) return

    try {
      const changeType = deleteConfirm.pageDraft?.changeType
      if (changeType === 'CREATE' && deleteConfirm.pageDraft) {
        await deleteDraft({
          variables: {
            namespaceCode,
            projectCode,
            pageDraftID: deleteConfirm.pageDraft.id,
          },
        })
      } else if (!changeType) {
        await createDraft({
          variables: {
            namespaceCode,
            projectCode,
            input: {
              oldPageID: deleteConfirm.id,
              newPage: null,
            },
          },
        })
      }
      setDeleteConfirm(null)
      refetch()
      refetchProject()
    } catch (err) {
      console.error('Failed to delete page:', err)
    }
  }

  const handleRollbackConfirm = async () => {
    if (!namespaceCode || !projectCode || !rollbackConfirm?.pageDraft) return

    try {
      await deleteDraft({
        variables: {
          namespaceCode,
          projectCode,
          pageDraftID: rollbackConfirm.pageDraft.id,
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

  const getDisplayData = (pageItem: PageWithDraft): { type: PageType; path: string; contentType: PageContentType } => {
    const changeType = pageItem.pageDraft?.changeType

    if ((changeType === 'CREATE' || changeType === 'UPDATE') && pageItem.pageDraft?.newPage) {
      return {
        type: pageItem.pageDraft.newPage.type,
        path: pageItem.pageDraft.newPage.path,
        contentType: pageItem.pageDraft.newPage.contentType,
      }
    }

    return {
      type: pageItem.type,
      path: pageItem.path ?? '',
      contentType: pageItem.contentType ?? 'TEXT_PLAIN',
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
            <p className="text-amber-700 dark:text-amber-400">You don't have permission to view pages for this project.</p>
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4">
        <p className="text-red-700 dark:text-red-400">Error loading pages: {error.message}</p>
      </div>
    )
  }

  const pages = (data?.projectsPages.items ?? []) as PageWithDraft[]
  const total = data?.projectsPages.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)
  const draftCount = project?.countPageDrafts ?? 0

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-slate-900 dark:text-white">Pages</h2>
          <p className="mt-1 text-slate-600 dark:text-slate-400">Manage your static pages (robots.txt, sitemap.xml, etc.)</p>
        </div>
        <div className="flex items-center gap-3">
          {canWrite && draftCount > 0 && (
            <button
              onClick={() => setRollbackAllConfirm(true)}
              disabled={rollbackAllLoading}
              className="flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg border border-red-200 dark:border-red-800 bg-white dark:bg-slate-800 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors disabled:opacity-50"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
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
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
              </svg>
              Publish ({draftCount})
            </button>
          )}
          {canWrite && (
            <button
              onClick={handleAddPage}
              className="flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg bg-gradient-to-r from-brand-purple to-brand-indigo text-white hover:opacity-90 transition-opacity"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 4v16m8-8H4" />
              </svg>
              Add Page
            </button>
          )}
        </div>
      </div>

      <SearchBar
        placeholder="Search by path or content..."
        value={searchFromUrl}
        onSearch={handleSearch}
        leftSlot={<ReloadButton onReload={() => refetch()} loading={loading} />}
      >
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

      {filtersOpen && (
        <div className="mb-4 p-4 rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50">
          <div className="flex flex-col sm:flex-row gap-6">
            <div className="flex-1">
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Type
              </label>
              <div className="flex flex-wrap gap-2">
                {(Object.keys(typeFilterLabels) as PageType[]).map((type) => (
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

            <div className="flex-1">
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Content Type
              </label>
              <div className="flex flex-wrap gap-2">
                {(Object.keys(contentTypeLabels) as PageContentType[]).map((contentType) => (
                  <button
                    key={contentType}
                    onClick={() => toggleContentTypeFilter(contentType)}
                    className={`px-3 py-1.5 text-sm font-medium rounded-lg transition-colors ${
                      contentTypeFilters.includes(contentType)
                        ? 'bg-brand-purple text-white'
                        : 'bg-white dark:bg-slate-700 text-slate-600 dark:text-slate-400 border border-slate-200 dark:border-slate-600 hover:bg-slate-100 dark:hover:bg-slate-600'
                    }`}
                  >
                    {contentTypeLabels[contentType]}
                  </button>
                ))}
              </div>
            </div>

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

      <div className="mb-4">
        <span className="text-sm text-slate-600 dark:text-slate-400">
          {total} page{total !== 1 ? 's' : ''} found
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
                <SortableHeader label="Path" column="path" currentSorts={currentSorts} onSort={handleSort} />
              </th>
              <th className="px-6 py-3 text-left">
                <span className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  Content Type
                </span>
              </th>
              <th className="px-6 py-3 text-left">
                <SortableHeader label="Size" column="contentSize" currentSorts={currentSorts} onSort={handleSort} />
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
            {pages.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-6 py-8 text-center text-slate-500 dark:text-slate-400">
                  No pages found
                </td>
              </tr>
            ) : (
              pages.map((pageItem) => {
                const changeType = pageItem.pageDraft?.changeType
                const displayData = getDisplayData(pageItem)

                return (
                  <tr key={pageItem.id} className={getRowClassName(changeType)}>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <DraftBadge changeType={changeType} />
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <PageTypeBadge type={displayData.type} />
                    </td>
                    <td className="px-6 py-4">
                      <PathCell value={displayData.path} className="text-slate-900 dark:text-white" />
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <ContentTypeBadge contentType={displayData.contentType} />
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-600 dark:text-slate-400">
                      {formatSize(Number(pageItem.contentSize))}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-slate-600 dark:text-slate-400">
                      <RelativeTime date={pageItem.updatedAt} />
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right">
                      <div className="flex items-center justify-end gap-2">
                        {canWrite && changeType && (
                          <button
                            onClick={() => handleRollback(pageItem)}
                            className="p-2 rounded-lg transition-colors text-slate-600 hover:text-amber-600 hover:bg-amber-50 dark:text-slate-400 dark:hover:text-amber-400 dark:hover:bg-amber-900/20"
                            title="Rollback change"
                          >
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
                            </svg>
                          </button>
                        )}

                        {canWrite && changeType !== 'DELETE' && (
                          <button
                            onClick={() => handleEditPage(pageItem)}
                            className="p-2 rounded-lg transition-colors text-slate-600 hover:text-brand-purple hover:bg-brand-purple/10 dark:text-slate-400 dark:hover:text-brand-purple"
                            title="Edit page"
                          >
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                            </svg>
                          </button>
                        )}

                        {canWrite && !changeType && (
                          <button
                            onClick={() => handleDeletePage(pageItem)}
                            className="p-2 rounded-lg transition-colors text-slate-600 hover:text-red-600 hover:bg-red-50 dark:text-slate-400 dark:hover:text-red-400 dark:hover:bg-red-900/20"
                            title="Delete page"
                          >
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
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

      {deleteConfirm && (
        <ConfirmModal
          title="Delete Page"
          message={
            deleteConfirm.pageDraft?.changeType === 'CREATE' ? (
              <p>
                Are you sure you want to delete this new page at{' '}
                <strong className="font-mono">{deleteConfirm.path}</strong>? This will remove the
                draft immediately.
              </p>
            ) : (
              <p>
                Are you sure you want to delete the page at{' '}
                <strong className="font-mono">{deleteConfirm.path}</strong>? This will create a draft
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

      {rollbackConfirm && (
        <ConfirmModal
          title="Rollback Change"
          message={
            rollbackConfirm.pageDraft?.changeType === 'CREATE' ? (
              <p>
                Are you sure you want to cancel this new page? It will be removed from the draft
                list.
              </p>
            ) : (
              <p>
                Are you sure you want to rollback this change? The page will be restored to its
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

      {publishConfirm && (
        <ConfirmModal
          title="Publish Changes"
          message={
            <p>
              Are you sure you want to publish <strong>{draftCount}</strong> page change
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

      {rollbackAllConfirm && (
        <ConfirmModal
          title="Rollback All Changes"
          message={
            <p>
              Are you sure you want to rollback <strong>{draftCount}</strong> pending page change
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
    </div>
  )
}