import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@apollo/client/react'
import { SearchNamespacesDocument, type SearchNamespacesQuery } from '../generated/graphql'
import { useTheme } from '../contexts/ThemeContext'
import { useDocumentTitle } from '../hooks/useDocumentTitle'
import { UserDropdown } from '../components/UserDropdown'
import flectoFullLight from '../assets/flecto-full-light.svg'
import flectoFullDark from '../assets/flecto-full-dark.svg'

type Namespace = SearchNamespacesQuery['searchNamespaces']['items'][number]
type Project = Namespace['projects'][number]

const PAGE_SIZE = 10
const SEARCH_DEBOUNCE_MS = 300

export function NamespaceSelector() {
  useDocumentTitle('Select Project')
  const navigate = useNavigate()
  const { theme } = useTheme()
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(0)
  const [expandedNamespaces, setExpandedNamespaces] = useState<Set<string>>(new Set())

  // Debounce search input
  useEffect(() => {
    const timer = setTimeout(() => {
      if (searchInput !== search) {
        setSearch(searchInput)
        setPage(0) // Reset to first page when search changes
      }
    }, SEARCH_DEBOUNCE_MS)

    return () => clearTimeout(timer)
  }, [searchInput, search])

  const { data, loading, error } = useQuery(SearchNamespacesDocument, {
    variables: {
      pagination: { limit: PAGE_SIZE, offset: page * PAGE_SIZE },
      filter: { search: search || null },
    },
  })

  const namespaces = data?.searchNamespaces.items ?? []
  const total = data?.searchNamespaces.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)

  const toggleNamespace = (code: string) => {
    setExpandedNamespaces((prev) => {
      const next = new Set(prev)
      if (next.has(code)) {
        next.delete(code)
      } else {
        next.add(code)
      }
      return next
    })
  }

  const handleProjectSelect = (namespace: Namespace, project: Project) => {
    navigate(`/${namespace.namespaceCode}/${project.projectCode}`)
  }

  const handleClearSearch = () => {
    setSearchInput('')
    setSearch('')
    setPage(0)
  }

  // Auto-expand namespaces when searching
  const isExpanded = (code: string) => {
    if (search.trim()) return true
    return expandedNamespaces.has(code)
  }

  return (
    <div className="min-h-screen bg-slate-100 dark:bg-slate-950">
      {/* User header */}
      <div className="fixed top-0 left-0 right-0 z-30 h-16 bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <div className="mx-auto max-w-4xl h-full flex items-center justify-between px-4">
          <img
            src={theme === 'dark' ? flectoFullDark : flectoFullLight}
            alt="Flecto Labs"
            className="h-8"
          />
          <UserDropdown />
        </div>
      </div>

      <div className="mx-auto max-w-4xl px-4 py-12 pt-24">
        <div className="mb-12 text-center">
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white mb-2">
            Select a Project
          </h1>
          <p className="text-slate-500 dark:text-slate-400">
            Choose a namespace and project to manage
          </p>
        </div>

        <div className="mb-8">
          <div className="relative">
            <svg
              className="absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-slate-400"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
              />
            </svg>
            <input
              type="text"
              placeholder="Search namespaces or projects..."
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              className="w-full rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 py-4 pl-12 pr-12 text-slate-900 dark:text-white placeholder-slate-400 shadow-sm focus:border-brand-purple focus:outline-none focus:ring-2 focus:ring-brand-purple/20"
            />
            {searchInput && (
              <button
                onClick={handleClearSearch}
                className="absolute right-4 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            )}
          </div>
        </div>

        {loading ? (
          <div className="rounded-xl bg-white dark:bg-slate-800 p-12 text-center shadow-sm border border-slate-200 dark:border-slate-700">
            <svg
              className="mx-auto h-8 w-8 animate-spin text-brand-purple"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                className="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="4"
              />
              <path
                className="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              />
            </svg>
            <p className="mt-4 text-slate-500 dark:text-slate-400">Loading namespaces...</p>
          </div>
        ) : error ? (
          <div className="rounded-xl bg-red-50 dark:bg-red-900/20 p-12 text-center shadow-sm border border-red-200 dark:border-red-800">
            <svg
              className="mx-auto h-12 w-12 text-red-400"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
              />
            </svg>
            <p className="mt-4 text-red-600 dark:text-red-400">Failed to load namespaces</p>
            <p className="mt-2 text-sm text-red-500 dark:text-red-500">{error.message}</p>
          </div>
        ) : namespaces.length === 0 ? (
          <div className="rounded-xl bg-white dark:bg-slate-800 p-12 text-center shadow-sm border border-slate-200 dark:border-slate-700">
            <svg
              className="mx-auto h-12 w-12 text-slate-300 dark:text-slate-600"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
            <p className="mt-4 text-slate-500 dark:text-slate-400">
              {search ? `No results found for "${search}"` : 'No namespaces available'}
            </p>
          </div>
        ) : (
          <>
            <div className="space-y-4">
              {namespaces.map((namespace) => (
                <div
                  key={namespace.namespaceCode}
                  className="rounded-xl bg-white dark:bg-slate-800 shadow-sm border border-slate-200 dark:border-slate-700 overflow-hidden"
                >
                  <button
                    onClick={() => toggleNamespace(namespace.namespaceCode)}
                    className="flex w-full items-center justify-between px-6 py-4 text-left hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
                  >
                    <div className="flex items-center gap-4">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gradient-to-br from-brand-purple to-brand-indigo">
                        <span className="text-sm font-bold text-white">
                          {namespace.name.charAt(0).toUpperCase()}
                        </span>
                      </div>
                      <div>
                        <h3 className="font-semibold text-slate-900 dark:text-white">
                          {namespace.name}
                        </h3>
                        <p className="text-sm text-slate-500 dark:text-slate-400">
                          {namespace.namespaceCode} · {namespace.projects.length} project
                          {namespace.projects.length !== 1 ? 's' : ''}
                        </p>
                      </div>
                    </div>
                    <svg
                      className={`h-5 w-5 text-slate-400 transition-transform ${
                        isExpanded(namespace.namespaceCode) ? 'rotate-180' : ''
                      }`}
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M19 9l-7 7-7-7"
                      />
                    </svg>
                  </button>

                  {isExpanded(namespace.namespaceCode) && (
                    <div className="border-t border-slate-100 dark:border-slate-700 bg-slate-50/50 dark:bg-slate-900/50">
                      {namespace.projects.length === 0 ? (
                        <p className="px-6 py-4 text-sm text-slate-500 dark:text-slate-400">
                          No projects in this namespace
                        </p>
                      ) : (
                        <div className="divide-y divide-slate-100 dark:divide-slate-700">
                          {namespace.projects.map((project) => (
                            <button
                              key={project.projectCode}
                              onClick={() => handleProjectSelect(namespace, project)}
                              className="flex w-full items-center gap-4 px-6 py-3 text-left hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                            >
                              <div className="flex h-8 w-8 items-center justify-center rounded-md bg-gradient-to-br from-brand-cyan to-brand-sky">
                                <svg
                                  className="h-4 w-4 text-white"
                                  fill="none"
                                  stroke="currentColor"
                                  viewBox="0 0 24 24"
                                >
                                  <path
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                    strokeWidth={2}
                                    d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"
                                  />
                                </svg>
                              </div>
                              <div className="flex-1">
                                <p className="font-medium text-slate-900 dark:text-white">
                                  {project.name}
                                </p>
                                <p className="text-sm text-slate-500 dark:text-slate-400">{project.projectCode}</p>
                              </div>
                              <svg
                                className="h-5 w-5 text-slate-300 dark:text-slate-600"
                                fill="none"
                                stroke="currentColor"
                                viewBox="0 0 24 24"
                              >
                                <path
                                  strokeLinecap="round"
                                  strokeLinejoin="round"
                                  strokeWidth={2}
                                  d="M9 5l7 7-7 7"
                                />
                              </svg>
                            </button>
                          ))}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="mt-6 flex items-center justify-between">
                <span className="text-sm text-slate-600 dark:text-slate-400">
                  Showing {page * PAGE_SIZE + 1} - {Math.min((page + 1) * PAGE_SIZE, total)} of {total} namespaces
                </span>
                <div className="flex gap-2">
                  <button
                    onClick={() => setPage(Math.max(0, page - 1))}
                    disabled={page === 0}
                    className="px-3 py-1.5 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    Previous
                  </button>
                  <button
                    onClick={() => setPage(Math.min(totalPages - 1, page + 1))}
                    disabled={page >= totalPages - 1}
                    className="px-3 py-1.5 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    Next
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}