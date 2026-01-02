import { useState, useEffect, type ReactNode, type KeyboardEvent } from 'react'

interface SearchBarProps {
  placeholder: string
  value: string
  onSearch: (value: string) => void
  onClear?: () => void
  leftSlot?: ReactNode
  children?: ReactNode
}

export function SearchBar({ placeholder, value, onSearch, onClear, leftSlot, children }: SearchBarProps) {
  const [inputValue, setInputValue] = useState(value)

  // Sync local state when external value changes (e.g., back/forward navigation)
  useEffect(() => {
    setInputValue(value)
  }, [value])

  const handleSearch = () => {
    onSearch(inputValue)
  }

  const handleClear = () => {
    setInputValue('')
    if (onClear) {
      onClear()
    } else {
      onSearch('')
    }
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      handleSearch()
    }
  }

  return (
    <div className="mb-4 flex flex-col sm:flex-row gap-4">
      <div className="relative flex-1 flex gap-2">
        {leftSlot}
        <div className="relative flex-1">
          <svg
            className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            />
          </svg>
          <input
            type="text"
            placeholder={placeholder}
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleKeyDown}
            className="w-full pl-10 pr-10 py-2 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-brand-purple focus:border-transparent"
          />
          {inputValue && (
            <button
              onClick={handleClear}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
              title="Clear search"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          )}
        </div>
        <button
          onClick={handleSearch}
          className="px-4 py-2 text-sm font-medium rounded-lg bg-brand-purple text-white hover:bg-brand-purple/90 transition-colors flex items-center gap-2"
          title="Search"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            />
          </svg>
          <span className="hidden sm:inline">Search</span>
        </button>
      </div>
      {children}
    </div>
  )
}