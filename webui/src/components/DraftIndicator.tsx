import { useState, useRef, useEffect } from 'react'

interface DraftIndicatorProps {
  redirectDraftCount: number
  pageDraftCount: number
}

export function DraftIndicator({ redirectDraftCount, pageDraftCount }: DraftIndicatorProps) {
  const [showTooltip, setShowTooltip] = useState(false)
  const tooltipRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)

  const totalCount = redirectDraftCount + pageDraftCount

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        tooltipRef.current &&
        buttonRef.current &&
        !tooltipRef.current.contains(event.target as Node) &&
        !buttonRef.current.contains(event.target as Node)
      ) {
        setShowTooltip(false)
      }
    }

    if (showTooltip) {
      document.addEventListener('mousedown', handleClickOutside)
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [showTooltip])

  if (totalCount === 0) return null

  return (
    <div className="relative">
      <button
        ref={buttonRef}
        onClick={() => setShowTooltip(!showTooltip)}
        className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-amber-100 dark:bg-amber-900/30 text-amber-800 dark:text-amber-400 hover:bg-amber-200 dark:hover:bg-amber-900/50 transition-colors"
        title={`${totalCount} unpublished change${totalCount > 1 ? 's' : ''}`}
      >
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
          />
        </svg>
        <span className="text-sm font-medium">
          {totalCount} draft{totalCount > 1 ? 's' : ''}
        </span>
      </button>

      {showTooltip && (
        <div
          ref={tooltipRef}
          className="absolute right-0 top-full mt-2 w-48 rounded-lg bg-white dark:bg-slate-800 shadow-lg border border-slate-200 dark:border-slate-700 py-2 z-50"
        >
          <div className="px-3 py-1.5 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
            Unpublished Changes
          </div>
          <div className="px-3 py-2 flex items-center justify-between text-sm">
            <span className="text-slate-700 dark:text-slate-300">Redirects</span>
            <span className="font-medium text-amber-600 dark:text-amber-400">{redirectDraftCount}</span>
          </div>
          <div className="px-3 py-2 flex items-center justify-between text-sm">
            <span className="text-slate-700 dark:text-slate-300">Pages</span>
            <span className="font-medium text-amber-600 dark:text-amber-400">{pageDraftCount}</span>
          </div>
        </div>
      )}
    </div>
  )
}