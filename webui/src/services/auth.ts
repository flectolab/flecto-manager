import type { AuthResponse, LoginRequest, OpenIDConfig } from '../types/user'
import { config } from '../config'

class AuthService {
  private getHeaders(includeAuth = false): HeadersInit {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    }
    if (includeAuth) {
      const token = this.getAccessToken()
      if (token) {
        headers[config.authHeaderName] = `Bearer ${token}`
      }
    }
    return headers
  }

  async login(data: LoginRequest): Promise<AuthResponse> {
    const response = await fetch(`${config.apiUrl}/auth/login`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify(data),
    })

    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.message || 'Login failed')
    }

    const result = await response.json()
    this.storeTokens(result.tokens)
    return result
  }

  async refresh(): Promise<AuthResponse> {
    const refreshToken = this.getRefreshToken()
    if (!refreshToken) {
      throw new Error('No refresh token available')
    }

    const response = await fetch(`${config.apiUrl}/auth/refresh`, {
      method: 'POST',
      headers: this.getHeaders(),
      body: JSON.stringify({ refreshToken }),
    })

    if (!response.ok) {
      this.clearTokens()
      throw new Error('Token refresh failed')
    }

    const result = await response.json()
    this.storeTokens(result.tokens)
    return result
  }

  async logout(): Promise<void> {
    try {
      await fetch(`${config.apiUrl}/auth/logout`, {
        method: 'POST',
        headers: this.getHeaders(true),
      })
    } finally {
      this.clearTokens()
    }
  }

  getAccessToken(): string | null {
    return localStorage.getItem('flecto-access-token')
  }

  getRefreshToken(): string | null {
    return localStorage.getItem('flecto-refresh-token')
  }

  private storeTokens(tokens: { accessToken: string; refreshToken: string }): void {
    localStorage.setItem('flecto-access-token', tokens.accessToken)
    localStorage.setItem('flecto-refresh-token', tokens.refreshToken)
  }

  private clearTokens(): void {
    localStorage.removeItem('flecto-access-token')
    localStorage.removeItem('flecto-refresh-token')
  }

  isAuthenticated(): boolean {
    return !!this.getAccessToken()
  }

  async getOpenIDConfig(): Promise<OpenIDConfig> {
    const response = await fetch(`${config.apiUrl}/auth/openid`, {
      method: 'GET',
      credentials: 'include', // Pour envoyer/recevoir les cookies
    })

    if (!response.ok) {
      throw new Error('Failed to fetch OpenID config')
    }

    return response.json()
  }

  handleOpenIDCallback(tokens: { accessToken: string; refreshToken: string }): void {
    this.storeTokens(tokens)
  }

  getAuthType(): string | null {
    const token = this.getAccessToken()
    if (!token) {
      return null
    }

    try {
      // Decode JWT payload (middle part)
      const parts = token.split('.')
      if (parts.length !== 3) {
        return null
      }

      const payload = JSON.parse(atob(parts[1]))
      return payload.authType || null
    } catch {
      return null
    }
  }
}

export const authService = new AuthService()
