import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation } from '@apollo/client/react'
import { GetTokenDocument, CreateTokenDocument, UpdateTokenPermissionsDocument, DeleteTokenDocument, GetNamespacesDocument } from '../../generated/graphql'
import { usePermissions, AdminSection, Action, validateCode } from '../../hooks/usePermissions'
import { useDocumentTitle } from '../../hooks/useDocumentTitle'
import { UnsavedChangesIndicator } from '../../components/UnsavedChangesIndicator'
import { RelativeTime } from '../../components/RelativeTime'
import { ResourcePermissionsEditor, AdminPermissionsEditor } from '../../components/admin'
import type { ResourcePermission, AdminPermission } from '../../components/admin'

export function TokenForm() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { canAdminResource } = usePermissions()

  const isEditing = !!id && id !== 'new'
  const tokenId = isEditing ? parseInt(id, 10) : 0
  useDocumentTitle(isEditing ? `Admin - Edit Token` : 'Admin - Create Token')
  const canWrite = canAdminResource(AdminSection.Tokens, Action.Write)
  const isReadOnly = isEditing && !canWrite

  const [tokenName, setTokenName] = useState('')
  const [tokenNameError, setTokenNameError] = useState('')
  const [expiresAt, setExpiresAt] = useState('')
  const [resourcePermissions, setResourcePermissions] = useState<ResourcePermission[]>([])
  const [adminPermissions, setAdminPermissions] = useState<AdminPermission[]>([])
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState(false)
  const [isModified, setIsModified] = useState(false)
  const [createdToken, setCreatedToken] = useState<{ name: string; plainToken: string } | null>(null)

  // Fetch the token we're editing
  const { data: tokenData, loading: tokenLoading, refetch } = useQuery(GetTokenDocument, {
    variables: { id: tokenId },
    skip: !isEditing,
  })

  // Fetch namespaces for dropdown
  const { data: namespacesData } = useQuery(GetNamespacesDocument)

  const [createToken, { loading: createLoading }] = useMutation(CreateTokenDocument)
  const [updateTokenPermissions, { loading: updateLoading }] = useMutation(UpdateTokenPermissionsDocument)
  const [deleteToken, { loading: deleteLoading }] = useMutation(DeleteTokenDocument)

  const isLoading = createLoading || updateLoading || deleteLoading

  // Populate form when token data is loaded
  useEffect(() => {
    if (tokenData?.token) {
      setTokenName(tokenData.token.name)
      setExpiresAt(tokenData.token.expiresAt ? new Date(tokenData.token.expiresAt).toISOString().slice(0, 16) : '')
      setResourcePermissions(tokenData.token.role?.resources?.map(r => ({
        namespace: r.namespace,
        project: r.project,
        resource: r.resource,
        action: r.action,
      })) || [])
      setAdminPermissions(tokenData.token.role?.admin?.map(a => ({
        section: a.section,
        action: a.action,
      })) || [])
      setIsModified(false)
    }
  }, [tokenData])

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

  const handleTokenNameChange = (value: string) => {
    setTokenName(value)
    setIsModified(true)
    if (tokenNameError) {
      setTokenNameError(validateCode(value, 'Token name'))
    }
  }

  const handleTokenNameBlur = () => {
    if (!isEditing) {
      setTokenNameError(validateCode(tokenName, 'Token name'))
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (isReadOnly) return

    setError('')
    setTokenNameError('')
    setSuccess('')

    try {
      if (isEditing) {
        await updateTokenPermissions({
          variables: {
            id: tokenId,
            input: {
              resourcePermissions: resourcePermissions,
              adminPermissions: adminPermissions,
            },
          },
        })
        setSuccess('Token permissions updated successfully')
        refetch()
      } else {
        // Validate token name
        const nameError = validateCode(tokenName, 'Token name')
        if (nameError) {
          setTokenNameError(nameError)
          return
        }

        const result = await createToken({
          variables: {
            input: {
              name: tokenName.trim(),
              expiresAt: expiresAt ? new Date(expiresAt).toISOString() : null,
              resourcePermissions: resourcePermissions.length > 0 ? resourcePermissions : null,
              adminPermissions: adminPermissions.length > 0 ? adminPermissions : null,
            },
          },
        })

        if (result.data?.createToken) {
          setCreatedToken({
            name: result.data.createToken.token.name,
            plainToken: result.data.createToken.plainToken,
          })
        }
      }

      setIsModified(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred')
    }
  }

  const handleCancel = () => {
    navigate('/admin/tokens')
  }

  const handleDeleteClick = () => {
    setDeleteConfirm(true)
  }

  const handleDeleteConfirm = async () => {
    if (!tokenData?.token) return
    setError('')
    try {
      await deleteToken({
        variables: { id: tokenId },
      })
      setDeleteConfirm(false)
      navigate('/admin/tokens')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete token')
      setDeleteConfirm(false)
    }
  }

  const handleDeleteCancel = () => {
    setDeleteConfirm(false)
  }

  const handleCopyToken = () => {
    if (createdToken) {
      navigator.clipboard.writeText(createdToken.plainToken)
    }
  }

  if (isEditing && tokenLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-purple border-t-transparent"></div>
      </div>
    )
  }

  if (isEditing && !tokenData?.token) {
    return (
      <div className="rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4">
        <p className="text-red-700 dark:text-red-400">Token not found</p>
      </div>
    )
  }

  const getPageTitle = () => {
    if (!isEditing) return 'Create API Token'
    if (isReadOnly) return 'View API Token'
    return 'Edit API Token'
  }

  const getPageDescription = () => {
    if (!isEditing) return 'Create a new API token for programmatic access'
    if (isReadOnly) return 'View token configuration'
    return 'Update token permissions'
  }

  // Show token created success modal
  if (createdToken) {
    return (
      <div>
        <div className="mb-6">
          <button
            onClick={handleCancel}
            className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400 hover:text-brand-purple dark:hover:text-brand-purple mb-4"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 19l-7-7 7-7" />
            </svg>
            Back to Tokens
          </button>
          <h2 className="text-2xl font-bold text-slate-900 dark:text-white">Token Created</h2>
        </div>

        <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
          <div className="mb-6 p-4 rounded-lg bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800">
            <div className="flex items-start gap-3">
              <svg className="w-6 h-6 text-green-600 dark:text-green-400 shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <div>
                <h3 className="font-semibold text-green-800 dark:text-green-300">Token created successfully!</h3>
                <p className="text-sm text-green-700 dark:text-green-400 mt-1">
                  Make sure to copy your token now. You won't be able to see it again!
                </p>
              </div>
            </div>
          </div>

          <div className="mb-6">
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              Token Name
            </label>
            <p className="text-slate-900 dark:text-white font-medium">{createdToken.name}</p>
          </div>

          <div className="mb-6">
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              API Token
            </label>
            <div className="flex items-center gap-2">
              <code className="flex-1 p-3 rounded-lg bg-slate-100 dark:bg-slate-900 font-mono text-sm text-slate-900 dark:text-white break-all">
                {createdToken.plainToken}
              </code>
              <button
                onClick={handleCopyToken}
                className="shrink-0 px-4 py-2 text-sm font-medium rounded-lg bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600 transition-colors"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                </svg>
              </button>
            </div>
            <p className="mt-2 text-xs text-amber-600 dark:text-amber-400">
              This token will only be shown once. Store it securely.
            </p>
          </div>

          <div className="flex gap-3 justify-end pt-4 border-t border-slate-200 dark:border-slate-700">
            <button
              onClick={handleCancel}
              className="px-4 py-2 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
            >
              Done
            </button>
          </div>
        </div>
      </div>
    )
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
          Back to Tokens
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

              {/* Token Name */}
              <div className="mb-6">
                <label htmlFor="tokenName" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                  Token Name
                </label>
                <input
                  id="tokenName"
                  type="text"
                  value={tokenName}
                  onChange={(e) => handleTokenNameChange(e.target.value)}
                  onBlur={handleTokenNameBlur}
                  disabled={isEditing || isReadOnly}
                  required
                  className={`w-full rounded-lg border bg-white dark:bg-slate-900 py-2 px-3 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 ${
                    tokenNameError
                      ? 'border-red-300 dark:border-red-700 focus:border-red-500 focus:ring-red-500/20'
                      : 'border-slate-200 dark:border-slate-700 focus:border-brand-purple focus:ring-brand-purple/20'
                  } ${
                    isEditing || isReadOnly ? 'opacity-50 cursor-not-allowed bg-slate-50 dark:bg-slate-800' : ''
                  }`}
                  placeholder="Enter token name (letters, numbers, _ and - only)"
                />
                {tokenNameError && (
                  <p className="mt-1 text-xs text-red-600 dark:text-red-400">
                    {tokenNameError}
                  </p>
                )}
                {isEditing && !isReadOnly && !tokenNameError && (
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Token name cannot be changed
                  </p>
                )}
              </div>

              {/* Token Preview (only in edit mode) */}
              {isEditing && tokenData?.token?.tokenPreview && (
                <div className="mb-6">
                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                    Token Preview
                  </label>
                  <code className="block p-3 rounded-lg bg-slate-100 dark:bg-slate-900 font-mono text-sm text-slate-600 dark:text-slate-400">
                    {tokenData.token.tokenPreview}
                  </code>
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    This is a preview of the token. The full token is only shown at creation.
                  </p>
                </div>
              )}

              {/* Expiration (only for new tokens) */}
              {!isEditing && (
                <div className="mb-6">
                  <label htmlFor="expiresAt" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                    Expiration (optional)
                  </label>
                  <input
                    id="expiresAt"
                    type="datetime-local"
                    value={expiresAt}
                    onChange={(e) => { setExpiresAt(e.target.value); setIsModified(true) }}
                    disabled={isReadOnly}
                    className="w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-2 px-3 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:border-brand-purple focus:ring-brand-purple/20"
                  />
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    Leave empty for a non-expiring token
                  </p>
                </div>
              )}

              {/* Permissions */}
              <>
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
              </>

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
                        {isEditing ? 'Saving...' : 'Creating...'}
                      </span>
                    ) : isEditing ? (
                      'Save Changes'
                    ) : (
                      'Create Token'
                    )}
                  </button>
                )}
              </div>
            </div>
          </form>
        </div>

        {/* Sidebar - Status & Actions (for editing) */}
        {isEditing && tokenData?.token && (
          <div className="space-y-6">
            {/* Status */}
            <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
              <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
                Status
              </h3>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Expiration</span>
                  <span className="text-sm text-slate-900 dark:text-white">
                    {tokenData.token.expiresAt ? (
                      new Date(tokenData.token.expiresAt).toLocaleDateString()
                    ) : (
                      <span className="text-slate-400">Never</span>
                    )}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Created</span>
                  <RelativeTime
                    date={tokenData.token.createdAt}
                    className="text-sm text-slate-900 dark:text-white cursor-help"
                  />
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-600 dark:text-slate-400">Last Updated</span>
                  <RelativeTime
                    date={tokenData.token.updatedAt}
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
                    Delete Token
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
              Delete Token
            </h3>
            <p className="text-slate-600 dark:text-slate-400 mb-6">
              Are you sure you want to delete token <strong>{tokenName}</strong>? Any applications using this token will no longer be able to authenticate. This action cannot be undone.
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
