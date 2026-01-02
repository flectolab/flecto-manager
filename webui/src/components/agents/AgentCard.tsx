import { useState } from 'react'
import type { Agent, AgentType, AgentStatus } from '../../generated/graphql'
import { RelativeTime } from '../RelativeTime'

interface AgentCardProps {
  agent: Agent
  projectVersion: number
}

function getAgentTypeIcon(type: AgentType): string {
  return `/assets/agent/types/${type}.svg`
}

function formatAgentType(type: AgentType): string {
  return type.charAt(0).toUpperCase() + type.slice(1)
}

function formatDuration(durationNs: number): string {
  if (durationNs < 1000) {
    return `${durationNs}ns`
  }
  const us = durationNs / 1000
  if (us < 1000) {
    return `${us.toFixed(1)}µs`
  }
  const ms = us / 1000
  if (ms < 1000) {
    return `${ms.toFixed(1)}ms`
  }
  const s = ms / 1000
  if (s < 60) {
    return `${s.toFixed(2)}s`
  }
  const min = Math.floor(s / 60)
  const sec = s % 60
  return `${min}m ${sec.toFixed(0)}s`
}

function AgentStatusBadge({ status }: { status: AgentStatus }) {
  const isSuccess = status === 'success'
  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-full ${
        isSuccess
          ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
          : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
      }`}
    >
      <span className={`w-1.5 h-1.5 rounded-full ${isSuccess ? 'bg-green-500' : 'bg-red-500'}`} />
      {isSuccess ? 'Success' : 'Error'}
    </span>
  )
}

function TruncatedError({ error }: { error: string }) {
  const [showTooltip, setShowTooltip] = useState(false)
  const maxLength = 50
  const isTruncated = error.length > maxLength
  const displayError = isTruncated ? `${error.slice(0, maxLength)}...` : error

  return (
    <div
      className="relative"
      onMouseEnter={() => isTruncated && setShowTooltip(true)}
      onMouseLeave={() => setShowTooltip(false)}
    >
      <p className="text-xs text-red-600 dark:text-red-400 font-mono break-all cursor-default">
        {displayError}
      </p>
      {showTooltip && (
        <div className="absolute bottom-full left-0 mb-2 p-3 max-w-xs bg-slate-900 dark:bg-slate-700 text-white text-xs font-mono rounded-lg shadow-xl z-50 whitespace-pre-wrap break-all">
          {error}
          <span className="absolute top-full left-4 border-4 border-transparent border-t-slate-900 dark:border-t-slate-700" />
        </div>
      )}
    </div>
  )
}

export function AgentCard({ agent, projectVersion }: AgentCardProps) {
  const iconSrc = getAgentTypeIcon(agent.type)
  const isError = agent.status === 'error'
  const isVersionMismatch = agent.version !== projectVersion

  return (
    <div
      className={`rounded-xl border bg-white dark:bg-slate-800 p-5 transition-all hover:shadow-lg ${
        isError
          ? 'border-red-200 dark:border-red-800/50'
          : 'border-slate-200 dark:border-slate-700'
      }`}
    >
      <div className="flex items-start gap-4">
        {/* Agent Type Icon */}
        <div className="shrink-0">
          <div className={`w-12 h-12 rounded-lg flex items-center justify-center ${
            isError
              ? 'bg-red-100 dark:bg-red-900/20'
              : 'bg-slate-100 dark:bg-slate-700'
          }`}>
            <img
              src={iconSrc}
              alt={formatAgentType(agent.type)}
              className="w-8 h-8"
            />
          </div>
        </div>

        {/* Agent Info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-2 mb-2">
            <h3 className="font-semibold text-slate-900 dark:text-white truncate" title={agent.name}>
              {agent.name}
            </h3>
            <AgentStatusBadge status={agent.status} />
          </div>

          <div className="space-y-2">
            {/* Type */}
            <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
              <span className="text-slate-400 dark:text-slate-500">Type:</span>
              <span className="font-medium">{formatAgentType(agent.type)}</span>
            </div>

            {/* Created At */}
            <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
              <svg className="w-4 h-4 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
              <span>Created: <RelativeTime date={agent.createdAt} /></span>
            </div>

            {/* Updated At */}
            <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
              <svg className="w-4 h-4 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
              <span>Updated: <RelativeTime date={agent.updatedAt} /></span>
            </div>

            {/* Version */}
            <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
              <svg className="w-4 h-4 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A2 2 0 013 12V7a4 4 0 014-4z" />
              </svg>
              <span>
                Version:{' '}
                <span className={`font-mono font-medium ${isVersionMismatch ? 'text-orange-500 dark:text-orange-400' : ''}`}>
                  {agent.version}
                  {isVersionMismatch && (
                    <span className="text-orange-500 dark:text-orange-400"> (current: {projectVersion})</span>
                  )}
                </span>
              </span>
            </div>

            {/* Load Duration */}
            <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
              <svg className="w-4 h-4 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span>Last Load: <span className="font-mono font-medium">{formatDuration(agent.load_duration)}</span></span>
            </div>

            {/* Last Hit */}
            <div className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
              <svg className="w-4 h-4 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              <span>Last hit: <RelativeTime date={agent.lastHitAt} /></span>
            </div>

            {/* Error Message (if any) */}
            {isError && agent.error && (
              <div className="mt-3 pt-3 border-t border-red-200 dark:border-red-800/50">
                <TruncatedError error={agent.error} />
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
