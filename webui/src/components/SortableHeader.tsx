import type { SortInput } from '../generated/graphql'

interface SortableHeaderProps {
  label: string
  column: string
  currentSorts: SortInput[]
  onSort: (column: string) => void
}

export function SortableHeader({ label, column, currentSorts, onSort }: SortableHeaderProps) {
  const sortIndex = currentSorts.findIndex(s => s.column === column)
  const isActive = sortIndex !== -1
  const direction = isActive ? currentSorts[sortIndex].direction : null

  return (
    <button
      onClick={() => onSort(column)}
      className="flex items-center gap-1 text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider hover:text-slate-700 dark:hover:text-slate-200 transition-colors"
    >
      {label}
      {isActive && currentSorts.length > 1 && (
        <span className="text-[10px] text-brand-purple font-bold">{sortIndex + 1}</span>
      )}
      <span className="flex flex-col">
        <svg
          className={`w-3 h-3 -mb-1 ${direction === 'ASC' ? 'text-brand-purple' : 'text-slate-300 dark:text-slate-600'}`}
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path d="M5 10l5-5 5 5H5z" />
        </svg>
        <svg
          className={`w-3 h-3 ${direction === 'DESC' ? 'text-brand-purple' : 'text-slate-300 dark:text-slate-600'}`}
          fill="currentColor"
          viewBox="0 0 20 20"
        >
          <path d="M5 10l5 5 5-5H5z" />
        </svg>
      </span>
    </button>
  )
}

// Helper hook for managing sort state from URL params
export function parseSortsFromUrl(sortParam: string | null): SortInput[] {
  if (!sortParam) return []
  return sortParam.split(',').map(s => {
    const [column, direction] = s.split(':')
    return { column, direction: (direction?.toUpperCase() || 'ASC') as 'ASC' | 'DESC' }
  })
}

export function serializeSortsToUrl(sorts: SortInput[]): string {
  return sorts.map(s => `${s.column}:${s.direction}`).join(',')
}
