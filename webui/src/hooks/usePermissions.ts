import { useQuery } from '@apollo/client/react'
import { GetMeDocument } from '../generated/graphql'

// Admin section constants (matching backend)
export const AdminSection = {
  Users: 'users',
  Roles: 'roles',
  Projects: 'projects',
  Namespaces: 'namespaces',
  Tokens: 'tokens',
} as const

export type AdminSectionType = (typeof AdminSection)[keyof typeof AdminSection]

// Admin section options for dropdowns
export const ADMIN_SECTION_OPTIONS = [
  { code: '*', label: '* (All)' },
  { code: AdminSection.Users, label: 'Users' },
  { code: AdminSection.Roles, label: 'Roles' },
  { code: AdminSection.Projects, label: 'Projects' },
  { code: AdminSection.Namespaces, label: 'Namespaces' },
  { code: AdminSection.Tokens, label: 'Tokens' },
] as const

// Action constants
export const Action = {
  Read: 'read',
  Write: 'write',
  All: '*',
} as const

export type ActionType = (typeof Action)[keyof typeof Action]

// Action options for dropdowns
export const ACTION_OPTIONS = [
  { code: Action.All, label: '* (All)' },
  { code: Action.Read, label: 'read' },
  { code: Action.Write, label: 'write' },
] as const

// Resource type constants
export const ResourceType = {
  Redirect: 'redirect',
  Page: 'page',
  Agent: 'agent',
  All: '*',
} as const

export type ResourceTypeType = (typeof ResourceType)[keyof typeof ResourceType]

// Resource type options for dropdowns
export const RESOURCE_TYPE_OPTIONS = [
  { code: ResourceType.All, label: '* (All)' },
  { code: ResourceType.Redirect, label: 'redirect' },
  { code: ResourceType.Page, label: 'page' },
  { code: ResourceType.Agent, label: 'agent' },
] as const

// Shared validation regex for codes (role names, namespace codes, project codes)
// Only allows alphanumeric characters, underscores, and hyphens
export const VALID_CODE_REGEX = /^[a-zA-Z0-9_-]+$/

export function validateCode(code: string, fieldName: string = 'Code'): string {
  if (!code.trim()) {
    return `${fieldName} is required`
  }
  if (!VALID_CODE_REGEX.test(code)) {
    return `${fieldName} can only contain letters, numbers, underscores and hyphens`
  }
  return ''
}

export function usePermissions() {
  const { data, loading, error, refetch } = useQuery(GetMeDocument)

  const permissions = data?.me?.permissions

  // Check if user can perform an action on an admin section
  const canAdmin = (section: AdminSectionType, action: ActionType): boolean => {
    if (!permissions?.admin) return false

    return permissions.admin.some((p) => {
      const sectionMatch = p.section === '*' || p.section === section
      const actionMatch = p.action === '*' || p.action === action
      return sectionMatch && actionMatch
    })
  }

  // Check if user has any admin access
  const hasAdminAccess = (): boolean => {
    return (permissions?.admin?.length ?? 0) > 0
  }

  // Check if user can perform an action on a resource (namespace/project/resource)
  const canResource = (
    namespace: string,
    project: string,
    resource: ResourceTypeType,
    action: ActionType
  ): boolean => {
    if (!permissions?.resources) return false

    return permissions.resources.some((p) => {
      const nsMatch = p.namespace === '*' || p.namespace === namespace
      const projMatch = p.project === '*' || p.project === project
      const resourceMatch = p.resource === '*' || p.resource === resource
      const actionMatch = p.action === '*' || p.action === action
      return nsMatch && projMatch && resourceMatch && actionMatch
    })
  }

  const canAdminResource = (section: string, action: ActionType): boolean => {
    if (!permissions?.admin) return false

    return permissions.admin.some((p) => {
      const sectionMatch = p.section === '*' || p.section === section
      const actionMatch = p.action === '*' || p.action === action
      return sectionMatch && actionMatch
    })
  }

  // Check if user can read a namespace
  const canReadNamespace = (namespace: string): boolean => {
    return canResource(namespace, '*', ResourceType.All, Action.Read)
  }

  // Check if user can write to a project
  const canWriteProject = (namespace: string, project: string): boolean => {
    return canResource(namespace, project, ResourceType.All, Action.Write)
  }

  return {
    user: data?.me,
    permissions,
    loading,
    error,
    refetch,
    // Admin permissions
    canAdmin,
    canAdminResource,
    hasAdminAccess,
    canManageUsers: () => canAdminResource(AdminSection.Users, Action.Read),
    canManageRoles: () => canAdminResource(AdminSection.Roles, Action.Read),
    canManageProjects: () => canAdminResource(AdminSection.Projects, Action.Read),
    // Resource permissions
    canResource,
    canReadNamespace,
    canWriteProject,
  }
}

// Utility type for permission checking in components
export type PermissionsResult = ReturnType<typeof usePermissions>
