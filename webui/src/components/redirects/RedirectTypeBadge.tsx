import type { RedirectType } from '../../generated/graphql'

interface RedirectTypeBadgeProps {
  type: RedirectType
}

const typeLabels: Record<RedirectType, string> = {
  BASIC: 'Basic',
  BASIC_HOST: 'Host',
  REGEX: 'Regex',
  REGEX_HOST: 'Regex Host',
}

const typeColors: Record<RedirectType, string> = {
  BASIC: 'bg-slate-100 text-slate-800 dark:bg-slate-700 dark:text-slate-300',
  BASIC_HOST: 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-400',
  REGEX: 'bg-pink-100 text-pink-800 dark:bg-pink-900/30 dark:text-pink-400',
  REGEX_HOST: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400',
}

export function RedirectTypeBadge({ type }: RedirectTypeBadgeProps) {
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${typeColors[type]}`}>
      {typeLabels[type]}
    </span>
  )
}