import { enabledOptions, redirectOptionDescriptions, type RedirectOptionsValue } from './redirectOptions'

const shortLabels = {
  ignoreQueryString: 'query string ignored',
  preserveQueryString: 'query string kept',
}

interface RedirectOptionsBadgesProps {
  options?: RedirectOptionsValue | null
  // emptyLabel is shown when no option is enabled; nothing is rendered without it.
  emptyLabel?: string
}

export function RedirectOptionsBadges({ options, emptyLabel }: RedirectOptionsBadgesProps) {
  const enabled = enabledOptions(options)

  if (enabled.length === 0) {
    return emptyLabel ? <span className="text-slate-400 dark:text-slate-500">{emptyLabel}</span> : null
  }

  return (
    <span className="inline-flex flex-wrap gap-1">
      {enabled.map((key) => (
        <span
          key={key}
          title={redirectOptionDescriptions[key]}
          className="inline-flex items-center whitespace-nowrap px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400"
        >
          {shortLabels[key]}
        </span>
      ))}
    </span>
  )
}
