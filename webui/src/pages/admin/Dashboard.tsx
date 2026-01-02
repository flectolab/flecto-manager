import { useNavigate } from 'react-router-dom'
import { usePermissions, Action } from '../../hooks/usePermissions'
import { useDocumentTitle } from '../../hooks/useDocumentTitle'

export function AdminDashboard() {
  useDocumentTitle('Admin')
  const navigate = useNavigate()
  const { hasAdminAccess, canAdminResource } = usePermissions()

  const adminSections = [
    {
      id: 'users',
      title: 'Users',
      description: 'Manage user accounts and permissions',
      icon: (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
        </svg>
      ),
      href: '/admin/users',
    },
    {
      id: 'roles',
      title: 'Roles',
      description: 'Configure roles and access control',
      icon: (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
        </svg>
      ),
      href: '/admin/roles',
    },
    {
      id: 'namespaces',
      title: 'Namespaces',
      description: 'Manage namespaces',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
        </svg>
      ),
      href: '/admin/namespaces',
    },
    {
      id: 'projects',
      title: 'Projects',
      description: 'Manage projects',
      icon: (
        <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
        </svg>
      ),
      href: '/admin/projects',
    },
  ]

  const accessibleSections = adminSections.filter((section) =>
    canAdminResource(section.id, Action.Read)
  )

  if (!hasAdminAccess()) {
    return (
      <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-8 text-center">
        <svg className="mx-auto w-12 h-12 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
        </svg>
        <h3 className="mt-4 text-lg font-medium text-slate-900 dark:text-white">No Access</h3>
        <p className="mt-2 text-slate-600 dark:text-slate-400">
          You don't have permission to access the admin panel.
        </p>
      </div>
    )
  }

  return (
    <div>
      <div className="mb-8">
        <h2 className="text-2xl font-bold text-slate-900 dark:text-white">Welcome to Admin Panel</h2>
        <p className="mt-2 text-slate-600 dark:text-slate-400">
          Manage your application settings and configurations
        </p>
      </div>

      {accessibleSections.length > 0 && (
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {accessibleSections.map((section) => (
            <button
              key={section.id}
              onClick={() => navigate(section.href)}
              className="group rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6 text-left transition-all hover:border-brand-purple hover:shadow-lg hover:shadow-brand-purple/10"
            >
              <div className="flex items-start justify-between">
                <div className="rounded-lg bg-brand-purple/10 p-3 text-brand-purple">
                  {section.icon}
                </div>
                <svg
                  className="w-5 h-5 text-slate-400 transition-transform group-hover:translate-x-1 group-hover:text-brand-purple"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </div>
              <h3 className="mt-4 text-lg font-semibold text-slate-900 dark:text-white group-hover:text-brand-purple">
                {section.title}
              </h3>
              <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">{section.description}</p>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
