import type { ReactNode } from 'react'
import {
  usePermissions,
  type AdminSectionType,
  type ActionType,
  type ResourceTypeType,
} from '../hooks/usePermissions'

interface AdminPermissionGateProps {
  section: AdminSectionType
  action: ActionType
  children: ReactNode
  fallback?: ReactNode
}

// Gate for admin permissions
export function AdminPermissionGate({
  section,
  action,
  children,
  fallback = null,
}: AdminPermissionGateProps) {
  const { canAdmin, loading } = usePermissions()

  if (loading) return null
  if (!canAdmin(section, action)) return <>{fallback}</>

  return <>{children}</>
}

interface ResourcePermissionGateProps {
  namespace: string
  project: string
  resource: ResourceTypeType
  action: ActionType
  children: ReactNode
  fallback?: ReactNode
}

// Gate for resource permissions
export function ResourcePermissionGate({
  namespace,
  project,
  resource,
  action,
  children,
  fallback = null,
}: ResourcePermissionGateProps) {
  const { canResource, loading } = usePermissions()

  if (loading) return null
  if (!canResource(namespace, project, resource, action)) return <>{fallback}</>

  return <>{children}</>
}

interface HasAdminAccessGateProps {
  children: ReactNode
  fallback?: ReactNode
}

// Gate for any admin access
export function HasAdminAccessGate({
  children,
  fallback = null,
}: HasAdminAccessGateProps) {
  const { hasAdminAccess, loading } = usePermissions()

  if (loading) return null
  if (!hasAdminAccess()) return <>{fallback}</>

  return <>{children}</>
}