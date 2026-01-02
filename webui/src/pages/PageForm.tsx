import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation } from '@apollo/client/react'
import { useDocumentTitle } from '../hooks/useDocumentTitle'
import {
  GetProjectPageDocument,
  GetProjectPagesDocument,
  GetProjectDocument,
  CreatePageDraftDocument,
  UpdatePageDraftDocument,
  type PageType,
  type PageContentType,
} from '../generated/graphql'
import { useCurrentProject } from '../hooks/useCurrentProject'
import { usePermissions, Action, ResourceType } from '../hooks/usePermissions'
import { RelativeTime } from '../components/RelativeTime'
import { DraftBadge } from '../components/redirects'
import { formatSize } from '../utils/format'

interface FormData {
  type: PageType
  path: string
  content: string
  contentType: PageContentType
}

const pageTypes: { value: PageType; label: string; description: string }[] = [
  { value: 'BASIC', label: 'Basic', description: 'Simple path matching' },
  { value: 'BASIC_HOST', label: 'Host', description: 'Match with host header' },
]

const contentTypes: { value: PageContentType; label: string; mimeType: string }[] = [
  { value: 'TEXT_PLAIN', label: 'Text', mimeType: 'text/plain' },
  { value: 'XML', label: 'XML', mimeType: 'application/xml' },
]

const pathPlaceholders: Record<PageType, string> = {
  BASIC: '/robots.txt or /sitemap.xml',
  BASIC_HOST: 'example.com/robots.txt',
}

function isValidPath(path: string): boolean {
  if (!path.startsWith('/')) return false
  try {
    const url = new URL(path, 'http://localhost')
    return url.pathname === path.split('?')[0]
  } catch {
    return false
  }
}

function isValidHostPath(path: string): boolean {
  if (path.startsWith('/')) return false

  try {
    const normalized = '//' + path
    const url = new URL(normalized, 'http://localhost')
    return url.host !== '' && url.pathname.length > 1
  } catch {
    return false
  }
}

function PageTypeBadge({ type }: { type: PageType }) {
  const labels: Record<PageType, string> = {
    BASIC: 'Basic',
    BASIC_HOST: 'Host',
  }
  return (
    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-700 dark:bg-slate-700 dark:text-slate-300">
      {labels[type]}
    </span>
  )
}

