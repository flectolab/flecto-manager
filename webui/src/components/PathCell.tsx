import { useState } from 'react'

interface PathCellProps {
  value: string
  maxWidth?: string
  className?: string
}

export function PathCell({ value, maxWidth = 'max-w-xs', className = '' }: PathCellProps) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async (e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch (err) {
      console.error('Failed to copy:', err)
    }
  }

  if (!value) {
    return <span className="text-slate-400 dark:text-slate-500">-</span>
  }

  return (
    <div className={`flex items-center gap-2 ${maxWidth}`}>
      <button
        onClick={handleCopy}
        className="flex-shrink-0 p-1 rounded text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
        title={copied ? 'Copied!' : 'Copy to clipboard'}
      >
        {copied ? (
          <svg className="w-4 h-4 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
          </svg>
        ) : (
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
          </svg>
        )}
      </button>
      <span
        className={`font-mono text-sm truncate min-w-0 cursor-help ${className}`}
        title={value}
      >
        {value}
      </span>
    </div>
  )
}