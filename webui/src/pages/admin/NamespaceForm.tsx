import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useQuery, useMutation } from '@apollo/client/react'
import { GetNamespaceDocument, CreateNamespaceDocument, UpdateNamespaceDocument, DeleteNamespaceDocument, SearchProjectsDocument, DeleteProjectDocument } from '../../generated/graphql'
import { usePermissions, AdminSection, Action, validateCode } from '../../hooks/usePermissions'
import { useDocumentTitle } from '../../hooks/useDocumentTitle'
import { UnsavedChangesIndicator } from '../../components/UnsavedChangesIndicator'
import { RelativeTime } from '../../components/RelativeTime'

const PROJECTS_PAGE_SIZE = 5
const SEARCH_DEBOUNCE_MS = 300

export function NamespaceForm() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { canAdminResource } = usePermissions()

  const isEditing = !!id && id !== 'new'
  useDocumentTitle(isEditing ? `Admin - Edit Namespace` : 'Admin - Add Namespace')
  const canWrite = canAdminResource(AdminSection.Namespaces, Action.Write)
  const isReadOnly = isEditing && !canWrite

  const [namespaceCode, setNamespaceCode] = useState('')
  const [namespaceCodeError, setNamespaceCodeError] = useState('')
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState(false)
  const [isModified, setIsModified] = useState(false)

  // Projects table state
  const [projectsPage, setProjectsPage] = useState(0)
  const [projectsSearchInput, setProjectsSearchInput] = useState('')
  const [projectsSearch, setProjectsSearch] = useState('')
  const [deleteProjectConfirm, setDeleteProjectConfirm] = useState<{ namespaceCode: string; projectCode: string } | null>(null)
  const canWriteProjects = canAdminResource(AdminSection.Projects, Action.Write)

  // Fetch namespace data if editing
  const { data: namespaceData, loading: namespaceLoading } = useQuery(GetNamespaceDocument, {
    variables: { namespaceCode: id! },
    skip: !isEditing,
  })

  const [createNamespace, { loading: createLoading }] = useMutation(CreateNamespaceDocument)
  const [updateNamespace, { loading: updateLoading }] = useMutation(UpdateNamespaceDocument)
  const [deleteNamespace, { loading: deleteLoading }] = useMutation(DeleteNamespaceDocument)

  // Fetch projects for this namespace
  const { data: projectsData, loading: projectsLoading, refetch: refetchProjects } = useQuery(SearchProjectsDocument, {
    variables: {
      pagination: { limit: PROJECTS_PAGE_SIZE, offset: projectsPage * PROJECTS_PAGE_SIZE },
      filter: { namespaceCode: id, search: projectsSearch || null },
    },
    skip: !isEditing,
  })

  const [deleteProject, { loading: deleteProjectLoading }] = useMutation(DeleteProjectDocument)

  const isLoading = createLoading || updateLoading || deleteLoading

  const projects = projectsData?.searchProjects.items ?? []
  const projectsTotal = projectsData?.searchProjects.total ?? 0
  const projectsTotalPages = Math.ceil(projectsTotal / PROJECTS_PAGE_SIZE)

  // Debounce projects search
  useEffect(() => {
    const timer = setTimeout(() => {
      if (projectsSearchInput !== projectsSearch) {
        setProjectsSearch(projectsSearchInput)
        setProjectsPage(0)
      }
    }, SEARCH_DEBOUNCE_MS)

    return () => clearTimeout(timer)
  }, [projectsSearchInput, projectsSearch])

  // Populate form when namespace data is loaded
  useEffect(() => {
    if (namespaceData?.namespace) {
      setNamespaceCode(namespaceData.namespace.namespaceCode)
      setName(namespaceData.namespace.name)
      setIsModified(false)
    }
  }, [namespaceData])

  const handleNamespaceCodeChange = (value: string) => {
    setNamespaceCode(value)
    setIsModified(true)
    if (namespaceCodeError) {
      setNamespaceCodeError(validateCode(value, 'Namespace code'))
    }
  }

  const handleNamespaceCodeBlur = () => {
    if (!isEditing) {
      setNamespaceCodeError(validateCode(namespaceCode, 'Namespace code'))
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (isReadOnly) return

    // Validate namespace code for new namespaces
    if (!isEditing) {
      const codeError = validateCode(namespaceCode, 'Namespace code')
      if (codeError) {
        setNamespaceCodeError(codeError)
        return
      }
    }

    if (!name.trim()) {
      setError('Name is required')
      return
    }

    setError('')
    setNamespaceCodeError('')
    setSuccess('')

    try {
      if (isEditing) {
        await updateNamespace({
          variables: {
            namespaceCode: id!,
            input: { name },
          },
        })
        setSuccess('Namespace updated successfully')
        setIsModified(false)
      } else {
        await createNamespace({
          variables: {
            input: {
              namespaceCode,
              name,
            },
          },
        })
        navigate(`/admin/namespaces/${encodeURIComponent(namespaceCode)}`)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred')
    }
  }

  const handleCancel = () => {
    navigate('/admin/namespaces')
  }

  const handleDeleteClick = () => {
    setDeleteConfirm(true)
  }

  const handleDeleteConfirm = async () => {
    if (!isEditing) return
    setError('')
    try {
      await deleteNamespace({
        variables: { namespaceCode: id! },
      })
      setDeleteConfirm(false)
      navigate('/admin/namespaces')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete namespace')
      setDeleteConfirm(false)
    }
  }

  const handleDeleteCancel = () => {
    setDeleteConfirm(false)
  }

  // Project handlers
  const handleViewProject = (projectCode: string) => {
    navigate(`/admin/projects/${encodeURIComponent(id!)}/${encodeURIComponent(projectCode)}`)
  }

  const handleClearProjectsSearch = () => {
    setProjectsSearchInput('')
    setProjectsSearch('')
    setProjectsPage(0)
  }

  const handleProjectsPageChange = (newPage: number) => {
    setProjectsPage(newPage)
  }

  const handleDeleteProjectClick = (projectCode: string) => {
    setDeleteProjectConfirm({ namespaceCode: id!, projectCode })
  }

  const handleDeleteProjectConfirm = async () => {
    if (!deleteProjectConfirm) return
    try {
      await deleteProject({
        variables: {
          namespaceCode: deleteProjectConfirm.namespaceCode,
          projectCode: deleteProjectConfirm.projectCode,
        },
      })
      setDeleteProjectConfirm(null)
      refetchProjects()
    } catch (err) {
      console.error('Failed to delete project:', err)
      setDeleteProjectConfirm(null)
    }
  }

  const handleDeleteProjectCancel = () => {
    setDeleteProjectConfirm(null)
  }

  if (isEditing && namespaceLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-purple border-t-transparent"></div>
      </div>
    )
  }

  if (isEditing && !namespaceData?.namespace) {
    return (
      <div className="rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4">
        <p className="text-red-700 dark:text-red-400">Namespace not found</p>
      </div>
    )
  }

  const getPageTitle = () => {
    if (!isEditing) return 'Create Namespace'
    if (isReadOnly) return 'View Namespace'
    return 'Edit Namespace'
  }

  const getPageDescription = () => {
    if (!isEditing) return 'Add a new namespace to organize projects'
    if (isReadOnly) return 'View namespace information'
    return 'Update namespace information'
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <button
          onClick={handleCancel}
          className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400 hover:text-brand-purple dark:hover:text-brand-purple mb-4"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 19l-7-7 7-7" />
          </svg>
          Back to Namespaces
        </button>
        <div className="flex items-center gap-3">
          <h2 className="text-2xl font-bold text-slate-900 dark:text-white">
            {getPageTitle()}
          </h2>
          {isReadOnly && (
            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-400">
              Read Only
            </span>
          )}
        </div>
        <p className="mt-1 text-slate-600 dark:text-slate-400">
          {getPageDescription()}
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Form */}
        <div className="lg:col-span-2">
          <form onSubmit={handleSubmit}>
            <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                  Namespace Information
                </h3>
                <UnsavedChangesIndicator show={isModified && canWrite} />
              </div>

              {error && (
                <div className="mb-4 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-3">
                  <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
                </div>
              )}

              {success && (
                <div className="mb-4 rounded-lg bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 p-3">
                  <p className="text-sm text-green-600 dark:text-green-400">{success}</p>
                </div>
              )}

              {/* Namespace Code */}
              <div className="mb-4">
                <label htmlFor="namespaceCode" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Namespace Code
                </label>
                <input
                  id="namespaceCode"
                  type="text"
                  value={namespaceCode}
                  onChange={(e) => handleNamespaceCodeChange(e.target.value)}
                  onBlur={handleNamespaceCodeBlur}
                  disabled={isEditing || isReadOnly}
                  required
                  className={`w-full rounded-lg border bg-white dark:bg-slate-900 py-2 px-3 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 ${
                    namespaceCodeError
                      ? 'border-red-300 dark:border-red-700 focus:border-red-500 focus:ring-red-500/20'
                      : 'border-slate-200 dark:border-slate-700 focus:border-brand-purple focus:ring-brand-purple/20'
                  } ${
                    isEditing || isReadOnly ? 'opacity-50 cursor-not-allowed bg-slate-50 dark:bg-slate-800' : ''
                  }`}
                  placeholder="Enter namespace code (letters, numbers, _ and - only)"
                />
                {namespaceCodeError && (
                  <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                    {namespaceCodeError}
                  </p>
                )}
                {isEditing && !isReadOnly && !namespaceCodeError && (
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Namespace code cannot be changed
                  </p>
                )}
                {!isEditing && !namespaceCodeError && (
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Only letters, numbers, underscores and hyphens allowed
                  </p>
                )}
              </div>

              {/* Name */}
              <div className="mb-6">
                <label htmlFor="name" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Name
                </label>
                <input
                  id="name"
                  type="text"
                  value={name}
                  onChange={(e) => {
                    setName(e.target.value)
                    setIsModified(true)
                  }}
                  disabled={isReadOnly}
                  required
                  className={`w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-2 px-3 text-slate-900 dark:text-white placeholder-slate-400 focus:border-brand-purple focus:outline-none focus:ring-2 focus:ring-brand-purple/20 ${
                    isReadOnly ? 'opacity-50 cursor-not-allowed bg-slate-50 dark:bg-slate-800' : ''
                  }`}
                  placeholder="Enter namespace name"
                />
              </div>

              {/* Form Actions */}
              <div className="flex gap-3 pt-4 border-t border-slate-200 dark:border-slate-700">
                <button
                  type="button"
                  onClick={handleCancel}
                  disabled={isLoading}
                  className="px-4 py-2 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors disabled:opacity-50"
                >
                  {isReadOnly ? 'Back' : 'Cancel'}
                </button>
                {!isReadOnly && (
                  <button
                    type="submit"
                    disabled={isLoading || (isEditing && !isModified)}
                    className="px-4 py-2 text-sm font-medium rounded-lg bg-gradient-to-r from-brand-purple to-brand-indigo text-white hover:opacity-90 transition-opacity disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {isLoading ? (
                      <span className="flex items-center gap-2">
                        <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                        </svg>
                        Saving...
                      </span>
                    ) : isEditing ? (
                      'Save Changes'
                    ) : (
                      'Create Namespace'
                    )}
                  </button>
                )}
              </div>
            </div>
          </form>
        </div>

        {/* Sidebar - Status & Actions (for editing/viewing) */}
        {isEditing && namespaceData?.namespace && (
          <div className="space-y-6">
            {/* Status */}
            <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
              <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
                Status
              </h3>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Projects</span>
                  <span className="text-sm font-medium text-slate-900 dark:text-white">
                    {namespaceData.namespace.projects?.length || 0}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Created</span>
                  <RelativeTime
                    date={namespaceData.namespace.createdAt}
                    className="text-sm text-slate-900 dark:text-white cursor-help"
                  />
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Last Updated</span>
                  <RelativeTime
                    date={namespaceData.namespace.updatedAt}
                    className="text-sm text-slate-900 dark:text-white cursor-help"
                  />
                </div>
              </div>
            </div>

            {/* Actions */}
            {canWrite && (
              <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
                <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
                  Actions
                </h3>
                <div className="space-y-3">
                  <button
                    onClick={handleDeleteClick}
                    disabled={isLoading}
                    className="w-full flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium rounded-lg border border-red-200 dark:border-red-800 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors disabled:opacity-50"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                    Delete Namespace
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Projects Table (only when editing) */}
      {isEditing && (
        <div className="mt-6">
          <div className="mb-4 flex items-center justify-between">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
              Projects in this Namespace
            </h3>
            {canWriteProjects && (
              <Link
                to={`/admin/projects/new?namespace=${encodeURIComponent(id!)}`}
                className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-lg bg-gradient-to-r from-brand-purple to-brand-indigo text-white hover:opacity-90 transition-opacity"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                New Project
              </Link>
            )}
          </div>

          {/* Search */}
          <div className="mb-4">
            <div className="relative">
              <svg
                className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1.5}
                  d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                />
              </svg>
              <input
                type="text"
                placeholder="Search projects by code or name..."
                value={projectsSearchInput}
                onChange={(e) => setProjectsSearchInput(e.target.value)}
                className="w-full pl-10 pr-4 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-brand-purple focus:border-transparent"
              />
              {projectsSearchInput && (
                <button
                  onClick={handleClearProjectsSearch}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
                >
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              )}
            </div>
          </div>

          {/* Projects Content */}
          {projectsLoading ? (
            <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-8">
              <div className="flex items-center justify-center">
                <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-purple border-t-transparent"></div>
              </div>
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
                {projectsSearch ? `No projects found for "${projectsSearch}"` : 'No projects in this namespace'}
              </p>
              {canWriteProjects && !projectsSearch && (
                <Link
                  to={`/admin/projects/new?namespace=${encodeURIComponent(id!)}`}
                  className="mt-4 inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg bg-gradient-to-r from-brand-purple to-brand-indigo text-white hover:opacity-90 transition-opacity"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                  </svg>
                  Create first project
                </Link>
              )}
            </div>
          ) : (
            <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 overflow-hidden">
              <table className="w-full">
                <thead className="bg-slate-50 dark:bg-slate-700/50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                      Code
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                      Name
                    </th>
                    <th className="px-6 py-3 text-right text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                  {projects.map((project) => (
                    <tr
                      key={project.projectCode}
                      className="hover:bg-slate-50 dark:hover:bg-slate-700/30"
                    >
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="font-mono text-sm font-medium text-slate-900 dark:text-white">
                          {project.projectCode}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="text-slate-600 dark:text-slate-400">{project.name}</span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right">
                        <div className="flex items-center justify-end gap-2">
                          <button
                            onClick={() => handleViewProject(project.projectCode)}
                            className="p-2 rounded-lg transition-colors text-slate-600 hover:text-brand-purple hover:bg-brand-purple/10 dark:text-slate-400 dark:hover:text-brand-purple"
                            title={canWriteProjects ? 'Edit project' : 'View project'}
                          >
                            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              {canWriteProjects ? (
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
                          {canWriteProjects && (
                            <button
                              onClick={() => handleDeleteProjectClick(project.projectCode)}
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
              {projectsTotalPages > 1 && (
                <div className="px-6 py-4 border-t border-slate-200 dark:border-slate-700 flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">
                    Page {projectsPage + 1} of {projectsTotalPages} ({projectsTotal} projects)
                  </span>
                  <div className="flex gap-2">
                    <button
                      onClick={() => handleProjectsPageChange(Math.max(0, projectsPage - 1))}
                      disabled={projectsPage === 0}
                      className="px-3 py-1.5 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Previous
                    </button>
                    <button
                      onClick={() => handleProjectsPageChange(Math.min(projectsTotalPages - 1, projectsPage + 1))}
                      disabled={projectsPage >= projectsTotalPages - 1}
                      className="px-3 py-1.5 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Next
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Delete Project Confirmation Modal */}
      {deleteProjectConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="absolute inset-0 bg-black/50 backdrop-blur-sm"
            onClick={handleDeleteProjectCancel}
          />
          <div className="relative w-full max-w-md mx-4 rounded-xl bg-white dark:bg-slate-800 shadow-xl border border-slate-200 dark:border-slate-700 p-6">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
              Delete Project
            </h3>
            <p className="text-slate-600 dark:text-slate-400 mb-6">
              Are you sure you want to delete project <strong>{deleteProjectConfirm.projectCode}</strong>? This action cannot be undone.
            </p>
            <div className="flex gap-3 justify-end">
              <button
                onClick={handleDeleteProjectCancel}
                className="px-4 py-2 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleDeleteProjectConfirm}
                disabled={deleteProjectLoading}
                className="px-4 py-2 text-sm font-medium rounded-lg bg-red-600 text-white hover:bg-red-700 transition-colors disabled:opacity-50"
              >
                {deleteProjectLoading ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
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
              Delete Namespace
            </h3>
            <p className="text-slate-600 dark:text-slate-400 mb-6">
              Are you sure you want to delete namespace <strong>{namespaceCode}</strong>? This will also delete all projects in this namespace. This action cannot be undone.
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