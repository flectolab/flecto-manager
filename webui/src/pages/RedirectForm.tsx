import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation } from '@apollo/client/react'
import { useDocumentTitle } from '../hooks/useDocumentTitle'
import {
  GetProjectRedirectDocument,
  GetProjectRedirectsDocument,
  GetProjectDocument,
  CreateRedirectDraftDocument,
  UpdateRedirectDraftDocument,
  type RedirectType,
  type RedirectStatus,
} from '../generated/graphql'
import { useCurrentProject } from '../hooks/useCurrentProject'
import { usePermissions, Action, ResourceType } from '../hooks/usePermissions'
import { RelativeTime } from '../components/RelativeTime'
import { RedirectTypeBadge, RedirectStatusBadge, DraftBadge, RedirectTester } from '../components/redirects'

interface FormData {
  type: RedirectType
  source: string
  target: string
  status: RedirectStatus
}

const redirectTypes: { value: RedirectType; label: string; description: string }[] = [
  { value: 'BASIC', label: 'Basic', description: 'Simple path matching' },
  { value: 'BASIC_HOST', label: 'Host', description: 'Match with host header' },
  { value: 'REGEX', label: 'Regex', description: 'Regular expression matching' },
  { value: 'REGEX_HOST', label: 'Regex Host', description: 'Regex matching with host header' },
]

const redirectStatuses: { value: RedirectStatus; label: string; code: number }[] = [
  { value: 'MOVED_PERMANENT', label: 'Moved Permanently', code: 301 },
  { value: 'FOUND', label: 'Found', code: 302 },
  { value: 'TEMPORARY_REDIRECT', label: 'Temporary Redirect', code: 307 },
  { value: 'PERMANENT_REDIRECT', label: 'Permanent Redirect', code: 308 },
]

// Placeholders based on redirect type
const sourcePlaceholders: Record<RedirectType, string> = {
  BASIC: '/old-path or /old-path?query=value',
  BASIC_HOST: 'example.com/old-path',
  REGEX: '^/old-path/.*$ or ^/old-path/(.*)$',
  REGEX_HOST: '^example\\.com/old-path/.*$ or ^example\\.com/old-path/(.*)$',
}

const targetPlaceholders: Record<RedirectType, string> = {
  BASIC: '/new-path',
  BASIC_HOST: 'example.com/new-path or /new-path',
  REGEX: '/new-path or /new-path/$1',
  REGEX_HOST: 'example.com/new-path or example.com/new-path/$1',
}

// Validation helpers
function isValidPath(source: string): boolean {
  if (!source.startsWith('/')) return false
  try {
    const url = new URL(source, 'http://localhost')
    return url.pathname === source.split('?')[0]
  } catch {
    return false
  }
}

function isValidHostPath(source: string): boolean {
  // Pour BASIC_HOST, le format attendu est "host/path", pas "/path"
  if (source.startsWith('/')) return false

  try {
    const normalized = '//' + source
    const url = new URL(normalized, 'http://localhost')
    // host doit exister et pathname doit contenir plus que juste '/'
    return url.host !== '' && url.pathname.length > 1
  } catch {
    return false
  }
}

function isValidRegex(source: string): boolean {
  try {
    new RegExp(source)
    return true
  } catch {
    return false
  }
}

