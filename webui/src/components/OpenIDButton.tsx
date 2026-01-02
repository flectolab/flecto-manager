import { useState, useEffect } from 'react'
import { authService } from '../services/auth'
import type { OpenIDConfig } from '../types/user'

const PROVIDER_ICONS: Record<string, React.ReactNode> = {
  google: (
    <svg className="w-5 h-5" viewBox="0 0 24 24">
      <path fill="currentColor" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
      <path fill="currentColor" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
      <path fill="currentColor" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
      <path fill="currentColor" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
    </svg>
  ),
  keycloak: (
    <svg className="w-5 h-5" viewBox="0 0 24 24">
      <path fill="currentColor" d="M12 2L2 7v10l10 5 10-5V7L12 2zm0 2.18l7.5 3.75v7.14L12 18.82l-7.5-3.75V7.93L12 4.18z"/>
    </svg>
  ),
  microsoft: (
    <svg className="w-5 h-5" viewBox="0 0 24 24">
      <path fill="currentColor" d="M11.4 24H0V12.6h11.4V24zM24 24H12.6V12.6H24V24zM11.4 11.4H0V0h11.4v11.4zm12.6 0H12.6V0H24v11.4z"/>
    </svg>
  ),
}

function DefaultIcon() {
  return (
    <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path strokeLinecap="round" strokeLinejoin="round" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
    </svg>
  )
}

function getIcon(icon?: string): React.ReactNode {
  if (!icon) return <DefaultIcon />

  // Si c'est une icône prédéfinie
  if (PROVIDER_ICONS[icon.toLowerCase()]) {
    return PROVIDER_ICONS[icon.toLowerCase()]
  }

  // Si c'est une URL
  if (icon.startsWith('http://') || icon.startsWith('https://') || icon.startsWith('/')) {
    return <img src={icon} alt="" className="w-5 h-5" />
  }

  return <DefaultIcon />
}

interface OpenIDButtonProps {
  className?: string
}

export function OpenIDButton({ className }: OpenIDButtonProps) {
  const [config, setConfig] = useState<OpenIDConfig | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchConfig = async () => {
      try {
        const openidConfig = await authService.getOpenIDConfig()
        setConfig(openidConfig)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load OpenID config')
      } finally {
        setIsLoading(false)
      }
    }

    fetchConfig()
  }, [])

  if (isLoading) {
    return null
  }

  if (error || !config?.enabled) {
    return null
  }

  const handleClick = () => {
    if (config.authUrl) {
      window.location.href = config.authUrl
    }
  }

  const buttonLabel = config.name || 'SSO'

  return (
    <button
      type="button"
      onClick={handleClick}
      className={`w-full flex items-center justify-center gap-3 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-3 px-4 text-slate-700 dark:text-slate-300 font-medium hover:bg-slate-50 dark:hover:bg-slate-800 focus:outline-none focus:ring-2 focus:ring-brand-purple focus:ring-offset-2 dark:focus:ring-offset-slate-800 transition-colors ${className || ''}`}
    >
      {getIcon(config.icon)}
      <span>Continue with {buttonLabel}</span>
    </button>
  )
}

interface OpenIDSectionProps {
  separatorText?: string
}

export function OpenIDSection({ separatorText = 'Or continue with' }: OpenIDSectionProps) {
  const [config, setConfig] = useState<OpenIDConfig | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const fetchConfig = async () => {
      try {
        const openidConfig = await authService.getOpenIDConfig()
        setConfig(openidConfig)
      } catch {
        // Silently fail - just don't show the section
      } finally {
        setIsLoading(false)
      }
    }

    fetchConfig()
  }, [])

  if (isLoading || !config?.enabled) {
    return null
  }

  const handleClick = () => {
    if (config.authUrl) {
      window.location.href = config.authUrl
    }
  }

  const buttonLabel = config.name || 'SSO'

  return (
    <div className="mt-6">
      <div className="relative">
        <div className="absolute inset-0 flex items-center">
          <div className="w-full border-t border-slate-200 dark:border-slate-700" />
        </div>
        <div className="relative flex justify-center text-sm">
          <span className="px-2 bg-white dark:bg-slate-800 text-slate-500 dark:text-slate-400">
            {separatorText}
          </span>
        </div>
      </div>

      <div className="mt-6">
        <button
          type="button"
          onClick={handleClick}
          className="w-full flex items-center justify-center gap-3 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-3 px-4 text-slate-700 dark:text-slate-300 font-medium hover:bg-slate-50 dark:hover:bg-slate-800 focus:outline-none focus:ring-2 focus:ring-brand-purple focus:ring-offset-2 dark:focus:ring-offset-slate-800 transition-colors"
        >
          {getIcon(config.icon)}
          <span>Continue with {buttonLabel}</span>
        </button>
      </div>
    </div>
  )
}