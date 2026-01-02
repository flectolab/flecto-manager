import { ADMIN_SECTION_OPTIONS, ACTION_OPTIONS } from '../../hooks/usePermissions'

export interface AdminPermission {
  type?: 'user' | 'role'
  section: string
  action: string
}

interface AdminPermissionsEditorProps {
  permissions: AdminPermission[]
  onChange: (index: number, field: 'section' | 'action', value: string) => void
  onAdd: () => void
  onRemove: (index: number) => void
  readOnly?: boolean
  canWrite?: boolean
  showInheritedBadge?: boolean
}

export function AdminPermissionsEditor({
  permissions,
  onChange,
  onAdd,
  onRemove,
  readOnly = false,
  canWrite = true,
  showInheritedBadge = false,
}: AdminPermissionsEditorProps) {
  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <h4 className="text-sm font-medium text-slate-700 dark:text-slate-300">
          Admin Permissions
        </h4>
        {canWrite && !readOnly && (
          <button
            type="button"
            onClick={onAdd}
            className="text-sm text-brand-purple hover:text-brand-purple/80 transition-colors"
          >
            + Add
          </button>
        )}
      </div>

      {permissions.length === 0 ? (
        <p className="text-sm text-slate-500 dark:text-slate-400">No admin permissions</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-200 dark:border-slate-700">
                <th className="text-left py-2 px-2 font-medium text-slate-600 dark:text-slate-400">Section</th>
                <th className="text-left py-2 px-2 font-medium text-slate-600 dark:text-slate-400 w-24">Action</th>
                <th className="w-20"></th>
              </tr>
            </thead>
            <tbody>
              {permissions.map((perm, index) => {
                const isRolePermission = showInheritedBadge && perm.type === 'role'
                const rowClass = isRolePermission
                  ? 'opacity-50 bg-slate-50 dark:bg-slate-900/50'
                  : ''
                const isRowReadOnly = readOnly || isRolePermission

                return (
                  <tr key={index} className={`border-b border-slate-100 dark:border-slate-700/50 ${rowClass}`}>
                    <td className="py-2 px-2">
                      {isRowReadOnly ? (
                        <span className="text-slate-600 dark:text-slate-400">{perm.section}</span>
                      ) : (
                        <select
                          value={perm.section}
                          onChange={(e) => onChange(index, 'section', e.target.value)}
                          className="w-full rounded border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-1 px-2 text-slate-900 dark:text-white focus:border-brand-purple focus:outline-none"
                        >
                          {ADMIN_SECTION_OPTIONS.map(section => (
                            <option key={section.code} value={section.code}>{section.label}</option>
                          ))}
                        </select>
                      )}
                    </td>
                    <td className="py-2 px-2">
                      {isRowReadOnly ? (
                        <span className="text-slate-600 dark:text-slate-400">{perm.action}</span>
                      ) : (
                        <select
                          value={perm.action}
                          onChange={(e) => onChange(index, 'action', e.target.value)}
                          className="w-full rounded border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-1 px-2 text-slate-900 dark:text-white focus:border-brand-purple focus:outline-none"
                        >
                          {ACTION_OPTIONS.map(opt => (
                            <option key={opt.code} value={opt.code}>{opt.label}</option>
                          ))}
                        </select>
                      )}
                    </td>
                    <td className="py-2 px-2 text-right">
                      {!isRolePermission && canWrite && !readOnly && (
                        <button
                          type="button"
                          onClick={() => onRemove(index)}
                          className="text-slate-400 hover:text-red-500 transition-colors"
                        >
                          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M6 18L18 6M6 6l12 12" />
                          </svg>
                        </button>
                      )}
                      {isRolePermission && (
                        <span className="text-xs text-slate-400 italic">from role</span>
                      )}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}