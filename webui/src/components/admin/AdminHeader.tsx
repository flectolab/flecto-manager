import { useLocation } from 'react-router-dom'
import { UserDropdown } from '../UserDropdown'

interface AdminHeaderProps {
  sidebarCollapsed: boolean
}

const pageTitles: Record<string, string> = {
  '': 'Dashboard',
  'users': 'Users',
  'roles': 'Roles',
  'projects': 'Projects',
  'namespaces': 'Namespaces',
}

export function AdminHeader({ sidebarCollapsed }: AdminHeaderProps) {
  const location = useLocation()

  const getPageTitle = () => {
    const currentPath = location.pathname.replace('/admin', '').replace(/^\//, '')
    return pageTitles[currentPath] || 'Dashboard'
  }

  return (
    <header
      className={`fixed top-0 right-0 z-30 h-16 bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700 transition-all duration-300 ${
        sidebarCollapsed ? 'left-16' : 'left-80'
      }`}
    >
      <div className="flex h-full items-center justify-between px-6">
        <h1 className="text-xl font-semibold text-slate-800 dark:text-white">{getPageTitle()}</h1>

        <div className="flex items-center gap-4">
          <UserDropdown />
        </div>
      </div>
    </header>
  )
}
