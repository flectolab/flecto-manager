import { useState } from 'react'
import { useLazyQuery } from '@apollo/client/react'
import {
  CheckRedirectDocument,
  type RedirectBaseInput,
  type RedirectScope,
  type RedirectCheckResult,
} from '../../generated/graphql'

const MAX_URLS = 10

const allScopeLabels: Record<RedirectScope, string> = {
  SINGLE: 'Current Only',
  PROJECT: 'Project',
  PROJECT_WITH_DRAFT: 'Project + Drafts',
}

const allScopeDescriptions: Record<RedirectScope, string> = {
  SINGLE: 'Test only against the current redirect',
  PROJECT: 'Test against all published redirects in the project',
  PROJECT_WITH_DRAFT: 'Test against all redirects including drafts',
}

interface RedirectTesterProps {
  namespaceCode: string
  projectCode: string
  redirect?: RedirectBaseInput | null
  availableScopes?: RedirectScope[]
  defaultScope?: RedirectScope
  title?: string
  showCard?: boolean
}

type MatchStatus = 'not_matched' | 'matched_project' | 'matched_current'

interface UrlResult {
  url: string
  result?: RedirectCheckResult
  status?: MatchStatus
}

function getMatchStatus(
  result: RedirectCheckResult,
  currentRedirect: RedirectBaseInput | null
): MatchStatus {
  if (!result.matched) {
    return 'not_matched'
  }

  // Check if the matched redirect is the current one being edited
  if (
    currentRedirect &&
    result.redirectMatched &&
    result.redirectMatched.source === currentRedirect.source &&
    result.redirectMatched.type === currentRedirect.type &&
    result.redirectMatched.target === currentRedirect.target &&
    result.redirectMatched.status === currentRedirect.status
  ) {
    return 'matched_current'
  }

  return 'matched_project'
}

const DEFAULT_SCOPES: RedirectScope[] = ['SINGLE', 'PROJECT', 'PROJECT_WITH_DRAFT']

