import type { DraftChangeType } from '../../generated/graphql'

interface DraftBadgeProps {
  changeType?: DraftChangeType | null
}

type StatusType = DraftChangeType | 'PUBLISHED'

const changeTypeLabels: Record<StatusType, string> = {
  CREATE: 'New',
  UPDATE: 'Modified',
  DELETE: 'Deleted',
  PUBLISHED: 'Published',
}

const changeTypeColors: Record<StatusType, string> = {
  CREATE: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400',
  UPDATE: 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400',
  DELETE: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
  PUBLISHED: 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-400',
}

export function DraftBadge({ changeType }: DraftBadgeProps) {
  const status: StatusType = changeType ?? 'PUBLISHED'

  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${changeTypeColors[status]}`}>
      {changeTypeLabels[status]}
    </span>
  )
}
