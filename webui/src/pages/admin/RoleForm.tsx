import { useState, useEffect, useRef } from 'react'
import { useParams, useNavigate, useSearchParams } from 'react-router-dom'
import { useQuery, useMutation, useLazyQuery } from '@apollo/client/react'
import {
  GetRoleDocument,
  CreateRoleDocument,
  UpdateRoleDocument,
  DeleteRoleDocument,
  GetNamespacesDocument,
  GetRoleUsersDocument,
  GetUsersNotInRoleDocument,
  AddUserToRoleDocument,
  RemoveUserFromRoleDocument,
} from '../../generated/graphql'
import { usePermissions, AdminSection, Action, validateCode } from '../../hooks/usePermissions'
import { useDocumentTitle } from '../../hooks/useDocumentTitle'
import { UnsavedChangesIndicator } from '../../components/UnsavedChangesIndicator'
import { RelativeTime } from '../../components/RelativeTime'
import { ResourcePermissionsEditor, AdminPermissionsEditor } from '../../components/admin'
import type { ResourcePermission, AdminPermission } from '../../components/admin'

const PAGE_SIZE = 10

export function RoleForm() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const { canAdminResource } = usePermissions()

  const isEditing = !!id && id !== 'new'
  useDocumentTitle(isEditing ? `Admin - Edit Role` : 'Admin - Add Role')
  const canWrite = canAdminResource(AdminSection.Roles, Action.Write)
  const isReadOnly = isEditing && !canWrite

  const [roleCode, setRoleCode] = useState('')
  const [roleCodeError, setRoleCodeError] = useState('')
  const [resourcePermissions, setResourcePermissions] = useState<ResourcePermission[]>([])
  const [adminPermissions, setAdminPermissions] = useState<AdminPermission[]>([])
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState(false)
  const [isModified, setIsModified] = useState(false)

  // Role Users state
  const [userSearch, setUserSearch] = useState(searchParams.get('userSearch') || '')
  const [userSearchInput, setUserSearchInput] = useState(searchParams.get('userSearch') || '')
  const currentPage = parseInt(searchParams.get('userPage') || '1', 10)
  const [removeUserConfirm, setRemoveUserConfirm] = useState<{ id: number; username: string } | null>(null)

  // Autocomplete state
  const [addUserInput, setAddUserInput] = useState('')
  const [showSuggestions, setShowSuggestions] = useState(false)
  const autocompleteRef = useRef<HTMLDivElement>(null)

  // Fetch the role we're editing
  const { data: roleData, loading: roleLoading, refetch } = useQuery(GetRoleDocument, {
    variables: { code: decodeURIComponent(id!) },
    skip: !isEditing,
  })

  // Fetch namespaces for dropdown
  const { data: namespacesData } = useQuery(GetNamespacesDocument)

  const [createRole, { loading: createLoading }] = useMutation(CreateRoleDocument)
  const [updateRole, { loading: updateLoading }] = useMutation(UpdateRoleDocument)
  const [deleteRole, { loading: deleteLoading }] = useMutation(DeleteRoleDocument)
  const [addUserToRole, { loading: addUserLoading }] = useMutation(AddUserToRoleDocument)
  const [removeUserFromRole, { loading: removeUserLoading }] = useMutation(RemoveUserFromRoleDocument)

  // Fetch role users (paginated)
  const { data: roleUsersData, loading: roleUsersLoading, refetch: refetchRoleUsers } = useQuery(GetRoleUsersDocument, {
    variables: {
      code: decodeURIComponent(id!),
      pagination: { limit: PAGE_SIZE, offset: (currentPage - 1) * PAGE_SIZE },
      filter: userSearch ? { search: userSearch } : {},
    },
    skip: !isEditing,
  })

  // Lazy query for autocomplete
  const [searchUsersNotInRole, { data: usersNotInRoleData, loading: searchingUsers }] = useLazyQuery(GetUsersNotInRoleDocument)

  const isLoading = createLoading || updateLoading || deleteLoading || addUserLoading || removeUserLoading

  // Populate form when role data is loaded
  useEffect(() => {
    if (roleData?.role) {
      setRoleCode(roleData.role.code)
      setResourcePermissions(roleData.role.resources.map(r => ({
        namespace: r.namespace,
        project: r.project,
        resource: r.resource,
        action: r.action,
      })))
      setAdminPermissions(roleData.role.admin.map(a => ({
        section: a.section,
        action: a.action,
      })))
      setIsModified(false)
    }
  }, [roleData])

  // Handle click outside autocomplete
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (autocompleteRef.current && !autocompleteRef.current.contains(event.target as Node)) {
        setShowSuggestions(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Trigger autocomplete search when input changes (min 3 chars)
  useEffect(() => {
    if (addUserInput.length >= 3 && isEditing) {
      searchUsersNotInRole({
        variables: {
          code: decodeURIComponent(id!),
          search: addUserInput,
          limit: 10,
        },
      })
      setShowSuggestions(true)
    } else {
      setShowSuggestions(false)
    }
  }, [addUserInput, isEditing, id, searchUsersNotInRole])

  // User search handlers
  const updateUserParams = (updates: Record<string, string | null>) => {
    const newParams = new URLSearchParams(searchParams)
    Object.entries(updates).forEach(([key, value]) => {
      if (value === null || value === '') {
        newParams.delete(key)
      } else {
        newParams.set(key, value)
      }
    })
    setSearchParams(newParams, { replace: true })
  }

  const handleUserSearch = () => {
    setUserSearch(userSearchInput)
    updateUserParams({ userSearch: userSearchInput || null, userPage: '1' })
  }

  const handleUserSearchClear = () => {
    setUserSearchInput('')
    setUserSearch('')
    updateUserParams({ userSearch: null, userPage: null })
  }

  const handleUserPageChange = (page: number) => {
    updateUserParams({ userPage: page > 1 ? String(page) : null })
  }

  const handleAddUser = async (userId: number) => {
    if (!canWrite) return
    try {
      await addUserToRole({
        variables: {
          roleCode: decodeURIComponent(id!),
          userId: String(userId),
        },
      })
      setAddUserInput('')
      setShowSuggestions(false)
      refetchRoleUsers()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add user to role')
    }
  }

  const handleRemoveUserClick = (userId: number, username: string) => {
    setRemoveUserConfirm({ id: userId, username })
  }

  const handleRemoveUserConfirm = async () => {
    if (!removeUserConfirm || !canWrite) return
    try {
      await removeUserFromRole({
        variables: {
          roleCode: decodeURIComponent(id!),
          userId: String(removeUserConfirm.id),
        },
      })
      setRemoveUserConfirm(null)
      refetchRoleUsers()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove user from role')
      setRemoveUserConfirm(null)
    }
  }

  const handleRefreshUsers = () => {
    refetchRoleUsers()
  }

  // Pagination calculations
  const totalUsers = roleUsersData?.roleUsers.total || 0
  const totalPages = Math.ceil(totalUsers / PAGE_SIZE)
  const roleUsers = roleUsersData?.roleUsers.items || []
  const suggestions = usersNotInRoleData?.usersNotInRole || []

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
      { namespace: '*', project: '*', resource: '*', action: 'read' }
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
    } else if (field === 'resource') {
      perm.resource = value
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

  const handleRoleCodeChange = (value: string) => {
    setRoleCode(value)
    setIsModified(true)
    // Clear error when user starts typing, validate on blur or submit
    if (roleCodeError) {
      setRoleCodeError(validateCode(value, 'Role name'))
    }
  }

  const handleRoleCodeBlur = () => {
    if (!isEditing) {
      setRoleCodeError(validateCode(roleCode, 'Role name'))
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (isReadOnly) return

    // Validate role name
    const nameError = validateCode(roleCode, 'Role name')
    if (nameError) {
      setRoleCodeError(nameError)
      return
    }

    setError('')
    setRoleCodeError('')
    setSuccess('')

    try {
      if (isEditing) {
        await updateRole({
          variables: {
            code: roleCode.trim(),
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
              code: roleCode.trim(),
              resourcePermissions: resourcePermissions,
              adminPermissions: adminPermissions,
            },
          },
        })
        setSuccess('Role created successfully')
        navigate(`/admin/roles/${encodeURIComponent(roleCode.trim())}`, { replace: true })
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
        variables: { code: roleData.role.code },
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

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Form */}
        <div className="lg:col-span-2">
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
                <label htmlFor="roleCode" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Role Name
                </label>
                <input
                  id="roleCode"
                  type="text"
                  value={roleCode}
                  onChange={(e) => handleRoleCodeChange(e.target.value)}
                  onBlur={handleRoleCodeBlur}
                  disabled={isEditing || isReadOnly}
                  required
                  className={`w-full rounded-lg border bg-white dark:bg-slate-900 py-2 px-3 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 ${
                    roleCodeError
                      ? 'border-red-300 dark:border-red-700 focus:border-red-500 focus:ring-red-500/20'
                      : 'border-slate-200 dark:border-slate-700 focus:border-brand-purple focus:ring-brand-purple/20'
                  } ${
                    isEditing || isReadOnly ? 'opacity-50 cursor-not-allowed bg-slate-50 dark:bg-slate-800' : ''
                  }`}
                  placeholder="Enter role name (letters, numbers, _ and - only)"
                />
                {roleCodeError && (
                  <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                    {roleCodeError}
                  </p>
                )}
                {isEditing && !isReadOnly && !roleCodeError && (
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Role name cannot be changed
                  </p>
                )}
                {!isEditing && !roleCodeError && (
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
          </form>
        </div>

        {/* Sidebar - Status & Actions (for editing) */}
        {isEditing && roleData?.role && (
          <div className="space-y-6">
            {/* Status */}
            <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
              <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
                Status
              </h3>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Created</span>
                  <RelativeTime
                    date={roleData.role.createdAt}
                    className="text-sm text-slate-900 dark:text-white cursor-help"
                  />
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Last Updated</span>
                  <RelativeTime
                    date={roleData.role.updatedAt}
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
                    type="button"
                    onClick={handleDeleteClick}
                    disabled={isLoading}
                    className="w-full flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium rounded-lg border border-red-200 dark:border-red-800 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors disabled:opacity-50"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                    Delete Role
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Role Users Section - Only show when editing */}
      {isEditing && (
        <div className="mt-8 rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
            Users with this Role
          </h3>

          {/* Add User Autocomplete */}
          {canWrite && (
            <div className="mb-4" ref={autocompleteRef}>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                Add User
              </label>
              <div className="relative">
                <input
                  type="text"
                  value={addUserInput}
                  onChange={(e) => setAddUserInput(e.target.value)}
                  placeholder="Type at least 3 characters to search users..."
                  className="w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-2 px-3 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:border-brand-purple focus:ring-brand-purple/20"
                />
                {searchingUsers && (
                  <div className="absolute right-3 top-1/2 -translate-y-1/2">
                    <svg className="animate-spin h-4 w-4 text-slate-400" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                    </svg>
                  </div>
                )}
                {showSuggestions && suggestions.length > 0 && (
                  <div className="absolute z-10 w-full mt-1 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg max-h-48 overflow-auto">
                    {suggestions.map((user) => (
                      <button
                        key={user.id}
                        type="button"
                        onClick={() => handleAddUser(Number(user.id))}
                        className="w-full px-3 py-2 text-left hover:bg-slate-100 dark:hover:bg-slate-700 text-sm text-slate-900 dark:text-white"
                      >
                        <span className="font-medium">{user.username}</span>
                        {(user.firstname || user.lastname) && (
                          <span className="text-slate-500 dark:text-slate-400 ml-2">
                            ({user.firstname} {user.lastname})
                          </span>
                        )}
                      </button>
                    ))}
                  </div>
                )}
                {showSuggestions && suggestions.length === 0 && addUserInput.length >= 3 && !searchingUsers && (
                  <div className="absolute z-10 w-full mt-1 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg p-3 text-sm text-slate-500 dark:text-slate-400">
                    No users found
                  </div>
                )}
              </div>
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                Search for users by username, first name, or last name
              </p>
            </div>
          )}

          {/* Search and Refresh */}
          <div className="mb-4 flex flex-col sm:flex-row gap-4">
            <div className="relative flex-1 flex gap-2">
              <button
                type="button"
                onClick={handleRefreshUsers}
                disabled={roleUsersLoading}
                className="px-3 py-2 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors disabled:opacity-50"
                title="Refresh"
              >
                <svg className={`w-5 h-5 ${roleUsersLoading ? 'animate-spin' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
              </button>
              <div className="relative flex-1">
                <svg
                  className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
                <input
                  type="text"
                  placeholder="Search users..."
                  value={userSearchInput}
                  onChange={(e) => setUserSearchInput(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleUserSearch()}
                  className="w-full pl-10 pr-10 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-brand-purple focus:border-transparent"
                />
                {userSearchInput && (
                  <button
                    type="button"
                    onClick={handleUserSearchClear}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
                  >
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                )}
              </div>
              <button
                type="button"
                onClick={handleUserSearch}
                className="px-4 py-2 text-sm font-medium rounded-lg bg-brand-purple text-white hover:bg-brand-purple/90 transition-colors"
              >
                Search
              </button>
            </div>
          </div>

          {/* Users Table */}
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-200 dark:border-slate-700">
                  <th className="text-left py-3 px-4 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                    Username
                  </th>
                  <th className="text-left py-3 px-4 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                    Name
                  </th>
                  <th className="text-left py-3 px-4 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="text-right py-3 px-4 text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                {roleUsersLoading ? (
                  <tr>
                    <td colSpan={4} className="py-8 text-center">
                      <div className="flex items-center justify-center">
                        <div className="h-6 w-6 animate-spin rounded-full border-2 border-brand-purple border-t-transparent"></div>
                      </div>
                    </td>
                  </tr>
                ) : roleUsers.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="py-8 text-center text-slate-500 dark:text-slate-400">
                      {userSearch ? 'No users found matching your search' : 'No users have this role yet'}
                    </td>
                  </tr>
                ) : (
                  roleUsers.map((user) => (
                    <tr key={user.id} className="hover:bg-slate-50 dark:hover:bg-slate-700/50">
                      <td className="py-3 px-4">
                        <span className="font-medium text-slate-900 dark:text-white">{user.username}</span>
                      </td>
                      <td className="py-3 px-4 text-slate-600 dark:text-slate-400">
                        {user.firstname} {user.lastname}
                      </td>
                      <td className="py-3 px-4">
                        <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                          user.active
                            ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                            : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                        }`}>
                          {user.active ? 'Active' : 'Inactive'}
                        </span>
                      </td>
                      <td className="py-3 px-4 text-right">
                        <div className="flex items-center justify-end gap-1">
                          <button
                            type="button"
                            onClick={() => navigate(`/admin/users/${encodeURIComponent(user.username)}`)}
                            className="p-1.5 text-slate-500 hover:text-brand-purple dark:text-slate-400 dark:hover:text-brand-purple hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
                            title="View user"
                          >
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                            </svg>
                          </button>
                          {canWrite && (
                            <button
                              type="button"
                              onClick={() => handleRemoveUserClick(Number(user.id), user.username)}
                              className="p-1.5 text-red-500 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors"
                              title="Remove from role"
                            >
                              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M22 11H16M4 19.5v-.75C4 15.361 6.462 13 12 13c.83 0 1.595.077 2.295.216M15 7a4 4 0 11-8 0 4 4 0 018 0z" />
                              </svg>
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="mt-4 flex items-center justify-between">
              <p className="text-sm text-slate-500 dark:text-slate-400">
                Showing {((currentPage - 1) * PAGE_SIZE) + 1} to {Math.min(currentPage * PAGE_SIZE, totalUsers)} of {totalUsers} users
              </p>
              <div className="flex gap-1">
                <button
                  type="button"
                  onClick={() => handleUserPageChange(currentPage - 1)}
                  disabled={currentPage === 1}
                  className="px-3 py-1 text-sm rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Previous
                </button>
                {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                  let pageNum: number
                  if (totalPages <= 5) {
                    pageNum = i + 1
                  } else if (currentPage <= 3) {
                    pageNum = i + 1
                  } else if (currentPage >= totalPages - 2) {
                    pageNum = totalPages - 4 + i
                  } else {
                    pageNum = currentPage - 2 + i
                  }
                  return (
                    <button
                      key={pageNum}
                      type="button"
                      onClick={() => handleUserPageChange(pageNum)}
                      className={`px-3 py-1 text-sm rounded-lg border ${
                        currentPage === pageNum
                          ? 'bg-brand-purple text-white border-brand-purple'
                          : 'border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700'
                      }`}
                    >
                      {pageNum}
                    </button>
                  )
                })}
                <button
                  type="button"
                  onClick={() => handleUserPageChange(currentPage + 1)}
                  disabled={currentPage === totalPages}
                  className="px-3 py-1 text-sm rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Next
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Remove User Confirmation Modal */}
      {removeUserConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="absolute inset-0 bg-black/50 backdrop-blur-sm"
            onClick={() => setRemoveUserConfirm(null)}
          />
          <div className="relative w-full max-w-md mx-4 rounded-xl bg-white dark:bg-slate-800 shadow-xl border border-slate-200 dark:border-slate-700 p-6">
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
              Remove User from Role
            </h3>
            <p className="text-slate-600 dark:text-slate-400 mb-6">
              Are you sure you want to remove <strong>{removeUserConfirm.username}</strong> from role <strong>{roleCode}</strong>?
            </p>
            <div className="flex gap-3 justify-end">
              <button
                type="button"
                onClick={() => setRemoveUserConfirm(null)}
                className="px-4 py-2 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleRemoveUserConfirm}
                disabled={removeUserLoading}
                className="px-4 py-2 text-sm font-medium rounded-lg bg-red-600 text-white hover:bg-red-700 transition-colors disabled:opacity-50"
              >
                {removeUserLoading ? 'Removing...' : 'Remove'}
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
              Delete Role
            </h3>
            <p className="text-slate-600 dark:text-slate-400 mb-6">
              Are you sure you want to delete role <strong>{roleCode}</strong>? This will also remove this role from all users. This action cannot be undone.
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
