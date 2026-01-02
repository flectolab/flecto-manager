
import { useMemo, useState } from 'react'

interface RelativeTimeProps {
  date: string | Date
  className?: string
}

function formatRelativeTime(date: Date): string {
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffSeconds = Math.floor(diffMs / 1000)
  const diffMinutes = Math.floor(diffSeconds / 60)
  const diffHours = Math.floor(diffMinutes / 60)
  const diffDays = Math.floor(diffHours / 24)

  // Less than 24 hours: show relative time
  if (diffDays < 1) {
    if (diffSeconds < 60) {
      return 'just now'
    }
    if (diffMinutes < 60) {
      return `${diffMinutes} minute${diffMinutes > 1 ? 's' : ''} ago`
    }
    return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`
  }

  // Otherwise show the date
  return date.toLocaleDateString()
}

function formatFullDateTime(date: Date): string {
  return date.toLocaleString(undefined, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

export function RelativeTime({ date, className }: RelativeTimeProps) {
  const [showTooltip, setShowTooltip] = useState(false)
  const dateObj = useMemo(() => (typeof date === 'string' ? new Date(date) : date), [date])

  const relativeTime = useMemo(() => formatRelativeTime(dateObj), [dateObj])
  const fullDateTime = useMemo(() => formatFullDateTime(dateObj), [dateObj])

  return (
    <span
      className={`relative inline-block ${className ?? ''}`}
      onMouseEnter={() => setShowTooltip(true)}
      onMouseLeave={() => setShowTooltip(false)}
    >
      {relativeTime}
      {showTooltip && (
        <span className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-2 py-1 text-xs font-medium text-white bg-slate-900 dark:bg-slate-700 rounded shadow-lg whitespace-nowrap z-50">
          {fullDateTime}
          <span className="absolute top-full left-1/2 -translate-x-1/2 border-4 border-transparent border-t-slate-900 dark:border-t-slate-700" />
        </span>
      )}
    </span>
  )
}