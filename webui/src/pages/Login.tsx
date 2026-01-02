import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import { useTheme } from '../contexts/ThemeContext'
import { useDocumentTitle } from '../hooks/useDocumentTitle'
import { OpenIDSection } from '../components/OpenIDButton'
import flectoFullLight from '../assets/flecto-full-light.svg'
import flectoFullDark from '../assets/flecto-full-dark.svg'

interface LocationState {
  from?: { pathname: string; search: string }
}

export function Login() {
  useDocumentTitle('Login')
  const navigate = useNavigate()
  const location = useLocation()
  const { login } = useAuth()
  const { theme } = useTheme()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)

    try {
      await login({ username, password })

      // Redirect to the page the user was trying to access, or home
      const state = location.state as LocationState
      const redirectTo = state?.from
        ? `${state.from.pathname}${state.from.search || ''}`
        : '/'
      navigate(redirectTo, { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-slate-100 dark:bg-slate-950 flex items-center justify-center">
      <div className="w-full max-w-md px-4">
        <div className="mb-8 text-center">
          <img
            src={theme === 'dark' ? flectoFullDark : flectoFullLight}
            alt="Flecto Labs"
            className="mx-auto h-12 mb-6"
          />
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Welcome back</h1>
          <p className="text-slate-500 dark:text-slate-400 mt-2">
            Sign in to your account to continue
          </p>
        </div>

        <div className="rounded-xl bg-white dark:bg-slate-800 shadow-sm border border-slate-200 dark:border-slate-700 p-8">
          <form onSubmit={handleSubmit} className="space-y-6">
            {error && (
              <div className="rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4">
                <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
              </div>
            )}

            <div>
              <label htmlFor="username" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Username
              </label>
              <input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                className="w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-3 px-4 text-slate-900 dark:text-white placeholder-slate-400 focus:border-brand-purple focus:outline-none focus:ring-2 focus:ring-brand-purple/20"
                placeholder="username"
              />
            </div>

            <div>
              <label htmlFor="password" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                Password
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                className="w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-3 px-4 text-slate-900 dark:text-white placeholder-slate-400 focus:border-brand-purple focus:outline-none focus:ring-2 focus:ring-brand-purple/20"
                placeholder="••••••••"
              />
            </div>

            <button
              type="submit"
              disabled={isLoading}
              className="w-full rounded-lg bg-gradient-to-r from-brand-purple to-brand-indigo py-3 px-4 text-white font-medium hover:opacity-90 focus:outline-none focus:ring-2 focus:ring-brand-purple focus:ring-offset-2 dark:focus:ring-offset-slate-800 transition-opacity disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isLoading ? (
                <span className="flex items-center justify-center gap-2">
                  <svg className="animate-spin h-5 w-5" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                  Signing in...
                </span>
              ) : (
                'Sign in'
              )}
            </button>
          </form>

          <OpenIDSection />
        </div>
      </div>
    </div>
  )
}