function ContentTypeBadge({ contentType }: { contentType: PageContentType }) {
  const labels: Record<PageContentType, string> = {
    TEXT_PLAIN: 'Text',
    XML: 'XML',
  }
  const colors: Record<PageContentType, string> = {
    TEXT_PLAIN: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
    XML: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400',
  }

  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${colors[contentType]}`}>
      {labels[contentType]}
    </span>
  )
}

export function PageForm() {
  const { namespace, project, id: pageId } = useParams()
  const navigate = useNavigate()
  const { namespaceCode, projectCode, project: currentProject } = useCurrentProject()
  const { canResource, loading: permissionsLoading } = usePermissions()

  const isCreateMode = !pageId
  const isEditMode = !!pageId

  const pageTitle = isCreateMode ? 'Add Page' : 'Edit Page'
  useDocumentTitle(namespaceCode && projectCode ? `${pageTitle} - ${namespaceCode}/${projectCode}` : pageTitle)

  const canWrite = namespaceCode && projectCode ? canResource(namespaceCode, projectCode, ResourceType.Page, Action.Write) : false

  const { data: pageData, loading: pageLoading } = useQuery(GetProjectPageDocument, {
    variables: {
      namespaceCode: namespaceCode ?? '',
      projectCode: projectCode ?? '',
      pageID: pageId ?? '',
    },
    skip: !namespaceCode || !projectCode || !pageId,
  })

  const [formData, setFormData] = useState<FormData>({
    type: 'BASIC',
    path: '',
    content: '',
    contentType: 'TEXT_PLAIN',
  })

  const [errors, setErrors] = useState<Partial<Record<keyof FormData, string>>>({})
  const [submitError, setSubmitError] = useState<string | null>(null)

  const [createDraft, { loading: createLoading }] = useMutation(CreatePageDraftDocument, {
    refetchQueries: [GetProjectPagesDocument, GetProjectDocument],
  })

  const [updateDraft, { loading: updateLoading }] = useMutation(UpdatePageDraftDocument, {
    refetchQueries: [GetProjectPagesDocument, GetProjectDocument],
  })

  useEffect(() => {
    if (isEditMode && pageData?.projectPage) {
      const page = pageData.projectPage
      if (page.pageDraft?.newPage) {
        setFormData({
          type: page.pageDraft.newPage.type,
          path: page.pageDraft.newPage.path,
          content: page.pageDraft.newPage.content,
          contentType: page.pageDraft.newPage.contentType,
        })
      } else {
        setFormData({
          type: page.type,
          path: page.path ?? '',
          content: page.content ?? '',
          contentType: page.contentType ?? 'TEXT_PLAIN',
        })
      }
    }
  }, [isEditMode, pageData])

  const validate = (): boolean => {
    const newErrors: Partial<Record<keyof FormData, string>> = {}

    if (!formData.path.trim()) {
      newErrors.path = 'Path is required'
    } else {
      switch (formData.type) {
        case 'BASIC':
          if (!isValidPath(formData.path)) {
            newErrors.path = 'Path must be a valid path starting with /'
          }
          break
        case 'BASIC_HOST':
          if (!isValidHostPath(formData.path)) {
            newErrors.path = 'Path must include a valid host and path (e.g., example.com/robots.txt)'
          }
          break
      }
    }

    if (!formData.content.trim()) {
      newErrors.content = 'Content is required'
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
        await createDraft({
          variables: {
            namespaceCode,
            projectCode,
            input: {
              newPage: formData,
            },
          },
        })
      } else if (isEditMode && pageId) {
        const page = pageData?.projectPage
        if (page?.pageDraft) {
          await updateDraft({
            variables: {
              namespaceCode,
              projectCode,
              pageDraftID: page.pageDraft.id,
              input: {
                newPage: formData,
              },
            },
          })
        } else {
          await createDraft({
            variables: {
              namespaceCode,
              projectCode,
              input: {
                oldPageID: pageId,
                newPage: formData,
              },
            },
          })
        }
      }

      navigate(`/${namespace}/${project}/pages`)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'An unexpected error occurred'
      setSubmitError(message)
    }
  }

  const handleCancel = () => {
    navigate(`/${namespace}/${project}/pages`)
  }

  const handleChange = (field: keyof FormData, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }))
    if (errors[field]) {
      setErrors((prev) => ({ ...prev, [field]: undefined }))
    }
  }

  const isLoading = pageLoading || permissionsLoading
  const isSaving = createLoading || updateLoading

  const page = pageData?.projectPage
  const draft = page?.pageDraft
  const changeType = draft?.changeType
  const oldPage = page

  const hasChanged = (field: keyof FormData): boolean => {
    if (!oldPage || changeType === 'CREATE') return false
    const oldValue = oldPage[field] ?? ''
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
            <p className="text-amber-700 dark:text-amber-400">You don't have permission to modify pages for this project.</p>
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
            {isCreateMode ? 'Add Page' : 'Edit Page'}
          </h2>
          {changeType && <DraftBadge changeType={changeType} />}
        </div>
        <p className="text-slate-600 dark:text-slate-400 ml-12">
          {isCreateMode
            ? 'Create a new static page (robots.txt, sitemap.xml, etc.)'
            : 'Modify the page content. Changes will be saved as a draft.'}
        </p>
      </div>

      <div className="flex flex-col lg:flex-row gap-6">
        {/* Form */}
        <div className="flex-1 min-w-0">
          <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
            <form onSubmit={handleSubmit} className="space-y-6">
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
                  {pageTypes.map((type) => (
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
                {hasChanged('type') && oldPage && (
                  <p className="mt-2 text-xs text-amber-600 dark:text-amber-400 flex items-center gap-1">
                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01" />
                    </svg>
                    Changed from <PageTypeBadge type={oldPage.type} />
                  </p>
                )}
              </div>

              {/* Path */}
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  Path
                </label>
                <input
                  type="text"
                  value={formData.path}
                  onChange={(e) => handleChange('path', e.target.value)}
                  placeholder={pathPlaceholders[formData.type]}
                  className={`w-full rounded-lg border bg-white dark:bg-slate-900 py-2.5 px-4 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-brand-purple/20 font-mono ${
                    errors.path
                      ? 'border-red-500 focus:border-red-500'
                      : 'border-slate-200 dark:border-slate-700 focus:border-brand-purple'
                  }`}
                />
                {errors.path && <p className="mt-1 text-sm text-red-500">{errors.path}</p>}
                {hasChanged('path') && oldPage && (
                  <p className="mt-2 text-xs text-amber-600 dark:text-amber-400 flex items-center gap-1">
                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01" />
                    </svg>
                    Changed from <code className="font-mono bg-slate-100 dark:bg-slate-700 px-1 rounded">{oldPage.path}</code>
                  </p>
                )}
              </div>

              {/* Content Type */}
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  Content Type
                </label>
                <div className="flex gap-2">
                  {contentTypes.map((ct) => (
                    <button
                      key={ct.value}
                      type="button"
                      onClick={() => handleChange('contentType', ct.value)}
                      className={`px-4 py-2.5 text-sm font-medium rounded-lg border transition-colors ${
                        formData.contentType === ct.value
                          ? 'border-brand-purple bg-brand-purple/10 text-brand-purple'
                          : 'border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700'
                      }`}
                      title={ct.mimeType}
                    >
                      {ct.label}
                    </button>
                  ))}
                </div>
                {hasChanged('contentType') && oldPage?.contentType && (
                  <p className="mt-2 text-xs text-amber-600 dark:text-amber-400 flex items-center gap-1">
                    <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01" />
                    </svg>
                    Changed from <ContentTypeBadge contentType={oldPage.contentType} />
                  </p>
                )}
              </div>

              {/* Content */}
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  Content
                </label>
                <textarea
                  value={formData.content}
                  onChange={(e) => handleChange('content', e.target.value)}
                  placeholder={formData.contentType === 'XML' ? '<?xml version="1.0"?>\n<urlset>...</urlset>' : 'User-agent: *\nDisallow: /admin/'}
                  rows={15}
                  className={`w-full rounded-lg border bg-white dark:bg-slate-900 py-2.5 px-4 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-brand-purple/20 font-mono text-sm resize-y ${
                    errors.content
                      ? 'border-red-500 focus:border-red-500'
                      : 'border-slate-200 dark:border-slate-700 focus:border-brand-purple'
                  }`}
                />
                {errors.content && <p className="mt-1 text-sm text-red-500">{errors.content}</p>}
                <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                  {formData.content.length} characters
                </p>
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
        </div>

        {/* Sidebar */}
        <div className="lg:w-[320px] flex-shrink-0 space-y-5">
          {/* Storage Usage */}
          {currentProject && (
            <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-5">
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
                <svg className="w-4 h-4 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
                </svg>
                Total Page Size
              </h3>
              <div className="space-y-3">
                <div className="flex justify-between items-center text-sm">
                  <span className="text-slate-500 dark:text-slate-400">Used</span>
                  <span className="text-slate-900 dark:text-white font-medium">
                    {formatSize(Number(currentProject.totalPageContentSize))} / {formatSize(Number(currentProject.totalPageContentSizeLimit))}
                  </span>
                </div>
                <div className="w-full bg-slate-200 dark:bg-slate-700 rounded-full h-2">
                  <div
                    className="bg-gradient-to-r from-brand-purple to-brand-indigo h-2 rounded-full transition-all"
                    style={{
                      width: `${Math.min(100, (Number(currentProject.totalPageContentSize) / Number(currentProject.totalPageContentSizeLimit)) * 100)}%`,
                    }}
                  />
                </div>
              </div>
            </div>
          )}

          {/* Old Values Card (only for UPDATE mode) */}
          {!isCreateMode && oldPage && changeType !== 'CREATE' && (
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
                  <PageTypeBadge type={oldPage.type} />
                </div>
                <div className="flex justify-between items-center gap-2 min-w-0">
                  <span className="text-slate-500 dark:text-slate-400 flex-shrink-0">Path</span>
                  <span
                    className="font-mono text-slate-900 dark:text-white truncate min-w-0 cursor-help hover:text-cyan-600 dark:hover:text-cyan-400 transition-colors"
                    title={oldPage.path ?? ''}
                  >
                    {oldPage.path}
                  </span>
                </div>
                {oldPage.contentType && (
                  <div className="flex justify-between items-center">
                    <span className="text-slate-500 dark:text-slate-400">Content Type</span>
                    <ContentTypeBadge contentType={oldPage.contentType} />
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Page Dates (only for existing pages) */}
          {!isCreateMode && oldPage && 'createdAt' in oldPage && (
            <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-5">
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
                <svg className="w-4 h-4 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
                Page Info
              </h3>
              <div className="space-y-3 text-sm">
                <div className="flex justify-between">
                  <span className="text-slate-500 dark:text-slate-400">Created</span>
                  <span className="text-slate-900 dark:text-white">
                    <RelativeTime date={oldPage.createdAt} />
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-500 dark:text-slate-400">Updated</span>
                  <span className="text-slate-900 dark:text-white">
                    <RelativeTime date={oldPage.updatedAt} />
                  </span>
                </div>
                {oldPage.isPublished && (
                  <div className="flex justify-between">
                    <span className="text-slate-500 dark:text-slate-400">Published</span>
                    <span className="text-slate-900 dark:text-white">
                      <RelativeTime date={oldPage.publishedAt} />
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

          {/* Help Card */}
          <div className="rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 p-5">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-3">
              About Pages
            </h3>
            <p className="text-sm text-slate-600 dark:text-slate-400 leading-relaxed mb-3">
              Static pages serve content like robots.txt, sitemap.xml, or other fixed content files.
            </p>
            <ul className="text-sm text-slate-600 dark:text-slate-400 space-y-1.5">
              <li><span className="font-medium text-slate-700 dark:text-slate-300">Basic</span> - Match by path only</li>
              <li><span className="font-medium text-slate-700 dark:text-slate-300">Host</span> - Match by host + path</li>
            </ul>
          </div>

          {/* About Drafts */}
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
