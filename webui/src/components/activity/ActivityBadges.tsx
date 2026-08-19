import type { ActivityAction, ActivityResource } from '../../generated/graphql'

const actionStyles: Record<ActivityAction, { label: string; className: string }> = {
  CREATE: { label: 'Created', className: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' },
  UPDATE: { label: 'Updated', className: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400' },
  DELETE: { label: 'Deleted', className: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' },
  IMPORT: { label: 'Imported', className: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400' },
  ROLLBACK: { label: 'Discarded', className: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400' },
  PUBLISH: { label: 'Published', className: 'bg-brand-purple/10 text-brand-purple dark:bg-brand-purple/20' },
  TRUNCATE: { label: 'Truncated', className: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' },
}

const resourceLabels: Record<ActivityResource, string> = {
  REDIRECT: 'Redirect',
  PAGE: 'Page',
  PROJECT: 'Project',
  ACTIVITY: 'Activity',
}

export function ActivityActionBadge({ action }: { action: ActivityAction }) {
  // An action sent by a newer server still renders, as its raw value
  const style = actionStyles[action] ?? {
    label: action,
    className: 'bg-slate-100 text-slate-700 dark:bg-slate-700 dark:text-slate-300',
  }

  return (
    <span className={`inline-flex px-2 py-0.5 text-xs font-semibold rounded-full ${style.className}`}>
      {style.label}
    </span>
  )
}

export function ActivityResourceBadge({ resource }: { resource: ActivityResource }) {
  return (
    <span className="inline-flex px-2 py-0.5 text-xs font-medium rounded-md bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300">
      {resourceLabels[resource] ?? resource}
    </span>
  )
}

/** Actor of an event, with the API token case made visible. */
export function ActivityActor({ actor, authType }: { actor: string; authType: string }) {
  return (
    // The name truncates, never the badge: a token name can reach 300 characters
    // (model.TokenNameMaxLength), so the column cannot be widened enough to always
    // fit it. The full value stays reachable on hover.
    <span className="flex items-center gap-1.5 min-w-0">
      <span className="truncate text-slate-900 dark:text-white" title={actor}>
        {actor}
      </span>
      {authType === 'token' && (
        <span
          className="shrink-0 inline-flex px-1.5 py-0.5 text-[10px] font-semibold rounded bg-slate-200 text-slate-600 dark:bg-slate-600 dark:text-slate-300"
          title="Acted through an API token"
        >
          TOKEN
        </span>
      )}
    </span>
  )
}
