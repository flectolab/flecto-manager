import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation } from '@apollo/client/react'
import { GetRoleDocument, CreateRoleDocument, UpdateRoleDocument, DeleteRoleDocument, GetNamespacesDocument } from '../../generated/graphql'
import { usePermissions, AdminSection, Action, validateCode } from '../../hooks/usePermissions'
import { UnsavedChangesIndicator } from '../../components/UnsavedChangesIndicator'
import { ResourcePermissionsEditor, AdminPermissionsEditor } from '../../components/admin'
import type { ResourcePermission, AdminPermission } from '../../components/admin'

export function RoleForm() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { canAdminResource } = usePermissions()

  const isEditing = !!id && id !== 'new'
  const canWrite = canAdminResource(AdminSection.Roles, Action.Write)
  const canDelete = canAdminResource(AdminSection.Roles, Action.Delete)
  const isReadOnly = isEditing && !canWrite

  const [roleName, setRoleName] = useState('')
  const [roleNameError, setRoleNameError] = useState('')
  const [resourcePermissions, setResourcePermissions] = useState<ResourcePermission[]>([])
  const [adminPermissions, setAdminPermissions] = useState<AdminPermission[]>([])
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState(false)
  const [isModified, setIsModified] = useState(false)

  // Fetch the role we're editing
  const { data: roleData, loading: roleLoading, refetch } = useQuery(GetRoleDocument, {
    variables: { name: decodeURIComponent(id!) },
    skip: !isEditing,
  })

  // Fetch namespaces for dropdown
  const { data: namespacesData } = useQuery(GetNamespacesDocument)

  const [createRole, { loading: createLoading }] = useMutation(CreateRoleDocument)
  const [updateRole, { loading: updateLoading }] = useMutation(UpdateRoleDocument)
  const [deleteRole, { loading: deleteLoading }] = useMutation(DeleteRoleDocument)

  const isLoading = createLoading || updateLoading || deleteLoading

  // Populate form when role data is loaded
  useEffect(() => {
    if (roleData?.role) {
      setRoleName(roleData.role.name)
      setResourcePermissions(roleData.role.resources.map(r => ({
        namespace: r.namespace,
        project: r.project,
        action: r.action,
      })))
      setAdminPermissions(roleData.role.admin.map(a => ({
        section: a.section,
        action: a.action,
      })))
      setIsModified(false)
    }
  }, [roleData])

  // Get namespaces list with * option
  const namespaceOptions = [
    { code: '*', label: '* (All)' },
    ...(namespacesData?.namespaces.map(n => ({ code: n.namespaceCode, label: `${n.namespaceCode} - ${n.name}` })) || [])
  ]

  // Get projects for a given namespace
  const getProjectOptions = (namespaceCode: string): { code: string; label: string }[] => {
    if (namespaceCode === '*') return [{ code: '*', label: '* (All)' }]
    const namespace = namespacesData?.namespaces.find(n => n.namespaceCode === namespaceCode)
    return [
      { code: '*', label: '* (All)' },
      ...(namespace?.projects.map(p => ({ code: p.projectCode, label: `${p.projectCode} - ${p.name}` })) || [])
    ]
  }

  // Resource permission handlers
  const handleAddResourcePermission = () => {
    setResourcePermissions([
      ...resourcePermissions,
      { namespace: '*', project: '*', action: 'read' }
    ])
    setIsModified(true)
  }

  const handleRemoveResourcePermission = (index: number) => {
    setResourcePermissions(resourcePermissions.filter((_, i) => i !== index))
    setIsModified(true)
  }

  const handleResourcePermissionChange = (index: number, field: keyof ResourcePermission, value: string) => {
    const updated = [...resourcePermissions]
    const perm = { ...updated[index] }

    if (field === 'namespace') {
      perm.namespace = value
      if (value === '*') {
        perm.project = '*'
      }
    } else if (field === 'project') {
      perm.project = value
    } else if (field === 'action') {
      perm.action = value
    }

    updated[index] = perm
    setResourcePermissions(updated)
    setIsModified(true)
  }

  // Admin permission handlers
  const handleAddAdminPermission = () => {
    setAdminPermissions([
      ...adminPermissions,
      { section: '*', action: 'read' }
    ])
    setIsModified(true)
  }

  const handleRemoveAdminPermission = (index: number) => {
    setAdminPermissions(adminPermissions.filter((_, i) => i !== index))
    setIsModified(true)
  }

  const handleAdminPermissionChange = (index: number, field: 'section' | 'action', value: string) => {
    const updated = [...adminPermissions]
    updated[index] = { ...updated[index], [field]: value }
    setAdminPermissions(updated)
    setIsModified(true)
  }

  const handleRoleNameChange = (value: string) => {
    setRoleName(value)
    setIsModified(true)
    // Clear error when user starts typing, validate on blur or submit
    if (roleNameError) {
      setRoleNameError(validateCode(value, 'Role name'))
    }
  }

  const handleRoleNameBlur = () => {
    if (!isEditing) {
      setRoleNameError(validateCode(roleName, 'Role name'))
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (isReadOnly) return

    // Validate role name
    const nameError = validateCode(roleName, 'Role name')
    if (nameError) {
      setRoleNameError(nameError)
      return
    }

    setError('')
    setRoleNameError('')
    setSuccess('')

    try {
      if (isEditing) {
        await updateRole({
          variables: {
            name: roleName.trim(),
            input: {
              resourcePermissions: resourcePermissions,
              adminPermissions: adminPermissions,
            },
          },
        })
        setSuccess('Role updated successfully')
        refetch()
      } else {
        await createRole({
          variables: {
            input: {
              name: roleName.trim(),
              resourcePermissions: resourcePermissions,
              adminPermissions: adminPermissions,
            },
          },
        })
        setSuccess('Role created successfully')
        navigate(`/admin/roles/${encodeURIComponent(roleName.trim())}`, { replace: true })
      }

      setIsModified(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred')
    }
  }

  const handleCancel = () => {
    navigate('/admin/roles')
  }

  const handleDeleteClick = () => {
    setDeleteConfirm(true)
  }

  const handleDeleteConfirm = async () => {
    if (!roleData?.role) return
    setError('')
    try {
      await deleteRole({
        variables: { name: roleData.role.name },
      })
      setDeleteConfirm(false)
      navigate('/admin/roles')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete role')
      setDeleteConfirm(false)
    }
  }

  const handleDeleteCancel = () => {
    setDeleteConfirm(false)
  }

  if (isEditing && roleLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-purple border-t-transparent"></div>
      </div>
    )
  }

  if (isEditing && !roleData?.role) {
    return (
      <div className="rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4">
        <p className="text-red-700 dark:text-red-400">Role not found</p>
      </div>
    )
  }

  const getPageTitle = () => {
    if (!isEditing) return 'Create Role'
    if (isReadOnly) return 'View Role'
    return 'Edit Role'
  }

  const getPageDescription = () => {
    if (!isEditing) return 'Add a new role with permissions'
    if (isReadOnly) return 'View role configuration'
    return 'Update role permissions'
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
          Back to Roles
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
          <UnsavedChangesIndicator show={isModified && canWrite} />
        </div>
        <p className="mt-1 text-slate-600 dark:text-slate-400">
          {getPageDescription()}
        </p>
      </div>

      <form onSubmit={handleSubmit}>
        <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
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

          {/* Role Name */}
          <div className="mb-6">
            <label htmlFor="roleName" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              Role Name
            </label>
            <input
              id="roleName"
              type="text"
              value={roleName}
              onChange={(e) => handleRoleNameChange(e.target.value)}
              onBlur={handleRoleNameBlur}
              disabled={isEditing || isReadOnly}
              required
              className={`w-full rounded-lg border bg-white dark:bg-slate-900 py-2 px-3 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 ${
                roleNameError
                  ? 'border-red-300 dark:border-red-700 focus:border-red-500 focus:ring-red-500/20'
                  : 'border-slate-200 dark:border-slate-700 focus:border-brand-purple focus:ring-brand-purple/20'
              } ${
                isEditing || isReadOnly ? 'opacity-50 cursor-not-allowed bg-slate-50 dark:bg-slate-800' : ''
              }`}
              placeholder="Enter role name (letters, numbers, _ and - only)"
            />
            {roleNameError && (
              <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                {roleNameError}
              </p>
            )}
            {isEditing && !isReadOnly && !roleNameError && (
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                Role name cannot be changed
              </p>
            )}
            {!isEditing && !roleNameError && (
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                Only letters, numbers, underscores and hyphens allowed
              </p>
            )}
          </div>

          {/* Resource Permissions */}
          <div className="mb-6">
            <ResourcePermissionsEditor
              permissions={resourcePermissions}
              namespaceOptions={namespaceOptions}
              getProjectOptions={getProjectOptions}
              onChange={handleResourcePermissionChange}
              onAdd={handleAddResourcePermission}
              onRemove={handleRemoveResourcePermission}
              readOnly={isReadOnly}
              canWrite={canWrite}
            />
          </div>

          {/* Admin Permissions */}
          <div className="mb-6">
            <AdminPermissionsEditor
              permissions={adminPermissions}
              onChange={handleAdminPermissionChange}
              onAdd={handleAddAdminPermission}
              onRemove={handleRemoveAdminPermission}
              readOnly={isReadOnly}
              canWrite={canWrite}
            />
          </div>

          {/* Actions */}
          <div className="flex items-center justify-between pt-4 border-t border-slate-200 dark:border-slate-700">
            <div>
              {isEditing && canDelete && (
                <button
                  type="button"
                  onClick={handleDeleteClick}
                  disabled={isLoading}
                  className="flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-lg text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors disabled:opacity-50"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                  Delete
                </button>
              )}
            </div>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={handleCancel}
                disabled={isLoading}
                className="px-4 py-2 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              {!isReadOnly && (
                <button
                  type="submit"
                  disabled={isLoading || !isModified}
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
                    'Create Role'
                  )}
                </button>
              )}
            </div>
          </div>
        </div>
      </form>

      {/* Delete Confirmation Modal */}
      {deleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="absolute inset-0 bg-black/50 backdrop-blur-sm"
            onClick={handleDeleteCancel}
          />
          <div className="relative w-full max-w-md mx-4 rounded-xl bg-white dark:bg-slate-800 shadow-xl border border-slate-200 dark:border-slate-700 p-6">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
              Delete Role
            </h3>
            <p className="text-slate-600 dark:text-slate-400 mb-6">
              Are you sure you want to delete role <strong>{roleName}</strong>? This will also remove this role from all users. This action cannot be undone.
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
