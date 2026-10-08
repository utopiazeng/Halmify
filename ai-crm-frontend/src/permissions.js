export function currentUser() {
  try {
    return JSON.parse(localStorage.getItem('current_user') || '{}')
  } catch {
    return {}
  }
}

export function userPermissions() {
  const permissions = currentUser().permissions
  return Array.isArray(permissions) ? permissions : []
}

export function isPlatformAdminUser() {
  const user = currentUser()
  if (typeof user.is_platform_admin === 'boolean') {
    return user.is_platform_admin
  }
  if (user.account_type === 'super_admin' || user.account_type === 'platform_super_admin') return true
  if (user.role === 'super_admin' || user.role === 'platform_super_admin') return true
  if ((user.account_type === 'admin' || user.role === 'admin') && Number(user.tenant_id || 0) === 0) return true
  return false
}

export function isTenantAdminUser() {
  const user = currentUser()
  return user.account_type === 'tenant_admin' || user.role === 'tenant_admin'
}

export function isAdminUser() {
  return isPlatformAdminUser() || isTenantAdminUser()
}

export function hasPermission(permission) {
  if (!permission) return true
  if (isPlatformAdminUser() || isTenantAdminUser()) return true
  const permissions = userPermissions()
  if (permissions.includes('*') || permissions.includes(permission)) return true
  const index = String(permission).lastIndexOf(':')
  if (index > 0) {
    const resource = permission.slice(0, index)
    const action = permission.slice(index + 1)
    if (action && action !== 'view' && action !== 'manage' && permissions.includes(`${resource}:manage`)) return true
    if (action === 'view' && permissions.includes(`${resource}:manage`)) return true
  }
  return false
}

export function canManageDomain(domain) {
  const actions = ['create', 'update', 'delete', 'export', 'assign', 'approve', 'manage']
  return actions.some((action) => hasPermission(`${domain}:${action}`))
}

export function canFollowRecord(row) {
  if (!row) return false
  if (typeof row.can_follow === 'boolean') return row.can_follow
  if (typeof row.can_edit === 'boolean') return row.can_edit
  if (isAdminUser()) return true
  const user = currentUser()
  const userID = Number(user.id || 0)
  return !!userID && (Number(row.owner_id || 0) === userID || Number(row.created_by || 0) === userID)
}

export function canEditRecord(row) {
  if (!row) return false
  if (typeof row.can_edit === 'boolean') return row.can_edit
  if (isAdminUser()) return true
  const user = currentUser()
  const userID = Number(user.id || 0)
  return !!userID && (Number(row.owner_id || 0) === userID || Number(row.created_by || 0) === userID)
}

export function canDeleteRecord(row) {
  if (!row) return false
  if (typeof row.can_delete === 'boolean') return row.can_delete
  return canEditRecord(row)
}

export function canManageRecordACL(row) {
  if (!row) return false
  if (typeof row.can_manage_acl === 'boolean') return row.can_manage_acl
  return canEditRecord(row)
}

export function hasExactPermission(permission) {
  if (!permission) return true
  if (isPlatformAdminUser() || isTenantAdminUser()) return true
  const permissions = userPermissions()
  return permissions.includes('*') || permissions.includes(permission)
}

export function canViewDomain(domain) {
  return hasPermission(`${domain}:view`) || canManageDomain(domain)
}

export function enabledModules() {
  const user = currentUser()
  const modules = user.enabled_modules
  return Array.isArray(modules) ? modules : []
}
