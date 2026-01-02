// Re-export generated types from GraphQL schema
export type {
  User,
  SubjectPermissions,
  ResourcePermission,
  AdminPermission,
  Namespace,
  Project,
  Redirect,
  RedirectType,
  RedirectStatus,
} from '../generated/graphql'

// Auth-specific types (REST API, not in GraphQL schema)
export interface TokenPair {
  accessToken: string
  refreshToken: string
  expiresAt: number
}

// User returned by REST /auth/login endpoint
export interface AuthUser {
  id: number
  username: string
  firstname: string
  lastname: string
}

export interface AuthResponse {
  user: AuthUser
  tokens: TokenPair
}

export interface LoginRequest {
  username: string
  password: string
}

export interface OpenIDConfig {
  enabled: boolean
  name?: string
  icon?: string
  authUrl?: string
}
