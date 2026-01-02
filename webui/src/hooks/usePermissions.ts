import { useQuery } from '@apollo/client/react'
import { GetMeDocument } from '../generated/graphql'

// Admin section constants (matching backend)
export const AdminSection = {
  Users: 'users',
  Roles: 'roles',
  Projects: 'projects',
  Namespaces: 'namespaces',
} as const

export type AdminSectionType = (typeof AdminSection)[keyof typeof AdminSection]

// Admin section options for dropdowns
export const ADMIN_SECTION_OPTIONS = [
  { code: '*', label: '* (All)' },
  { code: AdminSection.Users, label: 'Users' },
  { code: AdminSection.Roles, label: 'Roles' },
  { code: AdminSection.Projects, label: 'Projects' },
  { code: AdminSection.Namespaces, label: 'Namespaces' },
] as const

// Action constants
export const Action = {
  Read: 'read',
  Write: 'write',
  Delete: 'delete',
  All: '*',
} as const

export type ActionType = (typeof Action)[keyof typeof Action]

// Action options for dropdowns
export const ACTION_OPTIONS = [
  { code: Action.All, label: '* (All)' },
  { code: Action.Read, label: 'read' },
  { code: Action.Write, label: 'write' },
  { code: Action.Delete, label: 'delete' },
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

  // Check if user can perform an action on a resource (namespace/project)
  const canResource = (
    namespace: string,
    project: string,
    action: ActionType
  ): boolean => {
    if (!permissions?.resources) return false

    return permissions.resources.some((p) => {
      const nsMatch = p.namespace === '*' || p.namespace === namespace
      const projMatch = p.project === '*' || p.project === project
      const actionMatch = p.action === '*' || p.action === action
      return nsMatch && projMatch && actionMatch
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
    return canResource(namespace, '*', Action.Read)
  }

  // Check if user can write to a project
  const canWriteProject = (namespace: string, project: string): boolean => {
    return canResource(namespace, project, Action.Write)
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
