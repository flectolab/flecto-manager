import { Link, useLocation, useParams } from 'react-router-dom'
import { ProjectSwitcher } from '../ProjectSwitcher'
import { ThemeToggle } from '../ThemeToggle'
import { usePermissions, Action, ResourceType } from '../../hooks/usePermissions'
import flectoIcon from '../../assets/flecto.svg'

interface NavItem {
  name: string
  icon: React.ReactNode
  path: string
  requiredResource?: typeof ResourceType[keyof typeof ResourceType]
}

interface SidebarProps {
  collapsed: boolean
  onToggle: () => void
}

export function Sidebar({ collapsed, onToggle }: SidebarProps) {
  const location = useLocation()
  const { namespace, project } = useParams()
  const basePath = `/${namespace}/${project}`
  const { canResource, permissions } = usePermissions()

  // Check if user has any resource permission for this namespace/project
  const hasAnyProjectAccess = namespace && project && permissions?.resources?.some(p => {
    const nsMatch = p.namespace === '*' || p.namespace === namespace
    const projMatch = p.project === '*' || p.project === project
    return nsMatch && projMatch
  })

  // Permission checks for specific resources
  const canReadRedirects = namespace && project ? canResource(namespace, project, ResourceType.Redirect, Action.Read) : false
  const canReadPages = namespace && project ? canResource(namespace, project, ResourceType.Page, Action.Read) : false
  const canReadAgents = namespace && project ? canResource(namespace, project, ResourceType.Agent, Action.Read) : false

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
      name: 'Redirects',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
        </svg>
      ),
      path: 'redirects',
      requiredResource: ResourceType.Redirect,
    },
    {
      name: 'Redirect Tester',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
        </svg>
      ),
      path: 'redirect-tester',
      requiredResource: ResourceType.Redirect,
    },
    {
      name: 'Pages',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
      ),
      path: 'pages',
      requiredResource: ResourceType.Page,
    },
    {
      name: 'Agent Status',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
        </svg>
      ),
      path: 'agents',
      requiredResource: ResourceType.Agent,
    },
  ]

  // Filter navigation items based on permissions
  const filteredNavigation = navigation.filter(item => {
    // Dashboard is visible if user has any access to the project
    if (item.path === '') {
      return hasAnyProjectAccess
    }
    // Other items require specific resource permissions
    if (item.requiredResource) {
      switch (item.requiredResource) {
        case ResourceType.Redirect:
          return canReadRedirects
        case ResourceType.Page:
          return canReadPages
        case ResourceType.Agent:
          return canReadAgents
        default:
          return false
      }
    }
    return true
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
      <div className={`flex h-16 items-center border-b border-slate-700 ${collapsed ? 'justify-center px-2' : 'gap-2 px-3'}`}>
        {!collapsed && (
          <Link to="/" className="shrink-0">
            <img src={flectoIcon} alt="Flecto" className="h-8 w-8" />
          </Link>
        )}
        {!collapsed && <ProjectSwitcher collapsed={collapsed} />}
        <button
          onClick={onToggle}
          className={`shrink-0 rounded-lg p-2 hover:bg-slate-800 transition-colors ${collapsed ? '' : 'ml-auto'}`}
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
          {filteredNavigation.map((item) => (
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
