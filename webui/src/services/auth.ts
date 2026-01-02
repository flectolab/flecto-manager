import type { AuthResponse, LoginRequest, ChangePasswordRequest } from '../types/user'
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

  async changePassword(data: ChangePasswordRequest): Promise<void> {
    const response = await fetch(`${config.apiUrl}/auth/change-password`, {
      method: 'POST',
      headers: this.getHeaders(true),
      body: JSON.stringify(data),
    })

    if (!response.ok) {
      const error = await response.json()
      throw new Error(error.message || 'Password change failed')
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
}

export const authService = new AuthService()
