import { useState } from 'react'
import { useQuery, useMutation } from '@apollo/client/react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { SearchProjectsDocument, DeleteProjectDocument, type SortInput } from '../../generated/graphql'
import { usePermissions, AdminSection, Action } from '../../hooks/usePermissions'
import { useDocumentTitle } from '../../hooks/useDocumentTitle'
import { SearchBar } from '../../components/SearchBar'
import { ReloadButton } from '../../components/ReloadButton'
import { SortableHeader, parseSortsFromUrl, serializeSortsToUrl } from '../../components/SortableHeader'
import { RelativeTime } from '../../components/RelativeTime'

const PAGE_SIZE = 10

export function Projects() {
  useDocumentTitle('Admin - Projects')
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()
  const { canAdminResource } = usePermissions()

  const page = parseInt(searchParams.get('page') || '1', 10) - 1
  const searchFromUrl = searchParams.get('q') || ''
  const currentSorts = parseSortsFromUrl(searchParams.get('sort'))

  const [deleteConfirm, setDeleteConfirm] = useState<{ namespaceCode: string; projectCode: string } | null>(null)

  const canWrite = canAdminResource(AdminSection.Projects, Action.Write)

  const updateParams = (updates: { page?: number; q?: string; sorts?: SortInput[] }) => {
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

    if (updates.sorts !== undefined) {
      if (updates.sorts.length === 0) {
        newParams.delete('sort')
      } else {
        newParams.set('sort', serializeSortsToUrl(updates.sorts))
      }
    }

    setSearchParams(newParams)
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

  const { data, loading, error, refetch } = useQuery(SearchProjectsDocument, {
    variables: {
      pagination: { limit: PAGE_SIZE, offset: page * PAGE_SIZE },
      filter: { search: searchFromUrl || null },
      sort: currentSorts.length > 0 ? currentSorts : null,
    },
  })

  const [deleteProject, { loading: deleteLoading }] = useMutation(DeleteProjectDocument)

  const projects = data?.searchProjects.items ?? []
  const total = data?.searchProjects.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)

  const handleViewProject = (namespaceCode: string, projectCode: string) => {
    navigate(`/admin/projects/${encodeURIComponent(namespaceCode)}/${encodeURIComponent(projectCode)}`)
  }

  const handleSearch = (value: string) => {
    updateParams({ q: value, page: 0 })
  }

  const handlePageChange = (newPage: number) => {
    updateParams({ page: newPage })
  }

  const handleDeleteClick = (namespaceCode: string, projectCode: string) => {
    setDeleteConfirm({ namespaceCode, projectCode })
  }

  const handleDeleteConfirm = async () => {
    if (!deleteConfirm) return
    try {
      await deleteProject({
        variables: {
          namespaceCode: deleteConfirm.namespaceCode,
          projectCode: deleteConfirm.projectCode,
        },
      })
      setDeleteConfirm(null)
      refetch()
    } catch (err) {
      console.error('Failed to delete project:', err)
      setDeleteConfirm(null)
    }
  }

  const handleDeleteCancel = () => {
    setDeleteConfirm(null)
  }

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-slate-900 dark:text-white">Project Management</h2>
          <p className="mt-1 text-slate-600 dark:text-slate-400">
            Manage projects across namespaces
          </p>
        </div>
        {canWrite && (
          <button
            onClick={() => navigate('/admin/projects/new')}
            className="flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg bg-gradient-to-r from-brand-purple to-brand-indigo text-white hover:opacity-90 transition-opacity"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            New Project
          </button>
        )}
      </div>

      <SearchBar
        placeholder="Search by code, name or namespace..."
        value={searchFromUrl}
        onSearch={handleSearch}
        leftSlot={<ReloadButton onReload={() => refetch()} loading={loading} />}
      />

      {/* Content */}
      {loading ? (
        <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-8">
          <div className="flex items-center justify-center">
            <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-purple border-t-transparent"></div>
          </div>
        </div>
      ) : error ? (
        <div className="rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4">
          <p className="text-red-700 dark:text-red-400">Error loading projects: {error.message}</p>
        </div>
      ) : projects.length === 0 ? (
        <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-8 text-center">
          <svg
            className="mx-auto h-12 w-12 text-slate-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"
            />
          </svg>
          <p className="mt-4 text-slate-600 dark:text-slate-400">
            {searchFromUrl ? `No projects found for "${searchFromUrl}"` : 'No projects found'}
          </p>
          {canWrite && !searchFromUrl && (
            <button
              onClick={() => navigate('/admin/projects/new')}
              className="mt-4 inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg bg-gradient-to-r from-brand-purple to-brand-indigo text-white hover:opacity-90 transition-opacity"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              Create your first project
            </button>
          )}
        </div>
      ) : (
        <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 overflow-hidden">
          <table className="w-full">
            <thead className="bg-slate-50 dark:bg-slate-700/50">
              <tr>
                <th className="px-6 py-3 text-left">
                  <SortableHeader label="Namespace" column="namespace_code" currentSorts={currentSorts} onSort={handleSort} />
                </th>
                <th className="px-6 py-3 text-left">
                  <SortableHeader label="Code" column="code" currentSorts={currentSorts} onSort={handleSort} />
                </th>
                <th className="px-6 py-3 text-left">
                  <SortableHeader label="Name" column="name" currentSorts={currentSorts} onSort={handleSort} />
                </th>
                <th className="px-6 py-3 text-left">
                  <SortableHeader label="Created" column="createdAt" currentSorts={currentSorts} onSort={handleSort} />
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
              {projects.map((project) => (
                <tr
                  key={`${project.namespace.namespaceCode}-${project.projectCode}`}
                  className="hover:bg-slate-50 dark:hover:bg-slate-700/30"
                >
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="font-mono text-sm font-medium text-slate-900 dark:text-white">
                      {project.namespace.namespaceCode}
                    </span>
                    <p className="text-sm text-slate-500 dark:text-slate-400">
                      {project.namespace.name}
                    </p>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="font-mono text-sm font-medium text-slate-900 dark:text-white">
                      {project.projectCode}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="text-slate-600 dark:text-slate-400">{project.name}</span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                    <RelativeTime date={project.createdAt} />
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 dark:text-slate-400">
                    <RelativeTime date={project.updatedAt} />
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => handleViewProject(project.namespace.namespaceCode, project.projectCode)}
                        className="p-2 rounded-lg transition-colors text-slate-600 hover:text-brand-purple hover:bg-brand-purple/10 dark:text-slate-400 dark:hover:text-brand-purple"
                        title={canWrite ? 'Edit project' : 'View project'}
                      >
                        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          {canWrite ? (
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={1.5}
                              d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                            />
                          ) : (
                            <>
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={1.5}
                                d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                              />
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={1.5}
                                d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
                              />
                            </>
                          )}
                        </svg>
                      </button>
                      {canWrite && (
                        <button
                          onClick={() => handleDeleteClick(project.namespace.namespaceCode, project.projectCode)}
                          className="p-2 text-slate-400 hover:text-red-500 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
                          title="Delete project"
                        >
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                          </svg>
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
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
      )}

      {/* Delete Confirmation Modal */}
      {deleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="absolute inset-0 bg-black/50 backdrop-blur-sm"
            onClick={handleDeleteCancel}
          />
          <div className="relative w-full max-w-md mx-4 rounded-xl bg-white dark:bg-slate-800 shadow-xl border border-slate-200 dark:border-slate-700 p-6">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
              Delete Project
            </h3>
            <p className="text-slate-600 dark:text-slate-400 mb-6">
              Are you sure you want to delete project <strong>{deleteConfirm.projectCode}</strong> from namespace <strong>{deleteConfirm.namespaceCode}</strong>? This action cannot be undone.
            </p>
            <div className="flex gap-3 justify-end">
              <button
                onClick={handleDeleteCancel}
                className="px-4 py-2 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleDeleteConfirm}
                disabled={deleteLoading}
                className="px-4 py-2 text-sm font-medium rounded-lg bg-red-600 text-white hover:bg-red-700 transition-colors disabled:opacity-50"
              >
                {deleteLoading ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}