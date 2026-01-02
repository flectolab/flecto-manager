import { useNavigate, useParams } from 'react-router-dom'

interface AgentStatusIndicatorProps {
  count: number
  className?: string
}

export function AgentStatusIndicator({ count, className = '' }: AgentStatusIndicatorProps) {
  const navigate = useNavigate()
  const { namespace, project } = useParams()

  if (count <= 0) return null

  const handleClick = () => {
    navigate(`/${namespace}/${project}/agents?status=error`)
  }

  return (
    <button
      onClick={handleClick}
      className={`flex items-center gap-2 px-3 py-1.5 rounded-lg bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-400 hover:bg-red-200 dark:hover:bg-red-900/50 transition-colors ${className}`}
      title={`${count} agent error${count > 1 ? 's' : ''}`}
    >
      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
        />
      </svg>
      <span className="text-sm font-medium">
        {count} agent error{count > 1 ? 's' : ''}
      </span>
    </button>
  )
}