export function RedirectForm() {
  const { namespace, project, id: redirectId } = useParams()
  const navigate = useNavigate()
  const { namespaceCode, projectCode } = useCurrentProject()
  const { canResource, loading: permissionsLoading } = usePermissions()

  // Determine mode from URL
  // - /redirects/add -> create mode
  // - /redirects/edit/:id -> edit existing redirect (may have a draft)
  const isCreateMode = !redirectId
  const isEditMode = !!redirectId

  const pageTitle = isCreateMode ? 'Add Redirect' : 'Edit Redirect'
  useDocumentTitle(namespaceCode && projectCode ? `${pageTitle} - ${namespaceCode}/${projectCode}` : pageTitle)

  const canWrite = namespaceCode && projectCode ? canResource(namespaceCode, projectCode, ResourceType.Redirect, Action.Write) : false

  // Fetch existing redirect if editing (includes its draft if any)
  const { data: redirectData, loading: redirectLoading } = useQuery(GetProjectRedirectDocument, {
    variables: {
      namespaceCode: namespaceCode ?? '',
      projectCode: projectCode ?? '',
      redirectID: redirectId ?? '',
    },
    skip: !namespaceCode || !projectCode || !redirectId,
  })

  const [formData, setFormData] = useState<FormData>({
    type: 'BASIC',
    source: '',
    target: '',
    status: 'MOVED_PERMANENT',
  })

  const [errors, setErrors] = useState<Partial<Record<keyof FormData, string>>>({})
  const [submitError, setSubmitError] = useState<string | null>(null)

  // Mutations
  const [createDraft, { loading: createLoading }] = useMutation(CreateRedirectDraftDocument, {
    refetchQueries: [GetProjectRedirectsDocument, GetProjectDocument],
  })

  const [updateDraft, { loading: updateLoading }] = useMutation(UpdateRedirectDraftDocument, {
    refetchQueries: [GetProjectRedirectsDocument, GetProjectDocument],
  })

  // Initialize form data when redirect is loaded
  useEffect(() => {
    if (isEditMode && redirectData?.projectRedirect) {
      const redirect = redirectData.projectRedirect
      // If there's a draft, use the draft's newRedirect data
      if (redirect.redirectDraft?.newRedirect) {
        setFormData({
          type: redirect.redirectDraft.newRedirect.type,
          source: redirect.redirectDraft.newRedirect.source,
          target: redirect.redirectDraft.newRedirect.target,
          status: redirect.redirectDraft.newRedirect.status,
        })
      } else {
        // Use the redirect's current data
        setFormData({
          type: redirect.type,
          source: redirect.source ?? '',
          target: redirect.target,
          status: redirect.status,
        })
      }
    }
  }, [isEditMode, redirectData])

  const validate = (): boolean => {
    const newErrors: Partial<Record<keyof FormData, string>> = {}

    if (!formData.source.trim()) {
      newErrors.source = 'Source is required'
    } else {
      // Validate source based on type
      switch (formData.type) {
        case 'BASIC':
          if (!isValidPath(formData.source)) {
            newErrors.source = 'Source must be a valid path starting with /'
          }
          break
        case 'BASIC_HOST':
          if (!isValidHostPath(formData.source)) {
            newErrors.source = 'Source must include a valid host and path (e.g., example.com/path)'
          }
          break
        case 'REGEX':
        case 'REGEX_HOST':
          if (!isValidRegex(formData.source)) {
            newErrors.source = 'Source must be a valid regular expression'
          }
          break
      }
    }

    if (!formData.target.trim()) {
      newErrors.target = 'Target is required'
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!validate() || !namespaceCode || !projectCode) return

    setSubmitError(null)

    try {
      if (isCreateMode) {
        // Create new redirect draft
        await createDraft({
          variables: {
            namespaceCode,
            projectCode,
            input: {
              newRedirect: formData,
            },
          },
        })
      } else if (isEditMode && redirectId) {
        const redirect = redirectData?.projectRedirect
        if (redirect?.redirectDraft) {
          // Update existing draft
          await updateDraft({
            variables: {
              namespaceCode,
              projectCode,
              redirectDraftID: redirect.redirectDraft.id,
              input: {
                newRedirect: formData,
              },
            },
          })
        } else {
          // Create new UPDATE draft
          await createDraft({
            variables: {
              namespaceCode,
              projectCode,
              input: {
                oldRedirectID: redirectId,
                newRedirect: formData,
              },
            },
          })
        }
      }

      // Navigate back to redirects list
      navigate(`/${namespace}/${project}/redirects`)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'An unexpected error occurred'
      setSubmitError(message)
    }
  }

  const handleCancel = () => {
    navigate(`/${namespace}/${project}/redirects`)
  }

  const handleChange = (field: keyof FormData, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }))
    if (errors[field]) {
      setErrors((prev) => ({ ...prev, [field]: undefined }))
    }
  }

  const isLoading = redirectLoading || permissionsLoading
  const isSaving = createLoading || updateLoading

  // Get redirect and draft data
  const redirect = redirectData?.projectRedirect
  const draft = redirect?.redirectDraft
  const changeType = draft?.changeType
  // For comparison: use redirect's current values (before draft changes)
  const oldRedirect = redirect

  // Helper to check if a field has changed
  const hasChanged = (field: keyof FormData): boolean => {
    if (!oldRedirect || changeType === 'CREATE') return false
    const oldValue = field === 'source' ? oldRedirect.source ?? '' : oldRedirect[field]
    return formData[field] !== oldValue
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-purple border-t-transparent"></div>
      </div>
    )
  }

  if (!canWrite) {
    return (
      <div className="rounded-xl bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 p-6">
        <div className="flex items-center gap-3">
          <svg className="w-6 h-6 text-amber-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div>
            <h3 className="font-semibold text-amber-800 dark:text-amber-300">Access Denied</h3>
            <p className="text-amber-700 dark:text-amber-400">You don't have permission to modify redirects for this project.</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="max-w-[1600px]">
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center gap-3 mb-2">
          <button
            onClick={handleCancel}
            className="p-2 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <h2 className="text-2xl font-bold text-slate-900 dark:text-white">
            {isCreateMode ? 'Add Redirect' : 'Edit Redirect'}
          </h2>
          {changeType && <DraftBadge changeType={changeType} />}
        </div>
        <p className="text-slate-600 dark:text-slate-400 ml-12">
          {isCreateMode
            ? 'Create a new redirect rule'
            : 'Modify the redirect rule. Changes will be saved as a draft.'}
        </p>
      </div>

      <div className="flex flex-col lg:flex-row gap-6">
        {/* Form */}
        <div className="flex-1 min-w-0">
          <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
            <form onSubmit={handleSubmit} className="space-y-6">
              {/* Submit Error */}
              {submitError && (
                <div className="rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4">
                  <p className="text-red-700 dark:text-red-400">{submitError}</p>
                </div>
              )}

              {/* Type */}
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  Type
                </label>
                <div className="flex gap-2">
                  {redirectTypes.map((type) => (
                    <button
                      key={type.value}
                      type="button"
                      onClick={() => handleChange('type', type.value)}
                      className={`flex-1 px-4 py-2.5 text-sm font-medium rounded-lg border transition-colors ${
                        formData.type === type.value
                          ? 'border-brand-purple bg-brand-purple/10 text-brand-purple'
                          : 'border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700'
                      }`}
                      title={type.description}
                    >
                      {type.label}
                    </button>
                  ))}
                </div>
                {hasChanged('type') && oldRedirect && (
                  <p className="mt-2 text-xs text-amber-600 dark:text-amber-400 flex items-center gap-1">
                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01" />
                    </svg>
                    Changed from <RedirectTypeBadge type={oldRedirect.type} />
                  </p>
                )}
              </div>

              {/* Source */}
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  Source
                </label>
                <input
                  type="text"
                  value={formData.source}
                  onChange={(e) => handleChange('source', e.target.value)}
                  placeholder={sourcePlaceholders[formData.type]}
                  className={`w-full rounded-lg border bg-white dark:bg-slate-900 py-2.5 px-4 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-brand-purple/20 font-mono ${
                    errors.source
                      ? 'border-red-500 focus:border-red-500'
                      : 'border-slate-200 dark:border-slate-700 focus:border-brand-purple'
                  }`}
                />
                {errors.source && <p className="mt-1 text-sm text-red-500">{errors.source}</p>}
                {hasChanged('source') && oldRedirect && (
                  <p className="mt-2 text-xs text-amber-600 dark:text-amber-400 flex items-center gap-1">
                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01" />
                    </svg>
                    Changed from <code className="font-mono bg-slate-100 dark:bg-slate-700 px-1 rounded">{oldRedirect.source}</code>
                  </p>
                )}
              </div>

              {/* Target */}
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  Target
                </label>
                <input
                  type="text"
                  value={formData.target}
                  onChange={(e) => handleChange('target', e.target.value)}
                  placeholder={targetPlaceholders[formData.type]}
                  className={`w-full rounded-lg border bg-white dark:bg-slate-900 py-2.5 px-4 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-brand-purple/20 font-mono ${
                    errors.target
                      ? 'border-red-500 focus:border-red-500'
                      : 'border-slate-200 dark:border-slate-700 focus:border-brand-purple'
                  }`}
                />
                {errors.target && <p className="mt-1 text-sm text-red-500">{errors.target}</p>}
                {hasChanged('target') && oldRedirect && (
                  <p className="mt-2 text-xs text-amber-600 dark:text-amber-400 flex items-center gap-1">
                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01" />
                    </svg>
                    Changed from <code className="font-mono bg-slate-100 dark:bg-slate-700 px-1 rounded">{oldRedirect.target}</code>
                  </p>
                )}
              </div>

              {/* Status */}
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  Status Code
                </label>
                <select
                  value={formData.status}
                  onChange={(e) => handleChange('status', e.target.value)}
                  className="w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-2.5 px-4 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-purple/20 focus:border-brand-purple"
                >
                  {redirectStatuses.map((status) => (
                    <option key={status.value} value={status.value}>
                      {status.code} - {status.label}
                    </option>
                  ))}
                </select>
                {hasChanged('status') && oldRedirect && (
                  <p className="mt-2 text-xs text-amber-600 dark:text-amber-400 flex items-center gap-1">
                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01" />
                    </svg>
                    Changed from <RedirectStatusBadge status={oldRedirect.status} />
                  </p>
                )}
              </div>

              {/* Actions */}
              <div className="flex gap-3 justify-end pt-4 border-t border-slate-200 dark:border-slate-700">
                <button
                  type="button"
                  onClick={handleCancel}
                  disabled={isSaving}
                  className="px-4 py-2.5 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors disabled:opacity-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isSaving}
                  className="px-6 py-2.5 text-sm font-medium rounded-lg bg-gradient-to-r from-brand-purple to-brand-indigo text-white hover:opacity-90 transition-opacity disabled:opacity-50 flex items-center gap-2"
                >
                  {isSaving && (
                    <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                  )}
                  {isCreateMode ? 'Create Draft' : 'Save Changes'}
                </button>
              </div>
            </form>
          </div>

          {/* Test Section */}
          {namespaceCode && projectCode && (
            <div className="mt-6">
              <RedirectTester
                namespaceCode={namespaceCode}
                projectCode={projectCode}
                redirect={formData.source && formData.target ? formData : null}
              />
            </div>
          )}
        </div>

        {/* Sidebar - Info & Dates */}
        <div className="lg:w-[320px] flex-shrink-0 space-y-5">
          {/* Old Values Card (only for UPDATE mode) */}
          {!isCreateMode && oldRedirect && changeType !== 'CREATE' && (
            <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-5">
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
                <svg className="w-4 h-4 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                Current Values
              </h3>
              <div className="space-y-3 text-sm">
                <div className="flex justify-between items-center">
                  <span className="text-slate-500 dark:text-slate-400">Type</span>
                  <RedirectTypeBadge type={oldRedirect.type} />
                </div>
                <div className="flex justify-between items-center gap-2 min-w-0">
                  <span className="text-slate-500 dark:text-slate-400 flex-shrink-0">Source</span>
                  <span
                    className="font-mono text-slate-900 dark:text-white truncate min-w-0 cursor-help hover:text-cyan-600 dark:hover:text-cyan-400 transition-colors"
                    title={oldRedirect.source ?? ''}
                  >
                    {oldRedirect.source}
                  </span>
                </div>
                <div className="flex justify-between items-center gap-2 min-w-0">
                  <span className="text-slate-500 dark:text-slate-400 flex-shrink-0">Target</span>
                  <span
                    className="font-mono text-slate-900 dark:text-white truncate min-w-0 cursor-help hover:text-cyan-600 dark:hover:text-cyan-400 transition-colors"
                    title={oldRedirect.target}
                  >
                    {oldRedirect.target}
                  </span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-slate-500 dark:text-slate-400">Status</span>
                  <RedirectStatusBadge status={oldRedirect.status} />
                </div>
              </div>
            </div>
          )}

          {/* Redirect Dates (only for existing redirects) */}
          {!isCreateMode && oldRedirect && 'createdAt' in oldRedirect && (
            <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-5">
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
                <svg className="w-4 h-4 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                Redirect Info
              </h3>
              <div className="space-y-3 text-sm">
                <div className="flex justify-between">
                  <span className="text-slate-500 dark:text-slate-400">Created</span>
                  <span className="text-slate-900 dark:text-white">
                    <RelativeTime date={oldRedirect.createdAt} />
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-500 dark:text-slate-400">Updated</span>
                  <span className="text-slate-900 dark:text-white">
                    <RelativeTime date={oldRedirect.updatedAt} />
                  </span>
                </div>
                {oldRedirect.isPublished && (
                  <div className="flex justify-between">
                    <span className="text-slate-500 dark:text-slate-400">Published</span>
                    <span className="text-slate-900 dark:text-white">
                      <RelativeTime date={oldRedirect.publishedAt} />
                    </span>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Draft Dates (only when there's a draft) */}
          {draft && (
            <div className="rounded-xl bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700 p-5">
              <h3 className="text-sm font-semibold text-amber-800 dark:text-amber-300 mb-4 flex items-center gap-2">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                </svg>
                Draft Info
              </h3>
              <div className="space-y-3 text-sm">
                <div className="flex justify-between items-center">
                  <span className="text-amber-700 dark:text-amber-400">Status</span>
                  <DraftBadge changeType={changeType} />
                </div>
                <div className="flex justify-between">
                  <span className="text-amber-700 dark:text-amber-400">Created</span>
                  <span className="text-amber-900 dark:text-amber-200">
                    <RelativeTime date={draft.createdAt} />
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-amber-700 dark:text-amber-400">Updated</span>
                  <span className="text-amber-900 dark:text-amber-200">
                    <RelativeTime date={draft.updatedAt} />
                  </span>
                </div>
              </div>
            </div>
          )}

          {/* Priority Card */}
          <div className="rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 p-5">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-3">
              Redirect Priority
            </h3>
            <p className="text-sm text-slate-600 dark:text-slate-400 mb-3">
              Redirects are evaluated in this order:
            </p>
            <ol className="text-sm text-slate-600 dark:text-slate-400 space-y-1.5 list-decimal list-inside">
              <li><span className="font-medium text-slate-700 dark:text-slate-300">Host</span> - Exact host + path match</li>
              <li><span className="font-medium text-slate-700 dark:text-slate-300">Basic</span> - Exact path match</li>
              <li><span className="font-medium text-slate-700 dark:text-slate-300">Regex Host</span> - Regex with host</li>
              <li><span className="font-medium text-slate-700 dark:text-slate-300">Regex</span> - Regex path match</li>
            </ol>
          </div>

          {/* Help Card */}
          <div className="rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 p-5">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-3">
              About Drafts
            </h3>
            <p className="text-sm text-slate-600 dark:text-slate-400 leading-relaxed">
              Changes are saved as drafts and won't take effect until you publish them.
              You can review all pending changes before publishing.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
