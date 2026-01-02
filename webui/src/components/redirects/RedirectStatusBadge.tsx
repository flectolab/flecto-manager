import type { RedirectStatus } from '../../generated/graphql'

interface RedirectStatusBadgeProps {
  status: RedirectStatus
}

const statusLabels: Record<RedirectStatus, string> = {
  MOVED_PERMANENT: '301',
  FOUND: '302',
  TEMPORARY_REDIRECT: '307',
  PERMANENT_REDIRECT: '308',
}

const statusColors: Record<RedirectStatus, string> = {
  MOVED_PERMANENT: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
  FOUND: 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400',
  TEMPORARY_REDIRECT: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400',
  PERMANENT_REDIRECT: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400',
}

export function RedirectStatusBadge({ status }: RedirectStatusBadgeProps) {
  return (
    <span
      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColors[status]}`}
      title={status.replace('_', ' ')}
    >
      {statusLabels[status]}
    </span>
  )
}