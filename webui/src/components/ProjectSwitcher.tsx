import { useState, useRef, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@apollo/client/react'
import { SearchNamespacesDocument, GetNamespaceDocument, type SearchNamespacesQuery } from '../generated/graphql'
import { useCurrentProject } from '../hooks/useCurrentProject'

type Namespace = SearchNamespacesQuery['searchNamespaces']['items'][number]

const DROPDOWN_LIMIT = 10
const SEARCH_DEBOUNCE_MS = 300

interface ProjectSwitcherProps {
  collapsed: boolean
}

export function ProjectSwitcher({ collapsed }: ProjectSwitcherProps) {
  const navigate = useNavigate()
  const { project: currentProject, namespace: currentNamespace, namespaceCode, projectCode } = useCurrentProject()
  const [isOpen, setIsOpen] = useState(false)
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const dropdownRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const isSearching = search.trim().length > 0

  // Debounce search input
  useEffect(() => {
    const timer = setTimeout(() => {
      if (searchInput !== search) {
        setSearch(searchInput)
      }
    }, SEARCH_DEBOUNCE_MS)

    return () => clearTimeout(timer)
  }, [searchInput, search])

  // Query for current namespace only (when not searching)
  const { data: currentNamespaceData, loading: currentLoading } = useQuery(GetNamespaceDocument, {
    skip: !isOpen || isSearching || !namespaceCode,
    variables: { namespaceCode: namespaceCode! },
  })

  // Query for search across all namespaces (when searching)
  const { data: searchData, loading: searchLoading } = useQuery(SearchNamespacesDocument, {
    skip: !isOpen || !isSearching,
    variables: {
      pagination: { limit: DROPDOWN_LIMIT, offset: 0 },
      filter: { search: search },
    },
  })

  // Determine which data to show
  const loading = isSearching ? searchLoading : currentLoading
  const namespaces: Namespace[] = isSearching
    ? (searchData?.searchNamespaces.items ?? [])
    : currentNamespaceData?.namespace
      ? [{
          namespaceCode: currentNamespaceData.namespace.namespaceCode,
          name: currentNamespaceData.namespace.name,
          createdAt: currentNamespaceData.namespace.createdAt,
          updatedAt: currentNamespaceData.namespace.updatedAt,
          projects: currentNamespaceData.namespace.projects,
        }]
      : []
  const total = isSearching ? (searchData?.searchNamespaces.total ?? 0) : namespaces.length
  const hasMore = isSearching && total > DROPDOWN_LIMIT

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false)
        setSearchInput('')
        setSearch('')
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  useEffect(() => {
    if (isOpen && inputRef.current) {
      inputRef.current.focus()
    }
  }, [isOpen])

  const handleSelect = (nsCode: string, projCode: string) => {
    navigate(`/${nsCode}/${projCode}`)
    setIsOpen(false)
    setSearchInput('')
    setSearch('')
  }

  const handleClearSearch = () => {
    setSearchInput('')
    setSearch('')
  }

  if (collapsed) {
    return (
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-indigo/20 text-brand-purple"
      >
        <span className="text-sm font-bold">
          {currentProject?.name.charAt(0).toUpperCase() || 'P'}
        </span>
      </button>
    )
  }

  return (
    <div className="relative flex-1 min-w-0" ref={dropdownRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left hover:bg-slate-800 transition-colors"
      >
        <div className="flex-1 min-w-0">
          <p className="text-xs text-slate-400 truncate">{currentNamespace?.name || namespaceCode}</p>
          <p className="text-sm font-medium text-white truncate">{currentProject?.name || projectCode}</p>
        </div>
        <svg
          className={`shrink-0 h-4 w-4 text-slate-400 transition-transform ${isOpen ? 'rotate-180' : ''}`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {isOpen && (
        <div className="absolute left-0 top-full mt-2 w-72 rounded-xl bg-slate-800 border border-slate-700 shadow-xl z-50">
          <div className="p-2 border-b border-slate-700">
            <div className="relative">
              <svg
                className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400"
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
                ref={inputRef}
                type="text"
                placeholder="Search all projects..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                className="w-full rounded-lg bg-slate-900 border border-slate-600 py-2 pl-9 pr-8 text-sm text-white placeholder-slate-400 focus:border-brand-purple focus:outline-none"
              />
              {searchInput && (
                <button
                  onClick={handleClearSearch}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-300"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              )}
            </div>
          </div>

          <div className="max-h-80 overflow-y-auto p-2">
            {loading ? (
              <div className="flex items-center justify-center py-4">
                <svg className="h-5 w-5 animate-spin text-brand-purple" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
              </div>
            ) : namespaces.length === 0 ? (
              <p className="px-3 py-4 text-center text-sm text-slate-400">
                {isSearching ? 'No results found' : 'No projects available'}
              </p>
            ) : (
              <div className="space-y-2">
                {/* Show hint when not searching */}
                {!isSearching && (
                  <p className="px-3 py-1 text-xs text-slate-500">
                    Current namespace
                  </p>
                )}

                {namespaces.map((ns) => (
                  <div key={ns.namespaceCode}>
                    <p className="px-3 py-1 text-xs font-medium text-slate-400 uppercase tracking-wider">
                      {ns.name}
                    </p>
                    {ns.projects.length === 0 ? (
                      <p className="px-3 py-2 text-xs text-slate-500">No projects</p>
                    ) : (
                      <div className="mt-1 space-y-0.5">
                        {ns.projects.map((p) => {
                          const isCurrentProject = ns.namespaceCode === namespaceCode && p.projectCode === projectCode
                          return (
                            <button
                              key={`${ns.namespaceCode}-${p.projectCode}`}
                              onClick={() => handleSelect(ns.namespaceCode, p.projectCode)}
                              className={`flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left transition-colors ${
                                isCurrentProject
                                  ? 'bg-brand-indigo/20 text-brand-purple'
                                  : 'text-slate-300 hover:bg-slate-700'
                              }`}
                            >
                              <div className={`flex h-6 w-6 items-center justify-center rounded-md ${
                                isCurrentProject
                                  ? 'bg-brand-purple/20'
                                  : 'bg-slate-600'
                              }`}>
                                <svg
                                  className={`h-3.5 w-3.5 ${isCurrentProject ? 'text-brand-purple' : 'text-slate-300'}`}
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
                              <div className="flex-1 min-w-0">
                                <p className="text-sm font-medium truncate">{p.name}</p>
                                <p className="text-xs text-slate-500 truncate">{p.projectCode}</p>
                              </div>
                              {isCurrentProject && (
                                <svg className="h-4 w-4 text-brand-purple" fill="currentColor" viewBox="0 0 20 20">
                                  <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                                </svg>
                              )}
                            </button>
                          )
                        })}
                      </div>
                    )}
                  </div>
                ))}

                {hasMore && (
                  <p className="px-3 py-2 text-xs text-slate-500 text-center">
                    +{total - DROPDOWN_LIMIT} more results
                  </p>
                )}
              </div>
            )}
          </div>

          <div className="border-t border-slate-700 p-2">
            <button
              onClick={() => {
                navigate('/')
                setIsOpen(false)
              }}
              className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-slate-400 hover:bg-slate-700 hover:text-white transition-colors"
            >
              <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h7" />
              </svg>
              View all projects
            </button>
          </div>
        </div>
      )}
    </div>
  )
}