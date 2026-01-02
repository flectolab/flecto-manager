import type { RedirectBase } from '../../generated/graphql'
import { RedirectStatusBadge } from './RedirectStatusBadge'
import { RedirectTypeBadge } from './RedirectTypeBadge'

interface DiffModalProps {
  oldData: RedirectBase
  newData: RedirectBase
  onClose: () => void
}

interface DiffRowProps {
  label: string
  oldValue: React.ReactNode
  newValue: React.ReactNode
  hasChanged: boolean
}

function DiffRow({ label, oldValue, newValue, hasChanged }: DiffRowProps) {
  return (
    <tr className={hasChanged ? 'bg-amber-50/50 dark:bg-amber-900/10' : ''}>
      <td className="px-4 py-3 text-sm font-medium text-slate-700 dark:text-slate-300 w-24">
        {label}
      </td>
      <td className="px-4 py-3">
        <div className={`text-sm ${hasChanged ? 'line-through text-slate-400 dark:text-slate-500' : 'text-slate-600 dark:text-slate-400'}`}>
          {oldValue}
        </div>
      </td>
      <td className="px-4 py-3">
        <div className={`text-sm ${hasChanged ? 'text-amber-700 dark:text-amber-400 font-medium' : 'text-slate-600 dark:text-slate-400'}`}>
          {newValue}
        </div>
      </td>
    </tr>
  )
}

export function DiffModal({ oldData, newData, onClose }: DiffModalProps) {
  const typeChanged = oldData.type !== newData.type
  const sourceChanged = oldData.source !== newData.source
  const targetChanged = oldData.target !== newData.target
  const statusChanged = oldData.status !== newData.status

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={onClose} />
      <div className="relative w-full max-w-2xl mx-4 rounded-xl bg-white dark:bg-slate-800 shadow-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
        <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white flex items-center gap-2">
            <svg className="w-5 h-5 text-amber-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
            Changes Preview
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

        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-slate-50 dark:bg-slate-700/50">
              <tr>
                <th className="px-4 py-2 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  Field
                </th>
                <th className="px-4 py-2 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  <span className="flex items-center gap-1">
                    <svg className="w-4 h-4 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 12H4" />
                    </svg>
                    Current
                  </span>
                </th>
                <th className="px-4 py-2 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  <span className="flex items-center gap-1">
                    <svg className="w-4 h-4 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                    </svg>
                    New
                  </span>
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
              <DiffRow
                label="Type"
                oldValue={<RedirectTypeBadge type={oldData.type} />}
                newValue={<RedirectTypeBadge type={newData.type} />}
                hasChanged={typeChanged}
              />
              <DiffRow
                label="Source"
                oldValue={<span className="font-mono">{oldData.source}</span>}
                newValue={<span className="font-mono">{newData.source}</span>}
                hasChanged={sourceChanged}
              />
              <DiffRow
                label="Target"
                oldValue={<span className="font-mono">{oldData.target}</span>}
                newValue={<span className="font-mono">{newData.target}</span>}
                hasChanged={targetChanged}
              />
              <DiffRow
                label="Status"
                oldValue={<RedirectStatusBadge status={oldData.status} />}
                newValue={<RedirectStatusBadge status={newData.status} />}
                hasChanged={statusChanged}
              />
            </tbody>
          </table>
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