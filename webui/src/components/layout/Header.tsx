import { useLocation, useParams } from 'react-router-dom'
import { UserDropdown } from '../UserDropdown'
import { useCurrentProject } from '../../hooks/useCurrentProject'
import { DraftIndicator } from '../DraftIndicator'
import { AgentStatusIndicator } from '../AgentStatusIndicator'

interface HeaderProps {
  sidebarCollapsed: boolean
}

const pageTitles: Record<string, string> = {
  '': 'Dashboard',
  'redirects': 'Redirects',
  'pages': 'Pages',
}

export function Header({ sidebarCollapsed }: HeaderProps) {
  const location = useLocation()
  const { namespace, project } = useParams()
  const { project: projectData } = useCurrentProject()

  const getPageTitle = () => {
    const basePath = `/${namespace}/${project}`
    const currentPath = location.pathname.replace(basePath, '').replace(/^\//, '')
    return pageTitles[currentPath] || 'Dashboard'
  }

  const redirectDraftCount = projectData?.countRedirectDrafts ?? 0
  const pageDraftCount = projectData?.countPageDrafts ?? 0
  const agentErrorCount = Number(projectData?.countAgentError ?? 0)

  return (
    <header
      className={`fixed top-0 right-0 z-30 h-16 bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700 transition-all duration-300 ${
        sidebarCollapsed ? 'left-16' : 'left-80'
      }`}
    >
      <div className="flex h-full items-center justify-between px-6">
        <h1 className="text-xl font-semibold text-slate-800 dark:text-white">{getPageTitle()}</h1>

        <div className="flex items-center gap-4">
          <AgentStatusIndicator count={agentErrorCount} />
          <DraftIndicator redirectDraftCount={redirectDraftCount} pageDraftCount={pageDraftCount} />
          <UserDropdown />
        </div>
      </div>
    </header>
  )
}