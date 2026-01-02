import { useCurrentProject } from '../hooks/useCurrentProject'
import { usePermissions, Action, ResourceType } from '../hooks/usePermissions'
import { useDocumentTitle } from '../hooks/useDocumentTitle'
import { RedirectTester } from '../components/redirects'

export function RedirectTesterPage() {
  const { namespaceCode, projectCode } = useCurrentProject()
  useDocumentTitle(namespaceCode && projectCode ? `Redirect Tester - ${namespaceCode}/${projectCode}` : 'Redirect Tester')
  const { canResource, loading: permissionsLoading } = usePermissions()

  const canRead = namespaceCode && projectCode ? canResource(namespaceCode, projectCode, ResourceType.Redirect, Action.Read) : false

  if (!namespaceCode || !projectCode || permissionsLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-purple border-t-transparent"></div>
      </div>
    )
  }

  if (!canRead) {
    return (
      <div className="rounded-xl bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 p-6">
        <div className="flex items-center gap-3">
          <svg className="w-6 h-6 text-amber-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div>
            <h3 className="font-semibold text-amber-800 dark:text-amber-300">Access Denied</h3>
            <p className="text-amber-700 dark:text-amber-400">You don't have permission to test redirects for this project.</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-slate-900 dark:text-white">Redirect Tester</h2>
        <p className="text-slate-600 dark:text-slate-400 mt-1">
          Test URLs against your project's redirect rules
        </p>
      </div>

      {/* Tester Component - Full width */}
      <RedirectTester
        namespaceCode={namespaceCode}
        projectCode={projectCode}
        availableScopes={['PROJECT', 'PROJECT_WITH_DRAFT']}
        defaultScope="PROJECT"
      />
    </div>
  )
}