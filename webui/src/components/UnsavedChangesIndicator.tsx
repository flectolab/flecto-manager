interface UnsavedChangesIndicatorProps {
  show: boolean
  className?: string
}

export function UnsavedChangesIndicator({ show, className = '' }: UnsavedChangesIndicatorProps) {
  if (!show) return null

  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400 ${className}`}
    >
      <svg
        className="w-3 h-3"
        fill="currentColor"
        viewBox="0 0 20 20"
      >
        <circle cx="10" cy="10" r="5" />
      </svg>
      Unsaved changes
    </span>
  )
}