import { Link, useLocation } from 'react-router-dom'
import { ThemeToggle } from '../ThemeToggle'
import { usePermissions, Action } from '../../hooks/usePermissions'
import flectoIcon from '../../assets/flecto.svg'

interface NavItem {
  name: string
  icon: React.ReactNode
  path: string
  section?: string // admin section required (users, roles, projects)
}

const navigation: NavItem[] = [
  {
    name: 'Dashboard',
    icon: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
      </svg>
    ),
    path: '',
  },
  {
    name: 'Users',
    icon: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
      </svg>
    ),
    path: 'users',
    section: 'users',
  },
  {
    name: 'Roles',
    icon: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
      </svg>
    ),
    path: 'roles',
    section: 'roles',
  },
  {
    name: 'Namespaces',
    icon: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
      </svg>
    ),
    path: 'namespaces',
    section: 'namespaces',
  },
  {
    name: 'Projects',
    icon: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
      </svg>
    ),
    path: 'projects',
    section: 'projects',
  },
  {
    name: 'API Tokens',
    icon: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
      </svg>
    ),
    path: 'tokens',
    section: 'tokens',
  },
]

interface AdminSidebarProps {
  collapsed: boolean
  onToggle: () => void
}

export function AdminSidebar({ collapsed, onToggle }: AdminSidebarProps) {
  const location = useLocation()
  const { canAdminResource } = usePermissions()
  const basePath = '/admin'

  const visibleNavigation = navigation.filter((item) => {
    if (!item.section) return true // Dashboard is always accessible
    return canAdminResource(item.section, Action.Read)
  })

  const isActive = (path: string) => {
    const fullPath = path ? `${basePath}/${path}` : basePath
    if (path === '') {
      return location.pathname === basePath || location.pathname === `${basePath}/`
    }
    return location.pathname.startsWith(fullPath)
  }

  return (
    <aside
      className={`fixed left-0 top-0 z-40 h-screen bg-brand-dark text-white transition-all duration-300 ${
        collapsed ? 'w-16' : 'w-80'
      }`}
    >
      <div className="flex h-16 items-center gap-2 border-b border-slate-700 px-3">
        <Link to="/" className="shrink-0">
          <img src={flectoIcon} alt="Flecto" className="h-8 w-8" />
        </Link>
        {!collapsed && (
          <div className="flex-1 min-w-0 px-2">
            <p className="text-xs text-slate-400">Flecto Manager</p>
            <p className="text-sm font-medium text-white">Administration</p>
          </div>
        )}
        <button
          onClick={onToggle}
          className="shrink-0 rounded-lg p-2 hover:bg-slate-800 transition-colors ml-auto"
        >
          <svg
            className={`w-5 h-5 transition-transform ${collapsed ? 'rotate-180' : ''}`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
          </svg>
        </button>
      </div>

      <nav className="mt-4 px-2">
        <ul className="space-y-1">
          {visibleNavigation.map((item) => (
            <li key={item.name}>
              <Link
                to={item.path ? `${basePath}/${item.path}` : basePath}
                className={`flex items-center gap-3 rounded-lg px-3 py-2 transition-colors ${
                  isActive(item.path)
                    ? 'bg-brand-indigo/20 text-brand-purple'
                    : 'text-slate-400 hover:bg-slate-800 hover:text-white'
                }`}
              >
                {item.icon}
                {!collapsed && <span>{item.name}</span>}
              </Link>
            </li>
          ))}
        </ul>
      </nav>

      <div className="absolute bottom-4 left-0 right-0 px-2">
        <ThemeToggle collapsed={collapsed} />
      </div>
    </aside>
  )
}
