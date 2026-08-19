import type { ActivityAction, ActivityResource } from '../../generated/graphql'
import type { ActivityDescription } from './describeActivityEvent'

interface ActivityDetailModalProps {
  resource: ActivityResource
  action: ActivityAction
  description: ActivityDescription
  onClose: () => void
}

function FieldRow({ label, before, after }: { label: string; before?: string; after: string }) {
  const hasChanged = before !== undefined

  return (
    <tr className={hasChanged ? 'bg-amber-50/50 dark:bg-amber-900/10' : ''}>
      <td className="px-4 py-3 text-sm font-medium text-slate-700 dark:text-slate-300 w-32 align-top">
        {label}
      </td>
      <td className="px-4 py-3 align-top">
        {hasChanged ? (
          <div className="text-sm font-mono line-through text-slate-400 dark:text-slate-500 break-all">
            {before}
          </div>
        ) : (
          <div className="text-sm text-slate-400 dark:text-slate-500">—</div>
        )}
      </td>
      <td className="px-4 py-3 align-top">
        <div
          className={`text-sm font-mono break-all ${
            hasChanged
              ? 'text-amber-700 dark:text-amber-400 font-medium'
              : 'text-slate-600 dark:text-slate-400'
          }`}
        >
          {after}
        </div>
      </td>
    </tr>
  )
}

/**
 * Full detail of one event, opened from the table when the one-line summary cannot
 * hold everything. Same shape as the redirect DiffModal so both read alike.
 */
export function ActivityDetailModal({ resource, action, description, onClose }: ActivityDetailModalProps) {
  const errors = description.errors ?? []
  const truncatedErrors = (description.errorCount ?? 0) > errors.length

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={onClose} />
      <div className="relative w-full max-w-3xl mx-4 max-h-[85vh] flex flex-col rounded-xl bg-white dark:bg-slate-800 shadow-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
        <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white flex items-center gap-2">
            <svg className="w-5 h-5 text-amber-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
            {resource} · {action}
          </h3>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="overflow-y-auto">
          <table className="w-full">
            <thead className="bg-slate-50 dark:bg-slate-700/50">
              <tr>
                <th className="px-4 py-2 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  Field
                </th>
                <th className="px-4 py-2 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  Before
                </th>
                <th className="px-4 py-2 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  After
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
              {description.fields.map((field) => (
                <FieldRow
                  key={field.label}
                  label={field.label}
                  before={field.before}
                  after={field.after}
                />
              ))}
            </tbody>
          </table>

          {errors.length > 0 && (
            <div className="px-4 py-4 border-t border-slate-200 dark:border-slate-700">
              <h4 className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-2">
                Import errors
              </h4>
              <ul className="space-y-0.5 text-xs text-slate-600 dark:text-slate-400">
                {errors.map((error, index) => (
                  <li key={`${error.line}-${index}`} className="font-mono">
                    line {error.line}
                    {error.source ? ` · ${error.source}` : ''} · {error.reason}
                  </li>
                ))}
                {truncatedErrors && (
                  <li className="italic">
                    … and {(description.errorCount ?? 0) - errors.length} more, not kept in the
                    journal
                  </li>
                )}
              </ul>
            </div>
          )}

          {description.raw !== undefined && (
            <div className="px-4 py-4 border-t border-slate-200 dark:border-slate-700">
              <h4 className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-2">
                Raw payload
              </h4>
              <pre className="p-2 rounded bg-slate-50 dark:bg-slate-900 text-xs text-slate-700 dark:text-slate-300 overflow-x-auto">
                {JSON.stringify(description.raw, null, 2)}
              </pre>
            </div>
          )}
        </div>

        <div className="px-6 py-4 border-t border-slate-200 dark:border-slate-700 flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
