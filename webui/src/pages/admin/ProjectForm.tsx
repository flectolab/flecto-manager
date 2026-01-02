import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useQuery, useMutation } from '@apollo/client/react'
import { GetProjectDocument, CreateProjectDocument, UpdateProjectDocument, DeleteProjectDocument, GetNamespacesDocument } from '../../generated/graphql'
import { usePermissions, AdminSection, Action, validateCode } from '../../hooks/usePermissions'
import { useDocumentTitle } from '../../hooks/useDocumentTitle'
import { UnsavedChangesIndicator } from '../../components/UnsavedChangesIndicator'
import { RelativeTime } from '../../components/RelativeTime'
import { formatSize } from '../../utils/format'

export function ProjectForm() {
  const { namespaceCode: paramNamespaceCode, projectCode: paramProjectCode } = useParams<{ namespaceCode: string; projectCode: string }>()
  const navigate = useNavigate()
  const { canAdminResource } = usePermissions()

  const isEditing = !!paramNamespaceCode && !!paramProjectCode
  useDocumentTitle(isEditing ? `Admin - Edit Project` : 'Admin - Add Project')
  const canWrite = canAdminResource(AdminSection.Projects, Action.Write)
  const isReadOnly = isEditing && !canWrite

  const [namespaceCode, setNamespaceCode] = useState('')
  const [projectCode, setProjectCode] = useState('')
  const [projectCodeError, setProjectCodeError] = useState('')
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState(false)
  const [isModified, setIsModified] = useState(false)

  // Fetch project data if editing
  const { data: projectData, loading: projectLoading } = useQuery(GetProjectDocument, {
    variables: { namespaceCode: paramNamespaceCode!, projectCode: paramProjectCode! },
    skip: !isEditing,
  })

  // Fetch namespaces for dropdown (only in create mode)
  const { data: namespacesData, loading: namespacesLoading } = useQuery(GetNamespacesDocument, {
    skip: isEditing,
  })

  const [createProject, { loading: createLoading }] = useMutation(CreateProjectDocument)
  const [updateProject, { loading: updateLoading }] = useMutation(UpdateProjectDocument)
  const [deleteProject, { loading: deleteLoading }] = useMutation(DeleteProjectDocument)

  const isLoading = createLoading || updateLoading || deleteLoading

  // Populate form when project data is loaded
  useEffect(() => {
    if (projectData?.project) {
      setNamespaceCode(projectData.project.namespace.namespaceCode)
      setProjectCode(projectData.project.projectCode)
      setName(projectData.project.name)
      setIsModified(false)
    }
  }, [projectData])

  const handleProjectCodeChange = (value: string) => {
    setProjectCode(value)
    setIsModified(true)
    if (projectCodeError) {
      setProjectCodeError(validateCode(value, 'Project code'))
    }
  }

  const handleProjectCodeBlur = () => {
    if (!isEditing) {
      setProjectCodeError(validateCode(projectCode, 'Project code'))
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (isReadOnly) return

    // Validate project code for new projects
    if (!isEditing) {
      const codeError = validateCode(projectCode, 'Project code')
      if (codeError) {
        setProjectCodeError(codeError)
        return
      }

      if (!namespaceCode) {
        setError('Please select a namespace')
        return
      }
    }

    if (!name.trim()) {
      setError('Name is required')
      return
    }

    setError('')
    setProjectCodeError('')
    setSuccess('')

    try {
      if (isEditing) {
        await updateProject({
          variables: {
            namespaceCode: paramNamespaceCode!,
            projectCode: paramProjectCode!,
            input: { name },
          },
        })
        setSuccess('Project updated successfully')
        setIsModified(false)
      } else {
        await createProject({
          variables: {
            namespaceCode,
            input: {
              projectCode,
              name,
            },
          },
        })
        navigate(`/admin/projects/${encodeURIComponent(namespaceCode)}/${encodeURIComponent(projectCode)}`)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred')
    }
  }

  const handleCancel = () => {
    navigate('/admin/projects')
  }

  const handleDeleteClick = () => {
    setDeleteConfirm(true)
  }

  const handleDeleteConfirm = async () => {
    if (!isEditing) return
    setError('')
    try {
      await deleteProject({
        variables: {
          namespaceCode: paramNamespaceCode!,
          projectCode: paramProjectCode!,
        },
      })
      setDeleteConfirm(false)
      navigate('/admin/projects')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete project')
      setDeleteConfirm(false)
    }
  }

  const handleDeleteCancel = () => {
    setDeleteConfirm(false)
  }

  if (isEditing && projectLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-purple border-t-transparent"></div>
      </div>
    )
  }

  if (isEditing && !projectData?.project) {
    return (
      <div className="rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4">
        <p className="text-red-700 dark:text-red-400">Project not found</p>
      </div>
    )
  }

  const getPageTitle = () => {
    if (!isEditing) return 'Create Project'
    if (isReadOnly) return 'View Project'
    return 'Edit Project'
  }

  const getPageDescription = () => {
    if (!isEditing) return 'Add a new project to a namespace'
    if (isReadOnly) return 'View project information'
    return 'Update project information'
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
          Back to Projects
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
                  Project Information
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

              {/* Namespace Dropdown (only for creation) */}
              <div className="mb-4">
                <label htmlFor="namespace" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Namespace
                </label>
                {isEditing ? (
                  <div className="flex items-center gap-2">
                    <input
                      type="text"
                      value={projectData?.project?.namespace.name || namespaceCode}
                      disabled
                      className="flex-1 rounded-lg border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 py-2 px-3 text-slate-900 dark:text-white opacity-50 cursor-not-allowed"
                    />
                  </div>
                ) : (
                  <>
                    <select
                      id="namespace"
                      value={namespaceCode}
                      onChange={(e) => {
                        setNamespaceCode(e.target.value)
                        setIsModified(true)
                      }}
                      required
                      disabled={namespacesLoading}
                      className="w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-2 px-3 text-slate-900 dark:text-white focus:border-brand-purple focus:outline-none focus:ring-2 focus:ring-brand-purple/20"
                    >
                      <option value="">Select a namespace...</option>
                      {namespacesData?.namespaces.map((ns) => (
                        <option key={ns.namespaceCode} value={ns.namespaceCode}>
                          {ns.name} ({ns.namespaceCode})
                        </option>
                      ))}
                    </select>
                    {namespacesLoading && (
                      <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">Loading namespaces...</p>
                    )}
                  </>
                )}
                {isEditing && !isReadOnly && (
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Namespace cannot be changed
                  </p>
                )}
              </div>

              {/* Project Code */}
              <div className="mb-4">
                <label htmlFor="projectCode" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Project Code
                </label>
                <input
                  id="projectCode"
                  type="text"
                  value={projectCode}
                  onChange={(e) => handleProjectCodeChange(e.target.value)}
                  onBlur={handleProjectCodeBlur}
                  disabled={isEditing || isReadOnly}
                  required
                  className={`w-full rounded-lg border bg-white dark:bg-slate-900 py-2 px-3 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 ${
                    projectCodeError
                      ? 'border-red-300 dark:border-red-700 focus:border-red-500 focus:ring-red-500/20'
                      : 'border-slate-200 dark:border-slate-700 focus:border-brand-purple focus:ring-brand-purple/20'
                  } ${
                    isEditing || isReadOnly ? 'opacity-50 cursor-not-allowed bg-slate-50 dark:bg-slate-800' : ''
                  }`}
                  placeholder="Enter project code (letters, numbers, _ and - only)"
                />
                {projectCodeError && (
                  <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                    {projectCodeError}
                  </p>
                )}
                {isEditing && !isReadOnly && !projectCodeError && (
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Project code cannot be changed
                  </p>
                )}
                {!isEditing && !projectCodeError && (
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
                  placeholder="Enter project name"
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
                      'Create Project'
                    )}
                  </button>
                )}
              </div>
            </div>
          </form>
        </div>

        {/* Sidebar - Status & Actions (for editing/viewing) */}
        {isEditing && projectData?.project && (
          <div className="space-y-6">
            {/* Status */}
            <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
              <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
                Status
              </h3>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Namespace</span>
                  <Link
                    to={`/admin/namespaces/${encodeURIComponent(projectData.project.namespace.namespaceCode)}`}
                    className="text-sm font-medium text-brand-purple hover:text-brand-indigo transition-colors"
                  >
                    {projectData.project.namespace.name}
                  </Link>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Version</span>
                  <span className="text-sm font-medium text-slate-900 dark:text-white">
                    {projectData.project.version}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Total redirects</span>
                  <span className="text-sm font-medium text-slate-900 dark:text-white">
                    {projectData.project.countRedirects}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Redirect drafts</span>
                  <span className="text-sm font-medium text-slate-900 dark:text-white">
                    {projectData.project.countRedirectDrafts}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Total pages</span>
                  <span className="text-sm font-medium text-slate-900 dark:text-white">
                    {projectData.project.countPages}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Page drafts</span>
                  <span className="text-sm font-medium text-slate-900 dark:text-white">
                    {projectData.project.countPageDrafts}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Page content size</span>
                  <span className="text-sm font-medium text-slate-900 dark:text-white">
                    {formatSize(Number(projectData.project.totalPageContentSize))} / {formatSize(Number(projectData.project.totalPageContentSizeLimit))}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Created</span>
                  <RelativeTime
                    date={projectData.project.createdAt}
                    className="text-sm text-slate-900 dark:text-white cursor-help"
                  />
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Last Updated</span>
                  <RelativeTime
                    date={projectData.project.updatedAt}
                    className="text-sm text-slate-900 dark:text-white cursor-help"
                  />
                </div>
                {projectData.project.publishedAt && !projectData.project.publishedAt.startsWith('0001-') && (
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-slate-600 dark:text-slate-400">Published</span>
                    <RelativeTime
                      date={projectData.project.publishedAt}
                      className="text-sm text-slate-900 dark:text-white cursor-help"
                    />
                  </div>
                )}
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
                    Delete Project
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>

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
              Are you sure you want to delete project <strong>{projectCode}</strong> from namespace <strong>{namespaceCode}</strong>? This action cannot be undone.
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
