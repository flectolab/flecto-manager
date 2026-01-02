import { useTheme } from '../contexts/ThemeContext'

interface ThemeToggleProps {
  collapsed?: boolean
}

export function ThemeToggle({ collapsed = false }: ThemeToggleProps) {
  const { theme, toggleTheme } = useTheme()

  return (
    <button
      onClick={toggleTheme}
      className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-slate-400 hover:bg-slate-800 hover:text-white transition-colors"
    >
      {theme === 'light' ? (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"
          />
        </svg>
      ) : (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"
          />
        </svg>
      )}
      {!collapsed && <span>{theme === 'light' ? 'Dark mode' : 'Light mode'}</span>}
    </button>
  )
}
