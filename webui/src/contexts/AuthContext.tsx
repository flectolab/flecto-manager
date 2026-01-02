import { createContext, useContext, useEffect, useState, useCallback, useRef } from 'react'
import { useApolloClient } from '@apollo/client/react'
import type { TypedDocumentNode } from '@graphql-typed-document-node/core'
import { authService } from '../services/auth'
import { GetMeDocument, type GetMeQuery, type GetMeQueryVariables } from '../generated/graphql'
import type { LoginRequest } from '../types/user'

type User = GetMeQuery['me']

// Refresh token 5 minutes before expiration
const REFRESH_THRESHOLD_MS = 5 * 60 * 1000

function getTokenExpiration(token: string): number | null {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return payload.exp ? payload.exp * 1000 : null
  } catch {
    return null
  }
}

interface AuthContextType {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  hasAnyAdminPermission: boolean
  login: (data: LoginRequest) => Promise<void>
  logout: () => Promise<void>
  refreshUser: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const client = useApolloClient()
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const initializedRef = useRef(false)
  const refreshTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const isRefreshingRef = useRef(false)

  const fetchMe = useCallback(async (): Promise<User | null> => {
    try {
      const result = await client.query({
        query: GetMeDocument as TypedDocumentNode<GetMeQuery, GetMeQueryVariables>,
        fetchPolicy: 'network-only',
      })
      return result.data?.me ?? null
    } catch (error) {
      console.error('Failed to fetch user:', error)
      return null
    }
  }, [client])

  // Token refresh function
  const performTokenRefresh = useCallback(async () => {
    if (isRefreshingRef.current) return
    isRefreshingRef.current = true

    try {
      await authService.refresh()
      isRefreshingRef.current = false
      // Return true to indicate success (for scheduling next refresh)
      return true
    } catch (error) {
      console.error('Token refresh failed:', error)
      isRefreshingRef.current = false
      // Token refresh failed, logout user
      setUser(null)
      await client.clearStore()
      return false
    }
  }, [client])

  // Schedule token refresh before expiration
  const scheduleTokenRefresh = useCallback(() => {
    // Clear any existing timer
    if (refreshTimerRef.current) {
      clearTimeout(refreshTimerRef.current)
      refreshTimerRef.current = null
    }

    const token = authService.getAccessToken()
    if (!token) return

    const expiration = getTokenExpiration(token)
    if (!expiration) return

    const now = Date.now()
    const timeUntilRefresh = expiration - now - REFRESH_THRESHOLD_MS

    if (timeUntilRefresh <= 0) {
      // Token already expired or about to expire, refresh immediately
      void performTokenRefresh().then(success => {
        if (success) scheduleTokenRefresh()
      })
    } else {
      // Schedule refresh
      refreshTimerRef.current = setTimeout(() => {
        void performTokenRefresh().then(success => {
          if (success) scheduleTokenRefresh()
        })
      }, timeUntilRefresh)
    }
  }, [performTokenRefresh])

  // Fetch user on mount if authenticated
  useEffect(() => {
    if (initializedRef.current) return
    initializedRef.current = true

    const initAuth = async () => {
      if (authService.isAuthenticated()) {
        const userData = await fetchMe()
        if (userData) {
          setUser(userData)
          scheduleTokenRefresh()
        } else {
          // Token invalid, clear it
          await authService.logout()
        }
      }
      setIsLoading(false)
    }

    void initAuth()
  }, [fetchMe, scheduleTokenRefresh])

  // Refresh token when page becomes visible (user returns to tab)
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible' && authService.isAuthenticated()) {
        const token = authService.getAccessToken()
        if (token) {
          const expiration = getTokenExpiration(token)
          if (expiration) {
            const timeUntilExpiration = expiration - Date.now()
            // If token expires in less than threshold, refresh immediately
            if (timeUntilExpiration < REFRESH_THRESHOLD_MS) {
              void performTokenRefresh()
            } else {
              // Reschedule refresh timer
              scheduleTokenRefresh()
            }
          }
        }
      }
    }

    document.addEventListener('visibilitychange', handleVisibilityChange)
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange)
      if (refreshTimerRef.current) {
        clearTimeout(refreshTimerRef.current)
      }
    }
  }, [performTokenRefresh, scheduleTokenRefresh])

  const refreshUser = useCallback(async () => {
    if (authService.isAuthenticated()) {
      const userData = await fetchMe()
      setUser(userData)
    }
  }, [fetchMe])

  const login = useCallback(async (loginData: LoginRequest) => {
    await authService.login(loginData)
    // After login, fetch user data via GraphQL
    const userData = await fetchMe()
    setUser(userData)
    // Schedule token refresh
    scheduleTokenRefresh()
  }, [fetchMe, scheduleTokenRefresh])

  const logout = useCallback(async () => {
    await authService.logout()
    setUser(null)
    // Clear Apollo cache on logout
    await client.clearStore()
  }, [client])

  const hasAnyAdminPermission = (user?.permissions?.admin?.length ?? 0) > 0

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: !!user,
        isLoading,
        hasAnyAdminPermission,
        login,
        logout,
        refreshUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
