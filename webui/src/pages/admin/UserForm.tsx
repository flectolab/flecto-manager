import { useState, useEffect, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation } from '@apollo/client/react'
import { CreateUserDocument, UpdateUserDocument, UpdateUserStatusDocument, DeleteUserDocument, GetRolesDocument, UpdateUserPermissionsDocument, GetNamespacesDocument, GetUserDocument } from '../../generated/graphql'
import { usePermissions, AdminSection, Action } from '../../hooks/usePermissions'
import { useDocumentTitle } from '../../hooks/useDocumentTitle'
import { RelativeTime } from '../../components/RelativeTime'
import { UnsavedChangesIndicator } from '../../components/UnsavedChangesIndicator'
import { ResourcePermissionsEditor, AdminPermissionsEditor } from '../../components/admin'
import type { ResourcePermission, AdminPermission } from '../../components/admin'

export function UserForm() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { canAdminResource } = usePermissions()

  const isEditing = !!id
  useDocumentTitle(isEditing ? `Admin - Edit User` : 'Admin - Add User')
  const canWrite = canAdminResource(AdminSection.Users, Action.Write)
  const canReadRoles = canAdminResource(AdminSection.Roles, Action.Read)
  const canManageRoles = canWrite && canReadRoles
  const isReadOnly = isEditing && !canWrite

  const [username, setUsername] = useState('')
  const [firstname, setFirstname] = useState('')
  const [lastname, setLastname] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState(false)
  const [userActive, setUserActive] = useState(true)
  const [isModified, setIsModified] = useState(false)

  // Roles state
  const [userRoles, setUserRoles] = useState<string[]>([])
  const [roleInput, setRoleInput] = useState('')
  const [showRoleSuggestions, setShowRoleSuggestions] = useState(false)
  const [permissionsModified, setPermissionsModified] = useState(false)
  const roleInputRef = useRef<HTMLInputElement>(null)
  const suggestionsRef = useRef<HTMLDivElement>(null)

  // Resource permissions state
  const [resourcePermissions, setResourcePermissions] = useState<ResourcePermission[]>([])

  // Admin permissions state
  const [adminPermissions, setAdminPermissions] = useState<AdminPermission[]>([])

  // Fetch user data if editing/viewing
  const { data: userData, loading: userLoading, refetch } = useQuery(GetUserDocument, {
    variables: { username: id! },
    skip: !isEditing,
  })

  // Fetch all available roles (only if user can manage roles)
  const { data: rolesData } = useQuery(GetRolesDocument, {
    skip: !canManageRoles,
  })

  // Fetch namespaces for dropdown
  const { data: namespacesData } = useQuery(GetNamespacesDocument, {
    skip: !canManageRoles,
  })

  const [createUser, { loading: createLoading }] = useMutation(CreateUserDocument)
  const [updateUser, { loading: updateLoading }] = useMutation(UpdateUserDocument)
  const [updateUserStatus, { loading: statusLoading }] = useMutation(UpdateUserStatusDocument)
  const [deleteUser, { loading: deleteLoading }] = useMutation(DeleteUserDocument)
  const [updateUserPermissions, { loading: permissionsLoading }] = useMutation(UpdateUserPermissionsDocument)

  const isLoading = createLoading || updateLoading || statusLoading || deleteLoading || permissionsLoading

  // Populate form when user data is loaded
  useEffect(() => {
    if (userData?.user) {
      setUsername(userData.user.username)
      setFirstname(userData.user.firstname || '')
      setLastname(userData.user.lastname || '')
      setUserActive(userData.user.active)

      // Extract roles and permissions from the roles array
      const roles = userData.user.roles
      const userRole = roles.find(r => r.type === 'user')
      const assignedRoles = roles.filter(r => r.type === 'role')

      // Set assigned role codes
      setUserRoles(assignedRoles.map(r => r.code))

      // Build permissions: user's direct permissions (type='user') + inherited from roles (type='role')
      const resourcePerms: ResourcePermission[] = []
      const adminPerms: AdminPermission[] = []

      // Direct permissions from user's personal role
      if (userRole) {
        userRole.resources.forEach(p => {
          resourcePerms.push({ type: 'user', namespace: p.namespace, project: p.project, resource: p.resource, action: p.action })
        })
        userRole.admin.forEach(p => {
          adminPerms.push({ type: 'user', section: p.section, action: p.action })
        })
      }

      // Inherited permissions from assigned roles
      assignedRoles.forEach(role => {
        role.resources.forEach(p => {
          resourcePerms.push({ type: 'role', namespace: p.namespace, project: p.project, resource: p.resource, action: p.action })
        })
        role.admin.forEach(p => {
          adminPerms.push({ type: 'role', section: p.section, action: p.action })
        })
      })

      setResourcePermissions(resourcePerms)
      setAdminPermissions(adminPerms)
      setIsModified(false)
      setPermissionsModified(false)
    }
  }, [userData])

  // Close suggestions when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        suggestionsRef.current &&
        !suggestionsRef.current.contains(event.target as Node) &&
        roleInputRef.current &&
        !roleInputRef.current.contains(event.target as Node)
      ) {
        setShowRoleSuggestions(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Get available roles for autocomplete (exclude already assigned roles)
  const availableRoles = rolesData?.roles
    .map(r => r.code)
    .filter(code => !userRoles.includes(code))
    .filter(code => code.toLowerCase().includes(roleInput.toLowerCase())) || []

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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (isReadOnly) return

    setError('')
    setSuccess('')

    try {
      if (isEditing && userData?.user) {
        await updateUser({
          variables: {
            id: userData.user.id,
            input: {
              firstname,
              lastname,
            },
          },
        })
        setSuccess('User updated successfully')
        setIsModified(false)
      } else {
        if (!password) {
          setError('Password is required')
          return
        }
        const result = await createUser({
          variables: {
            input: {
              username,
              password,
              firstname,
              lastname,
            },
          },
        })
        // Navigate to the new user's page after creation
        const newUsername = result.data?.createUser?.username
        if (newUsername) {
          navigate(`/admin/users/${newUsername}`)
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred')
    }
  }

  const handleCancel = () => {
    navigate('/admin/users')
  }

  const handleToggleActive = async () => {
    if (!userData?.user) return
    setError('')
    setSuccess('')
    try {
      await updateUserStatus({
        variables: {
          id: userData.user.id,
          input: { active: !userActive },
        },
      })
      setUserActive(!userActive)
      setSuccess(userActive ? 'User deactivated successfully' : 'User activated successfully')
      refetch()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update user status')
    }
  }

  const handleDeleteClick = () => {
    setDeleteConfirm(true)
  }

  const handleDeleteConfirm = async () => {
    if (!userData?.user) return
    setError('')
    try {
      await deleteUser({
        variables: { id: userData.user.id },
      })
      setDeleteConfirm(false)
      navigate('/admin/users')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete user')
      setDeleteConfirm(false)
    }
  }

  const handleDeleteCancel = () => {
    setDeleteConfirm(false)
  }

  // Role management handlers
  const handleAddRole = (role: string) => {
    if (!userRoles.includes(role)) {
      setUserRoles([...userRoles, role])
      setPermissionsModified(true)
    }
    setRoleInput('')
    setShowRoleSuggestions(false)
  }

  const handleRemoveRole = (role: string) => {
    setUserRoles(userRoles.filter(r => r !== role))
    setPermissionsModified(true)
  }

  // Resource permission handlers
  const handleAddResourcePermission = () => {
    setResourcePermissions([
      ...resourcePermissions,
      { type: 'user', namespace: '*', project: '*', resource: '*', action: 'read' }
    ])
    setPermissionsModified(true)
  }

  const handleRemoveResourcePermission = (index: number) => {
    setResourcePermissions(resourcePermissions.filter((_, i) => i !== index))
    setPermissionsModified(true)
  }

  const handleResourcePermissionChange = (index: number, field: keyof ResourcePermission, value: string) => {
    const updated = [...resourcePermissions]
    const perm = { ...updated[index] }

    if (field === 'namespace') {
      perm.namespace = value
      // If namespace is *, project must be *
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
    setPermissionsModified(true)
  }

  // Admin permission handlers
  const handleAddAdminPermission = () => {
    setAdminPermissions([
      ...adminPermissions,
      { type: 'user', section: '*', action: 'read' }
    ])
    setPermissionsModified(true)
  }

  const handleRemoveAdminPermission = (index: number) => {
    setAdminPermissions(adminPermissions.filter((_, i) => i !== index))
    setPermissionsModified(true)
  }

  const handleAdminPermissionChange = (index: number, field: 'section' | 'action', value: string) => {
    const updated = [...adminPermissions]
    updated[index] = { ...updated[index], [field]: value }
    setAdminPermissions(updated)
    setPermissionsModified(true)
  }

  const handleSavePermissions = async () => {
    if (!userData?.user) return
    setError('')
    setSuccess('')
    try {
      // Only send user-type permissions, not role-inherited ones
      const userResourcePermissions = resourcePermissions
        .filter(p => p.type === 'user')
        .map(p => ({
          namespace: p.namespace,
          project: p.project,
          resource: p.resource,
          action: p.action,
        }))

      const userAdminPermissions = adminPermissions
        .filter(p => p.type === 'user')
        .map(p => ({
          section: p.section,
          action: p.action,
        }))

      await updateUserPermissions({
        variables: {
          id: userData.user.id,
          input: {
            roles: userRoles,
            resources: userResourcePermissions,
            admin: userAdminPermissions,
          },
        },
      })
      setSuccess('Permissions updated successfully')
      setPermissionsModified(false)
      refetch()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update permissions')
    }
  }

  const handleRoleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      if (availableRoles.length > 0) {
        handleAddRole(availableRoles[0])
      }
    } else if (e.key === 'Escape') {
      setShowRoleSuggestions(false)
    }
  }

  if (isEditing && userLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-purple border-t-transparent"></div>
      </div>
    )
  }

  if (isEditing && !userData?.user) {
    return (
      <div className="rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4">
        <p className="text-red-700 dark:text-red-400">User not found</p>
      </div>
    )
  }

  const getPageTitle = () => {
    if (!isEditing) return 'Create User'
    if (isReadOnly) return 'View User'
    return 'Edit User'
  }

  const getPageDescription = () => {
    if (!isEditing) return 'Add a new user to the system'
    if (isReadOnly) return 'View user information'
    return 'Update user information'
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
          Back to Users
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
        <div className="lg:col-span-2 space-y-6">
          {/* User Information */}
          <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                User Information
              </h3>
              <UnsavedChangesIndicator show={isModified && isEditing && canWrite} />
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              {error && (
                <div className="rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-3">
                  <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
                </div>
              )}

              {success && (
                <div className="rounded-lg bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 p-3">
                  <p className="text-sm text-green-600 dark:text-green-400">{success}</p>
                </div>
              )}

              {/* Username */}
              <div>
                <label htmlFor="username" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Username
                </label>
                <input
                  id="username"
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  disabled={isEditing || isReadOnly}
                  required
                  className={`w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-2 px-3 text-slate-900 dark:text-white placeholder-slate-400 focus:border-brand-purple focus:outline-none focus:ring-2 focus:ring-brand-purple/20 ${
                    isEditing || isReadOnly ? 'opacity-50 cursor-not-allowed bg-slate-50 dark:bg-slate-800' : ''
                  }`}
                  placeholder="Enter username"
                />
                {isEditing && !isReadOnly && (
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Username cannot be changed
                  </p>
                )}
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                {/* Lastname */}
                <div>
                  <label htmlFor="lastname" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                    Last Name
                  </label>
                  <input
                    id="lastname"
                    type="text"
                    value={lastname}
                    onChange={(e) => { setLastname(e.target.value); setIsModified(true) }}
                    disabled={isReadOnly}
                    required={!isReadOnly}
                    className={`w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-2 px-3 text-slate-900 dark:text-white placeholder-slate-400 focus:border-brand-purple focus:outline-none focus:ring-2 focus:ring-brand-purple/20 ${
                      isReadOnly ? 'opacity-50 cursor-not-allowed bg-slate-50 dark:bg-slate-800' : ''
                    }`}
                    placeholder="Enter last name"
                  />
                </div>

                {/* Firstname */}
                <div>
                  <label htmlFor="firstname" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                    First Name
                  </label>
                  <input
                    id="firstname"
                    type="text"
                    value={firstname}
                    onChange={(e) => { setFirstname(e.target.value); setIsModified(true) }}
                    disabled={isReadOnly}
                    required={!isReadOnly}
                    className={`w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-2 px-3 text-slate-900 dark:text-white placeholder-slate-400 focus:border-brand-purple focus:outline-none focus:ring-2 focus:ring-brand-purple/20 ${
                      isReadOnly ? 'opacity-50 cursor-not-allowed bg-slate-50 dark:bg-slate-800' : ''
                    }`}
                    placeholder="Enter first name"
                  />
                </div>
              </div>

              {/* Password - only for creation */}
              {!isEditing && (
                <div>
                  <label htmlFor="password" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                    Password
                  </label>
                  <input
                    id="password"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    className="w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-2 px-3 text-slate-900 dark:text-white placeholder-slate-400 focus:border-brand-purple focus:outline-none focus:ring-2 focus:ring-brand-purple/20"
                    placeholder="Enter password"
                  />
                </div>
              )}

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
                      'Create User'
                    )}
                  </button>
                )}
              </div>
            </form>
          </div>

          {/* Permissions section (only when editing and user can manage roles) */}
          {isEditing && userData?.user && canManageRoles && (
            <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
              <div className="flex items-center justify-between mb-6">
                <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
                  Permissions
                </h3>
                <UnsavedChangesIndicator show={permissionsModified && canWrite} />
              </div>

              {/* Roles subsection */}
              <div className="mb-6">
                <h4 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-3">Roles</h4>

                {/* Current roles as tags */}
                <div className="flex flex-wrap gap-2 mb-3">
                  {userRoles.length === 0 ? (
                    <p className="text-sm text-slate-500 dark:text-slate-400">No roles assigned</p>
                  ) : (
                    userRoles.map((role) => (
                      <span
                        key={role}
                        className="inline-flex items-center gap-1 px-3 py-1 rounded-full text-sm font-medium bg-brand-purple/10 text-brand-purple dark:bg-brand-purple/20"
                      >
                        {role}
                        {canWrite && (
                          <button
                            type="button"
                            onClick={() => handleRemoveRole(role)}
                            className="ml-1 hover:text-red-500 transition-colors"
                          >
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                            </svg>
                          </button>
                        )}
                      </span>
                    ))
                  )}
                </div>

                {/* Add role input with autocomplete */}
                {canWrite && (
                  <div className="relative">
                    <input
                      ref={roleInputRef}
                      type="text"
                      value={roleInput}
                      onChange={(e) => {
                        setRoleInput(e.target.value)
                        setShowRoleSuggestions(true)
                      }}
                      onFocus={() => setShowRoleSuggestions(true)}
                      onKeyDown={handleRoleInputKeyDown}
                      placeholder="Type to search roles..."
                      className="w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-2 px-3 text-slate-900 dark:text-white placeholder-slate-400 focus:border-brand-purple focus:outline-none focus:ring-2 focus:ring-brand-purple/20"
                    />

                    {/* Autocomplete suggestions */}
                    {showRoleSuggestions && availableRoles.length > 0 && (
                      <div
                        ref={suggestionsRef}
                        className="absolute z-10 w-full mt-1 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg max-h-48 overflow-auto"
                      >
                        {availableRoles.map((role) => (
                          <button
                            key={role}
                            type="button"
                            onClick={() => handleAddRole(role)}
                            className="w-full px-3 py-2 text-left text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                          >
                            {role}
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>

              {/* Resource Permissions subsection */}
              <div className="mb-6">
                <ResourcePermissionsEditor
                  permissions={resourcePermissions}
                  namespaceOptions={namespaceOptions}
                  getProjectOptions={getProjectOptions}
                  onChange={handleResourcePermissionChange}
                  onAdd={handleAddResourcePermission}
                  onRemove={handleRemoveResourcePermission}
                  canWrite={canWrite}
                  showInheritedBadge
                />
              </div>

              {/* Admin Permissions subsection */}
              <div className="mb-6">
                <AdminPermissionsEditor
                  permissions={adminPermissions}
                  onChange={handleAdminPermissionChange}
                  onAdd={handleAddAdminPermission}
                  onRemove={handleRemoveAdminPermission}
                  canWrite={canWrite}
                  showInheritedBadge
                />
              </div>

              {/* Save Permissions button */}
              {canWrite && (
                <div className="pt-4 border-t border-slate-200 dark:border-slate-700">
                  <button
                    type="button"
                    onClick={handleSavePermissions}
                    disabled={!permissionsModified || isLoading}
                    className="px-4 py-2 text-sm font-medium rounded-lg bg-gradient-to-r from-brand-purple to-brand-indigo text-white hover:opacity-90 transition-opacity disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {permissionsLoading ? (
                      <span className="flex items-center gap-2">
                        <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24">
                          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                        </svg>
                        Saving...
                      </span>
                    ) : (
                      'Save Permissions'
                    )}
                  </button>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Sidebar - User info & Actions (for editing/viewing) */}
        {isEditing && userData?.user && (
          <div className="space-y-6">
            {/* User Status */}
            <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
              <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
                Status
              </h3>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Account Status</span>
                  <span
                    className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                      userActive
                        ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
                        : 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
                    }`}
                  >
                    {userActive ? 'Active' : 'Inactive'}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Created</span>
                  <RelativeTime
                    date={userData.user.createdAt}
                    className="text-sm text-slate-900 dark:text-white cursor-help"
                  />
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Last Updated</span>
                  <RelativeTime
                    date={userData.user.updatedAt}
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
                  {canWrite && (
                    <button
                      onClick={handleToggleActive}
                      disabled={isLoading}
                      className={`w-full flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium rounded-lg border transition-colors disabled:opacity-50 ${
                        userActive
                          ? 'border-orange-200 dark:border-orange-800 text-orange-600 dark:text-orange-400 hover:bg-orange-50 dark:hover:bg-orange-900/20'
                          : 'border-green-200 dark:border-green-800 text-green-600 dark:text-green-400 hover:bg-green-50 dark:hover:bg-green-900/20'
                      }`}
                    >
                      {userActive ? (
                        <>
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
                          </svg>
                          Deactivate User
                        </>
                      ) : (
                        <>
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                          </svg>
                          Activate User
                        </>
                      )}
                    </button>
                  )}
                  {canWrite && (
                    <button
                      onClick={handleDeleteClick}
                      disabled={isLoading}
                      className="w-full flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium rounded-lg border border-red-200 dark:border-red-800 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors disabled:opacity-50"
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                      Delete User
                    </button>
                  )}
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
              Delete User
            </h3>
            <p className="text-slate-600 dark:text-slate-400 mb-6">
              Are you sure you want to delete user <strong>{username}</strong>? This action cannot be undone.
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