export function RedirectTester({
  namespaceCode,
  projectCode,
  redirect = null,
  availableScopes = DEFAULT_SCOPES,
  defaultScope = 'SINGLE',
  title = 'Test',
  showCard = true,
}: RedirectTesterProps) {
  const [urls, setUrls] = useState<UrlResult[]>([])
  const [scope, setScope] = useState<RedirectScope>(defaultScope ?? 'PROJECT')

  const [checkRedirect, { loading }] = useLazyQuery(CheckRedirectDocument, {
    fetchPolicy: 'network-only',
  })

  const handleAddUrl = () => {
    if (urls.length < MAX_URLS) {
      setUrls([...urls, { url: '' }])
    }
  }

  const handleRemoveUrl = (index: number) => {
    setUrls(urls.filter((_, i) => i !== index))
  }

  const handleUrlChange = (index: number, value: string) => {
    const newUrls = [...urls]
    newUrls[index] = { url: value }
    setUrls(newUrls)
  }

  const handleTest = async () => {
    const validUrls = urls.map((u) => u.url).filter((u) => u.trim() !== '')
    if (validUrls.length === 0) return

    try {
      const { data } = await checkRedirect({
        variables: {
          namespaceCode,
          projectCode,
          redirectCheck: {
            redirect: redirect,
            urls: validUrls,
          },
          scope,
        },
      })

      if (data?.projectRedirectDraftCheck) {
        const results = data.projectRedirectDraftCheck
        const newUrls = urls.map((u) => {
          const result = results.find((r) => r.url === u.url)
          if (result) {
            return {
              ...u,
              result,
              status: getMatchStatus(result, redirect),
            }
          }
          return u
        })
        setUrls(newUrls)
      }
    } catch (err) {
      console.error('Failed to check redirects:', err)
    }
  }

  const hasUrls = urls.length > 0
  const hasValidUrls = urls.some((u) => u.url.trim() !== '')

  const content = (
    <>
      <div className="flex items-center justify-between mb-4">
        {title && <h3 className="text-lg font-semibold text-slate-900 dark:text-white">{title}</h3>}
        <div className={`flex items-center gap-2 ${!title ? 'ml-auto' : ''}`}>
          {hasValidUrls && (
            <button
              type="button"
              onClick={handleTest}
              disabled={loading}
              className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-lg bg-cyan-500 text-white hover:bg-cyan-600 transition-colors disabled:opacity-50"
            >
              {loading ? (
                <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
              ) : (
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
                  />
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
              )}
              Test
            </button>
          )}
          {urls.length < MAX_URLS && (
            <button
              type="button"
              onClick={handleAddUrl}
              className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              Add URL
            </button>
          )}
        </div>
      </div>

      {/* Scope selector */}
      {hasUrls && availableScopes && availableScopes.length > 1 && (
        <div className="mb-4">
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Scope
          </label>
          <div className="flex gap-2">
            {availableScopes?.map((s) => (
              <button
                key={s}
                type="button"
                onClick={() => setScope(s)}
                className={`flex-1 px-3 py-2 text-sm font-medium rounded-lg border transition-colors ${
                  scope === s
                    ? 'border-cyan-500 bg-cyan-50 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-400 dark:border-cyan-600'
                    : 'border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700'
                }`}
                title={allScopeDescriptions[s]}
              >
                {allScopeLabels[s]}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* URL fields */}
      {hasUrls ? (
        <div className="space-y-3">
          {urls.map((urlResult, index) => (
            <UrlField
              key={index}
              urlResult={urlResult}
              index={index}
              onChange={handleUrlChange}
              onRemove={handleRemoveUrl}
            />
          ))}
        </div>
      ) : (
        <div className="text-center py-8 text-slate-500 dark:text-slate-400">
          <svg
            className="w-12 h-12 mx-auto mb-3 text-slate-300 dark:text-slate-600"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"
            />
          </svg>
          <p className="text-sm">Add URLs to test redirects</p>
        </div>
      )}
    </>
  )

  if (showCard) {
    return (
      <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
        {content}
      </div>
    )
  }

  return <>{content}</>
}

interface UrlFieldProps {
  urlResult: UrlResult
  index: number
  onChange: (index: number, value: string) => void
  onRemove: (index: number) => void
}

function UrlField({ urlResult, index, onChange, onRemove }: UrlFieldProps) {
  const { url, result, status } = urlResult

  return (
    <div className="flex items-center gap-2 flex-nowrap">
      {/* URL Input */}
      <input
        type="text"
        value={url}
        onChange={(e) => onChange(index, e.target.value)}
        placeholder="https://example.com/path or /path"
        className="flex-1 min-w-0 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-2 px-3 text-sm text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-cyan-500/20 focus:border-cyan-500 font-mono"
      />

      {/* Result indicator - fixed width */}
      <div className="w-[280px] flex-shrink-0 h-[38px] flex items-center">
        {result ? (
          <ResultIndicator status={status} result={result} />
        ) : (
          <div className="text-xs text-slate-400 dark:text-slate-500 italic">No result yet</div>
        )}
      </div>

      {/* Remove button */}
      <button
        type="button"
        onClick={() => onRemove(index)}
        className="flex-shrink-0 p-2 rounded-lg text-slate-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
        title="Remove URL"
      >
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  )
}

interface ResultIndicatorProps {
  status?: MatchStatus
  result: RedirectCheckResult
}

function ResultIndicator({ status, result }: ResultIndicatorProps) {
  if (status === 'not_matched') {
    return (
      <div className="flex items-center gap-2 group relative">
        <div className="flex items-center gap-1 px-2 py-1.5 rounded-lg bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400">
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
          <span className="text-xs font-medium">No match</span>
        </div>
        <Tooltip>Not matched</Tooltip>
      </div>
    )
  }

  if (status === 'matched_current') {
    return (
      <div className="flex items-center group relative w-full">
        <div className="flex items-center gap-1 px-2 py-1.5 rounded-lg bg-green-50 dark:bg-green-900/20 text-green-600 dark:text-green-400 w-full min-w-0">
          <svg className="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
          </svg>
          <svg className="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 5l7 7m0 0l-7 7m7-7H3" />
          </svg>
          <span className="text-xs font-mono truncate cursor-help" title={result.target}>
            {result.target}
          </span>
        </div>
        <TooltipWithDetails
          title="Matched Current"
          source={result.redirectMatched?.source}
          target={result.redirectMatched?.target}
          targetResult={result.target}
        />
      </div>
    )
  }

  // matched_project
  return (
    <div className="flex items-center group relative w-full">
      <div className="flex items-center gap-1 px-2 py-1.5 rounded-lg bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 w-full min-w-0">
        <svg className="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
        </svg>
        <svg className="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 5l7 7m0 0l-7 7m7-7H3" />
        </svg>
        <span className="text-xs font-mono truncate cursor-help" title={result.target}>
          {result.target}
        </span>
      </div>
      <TooltipWithDetails
        title="Matched By Project"
        source={result.redirectMatched?.source}
        target={result.redirectMatched?.target}
        targetResult={result.target}
      />
    </div>
  )
}

function Tooltip({ children }: { children: React.ReactNode }) {
  return (
    <div className="absolute left-1/2 -translate-x-1/2 bottom-full mb-2 px-2 py-1 text-xs font-medium text-white bg-slate-800 dark:bg-slate-600 rounded-md opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap pointer-events-none z-10">
      {children}
      <div className="absolute left-1/2 -translate-x-1/2 top-full w-0 h-0 border-l-4 border-r-4 border-t-4 border-l-transparent border-r-transparent border-t-slate-800 dark:border-t-slate-600" />
    </div>
  )
}

interface TooltipWithDetailsProps {
  title: string
  source?: string | null
  target?: string | null
  targetResult?: string | null
}

function TooltipWithDetails({ title, source, target, targetResult }: TooltipWithDetailsProps) {
  return (
    <div className="absolute left-0 bottom-full mb-2 px-3 py-2 text-xs bg-slate-800 dark:bg-slate-600 rounded-md opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-10 min-w-[250px]">
      <div className="font-medium text-white mb-1">{title}</div>
      {source && (
        <div className="text-slate-300">
          <span className="text-slate-400">Source:</span>{' '}
          <span className="font-mono">{source}</span>
        </div>
      )}
      {target && (
        <div className="text-slate-300">
          <span className="text-slate-400">Target:</span>{' '}
          <span className="font-mono">{target}</span>
        </div>
      )}
      {targetResult && (
        <div className="text-slate-300 mt-1 pt-1 border-t border-slate-700">
          <span className="text-slate-400">Target result:</span>{' '}
          <span className="font-mono text-cyan-300">{targetResult}</span>
        </div>
      )}
      <div className="absolute left-4 top-full w-0 h-0 border-l-4 border-r-4 border-t-4 border-l-transparent border-r-transparent border-t-slate-800 dark:border-t-slate-600" />
    </div>
  )
}
