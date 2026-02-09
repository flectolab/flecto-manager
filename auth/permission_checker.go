package auth

import (
	"context"

	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/service"
	"gorm.io/gorm"
)

const (
	ColumnNamespaceCode = model.ColumnNamespaceCode
	ColumnProjectCode   = model.ColumnProjectCode
)

// PermissionChecker allows checking if actions are permitted
type PermissionChecker struct {
	roleService service.RoleService
}

// NewPermissionChecker creates a new PermissionChecker
func NewPermissionChecker(roleService service.RoleService) *PermissionChecker {
	return &PermissionChecker{roleService: roleService}
}

// --- Methods that fetch permissions from database ---

// CanResourceForUsername checks if a user can perform an action on a namespace/project/resource
func (c *PermissionChecker) CanResourceForUsername(ctx context.Context, username, namespace, project string, resource model.ResourceType, action model.ActionType) (bool, error) {
	permissions, err := c.roleService.GetPermissionsByUsername(ctx, username)
	if err != nil {
		return false, err
	}
	return c.CanResource(permissions, namespace, project, resource, action), nil
}

// CanAdminForUsername checks if a user can perform an action on an admin section
func (c *PermissionChecker) CanAdminForUsername(ctx context.Context, username string, section model.SectionType, action model.ActionType) (bool, error) {
	permissions, err := c.roleService.GetPermissionsByUsername(ctx, username)
	if err != nil {
		return false, err
	}
	return c.CanAdmin(permissions, section, action), nil
}

// CanResourceForRoleCode checks if a role can perform an action on a namespace/project/resource
func (c *PermissionChecker) CanResourceForRoleCode(ctx context.Context, roleCode, namespace, project string, resource model.ResourceType, action model.ActionType) (bool, error) {
	permissions, err := c.roleService.GetPermissionsByRoleCode(ctx, roleCode)
	if err != nil {
		return false, err
	}
	return c.CanResource(permissions, namespace, project, resource, action), nil
}

// CanAdminForRoleCode checks if a role can perform an action on an admin section
func (c *PermissionChecker) CanAdminForRoleCode(ctx context.Context, roleCode string, section model.SectionType, action model.ActionType) (bool, error) {
	permissions, err := c.roleService.GetPermissionsByRoleCode(ctx, roleCode)
	if err != nil {
		return false, err
	}
	return c.CanAdmin(permissions, section, action), nil
}

// --- Must methods (ignore errors, return false on error) ---

// MustCanResourceForUsername checks if a user can perform an action on a resource, returns false on error
func (c *PermissionChecker) MustCanResourceForUsername(ctx context.Context, username, namespace, project string, resource model.ResourceType, action model.ActionType) bool {
	result, _ := c.CanResourceForUsername(ctx, username, namespace, project, resource, action)
	return result
}

// MustCanAdminForUsername checks if a user can perform an action on an admin section, returns false on error
func (c *PermissionChecker) MustCanAdminForUsername(ctx context.Context, username string, section model.SectionType, action model.ActionType) bool {
	result, _ := c.CanAdminForUsername(ctx, username, section, action)
	return result
}

// MustCanResourceForRoleCode checks if a role can perform an action on a resource, returns false on error
func (c *PermissionChecker) MustCanResourceForRoleCode(ctx context.Context, roleCode, namespace, project string, resource model.ResourceType, action model.ActionType) bool {
	result, _ := c.CanResourceForRoleCode(ctx, roleCode, namespace, project, resource, action)
	return result
}

// MustCanAdminForRoleCode checks if a role can perform an action on an admin section, returns false on error
func (c *PermissionChecker) MustCanAdminForRoleCode(ctx context.Context, roleCode string, section model.SectionType, action model.ActionType) bool {
	result, _ := c.CanAdminForRoleCode(ctx, roleCode, section, action)
	return result
}

// --- Methods that work directly with SubjectPermissions (no DB call) ---

// CanResource checks if permissions allow an action on a namespace/project/resource
func (c *PermissionChecker) CanResource(permissions *model.SubjectPermissions, namespace, project string, resource model.ResourceType, action model.ActionType) bool {
	for _, p := range permissions.Resources {
		if c.matchResource(p, namespace, project, resource, action) {
			return true
		}
	}
	return false
}

// CanAdmin checks if permissions allow an action on an admin section
func (c *PermissionChecker) CanAdmin(permissions *model.SubjectPermissions, section model.SectionType, action model.ActionType) bool {
	for _, p := range permissions.Admin {
		if c.matchAdmin(p, section, action) {
			return true
		}
	}
	return false
}

// matchResource checks if a ResourcePermission matches the given criteria
func (c *PermissionChecker) matchResource(p model.ResourcePermission, namespace, project string, resource model.ResourceType, action model.ActionType) bool {
	namespaceMatch := p.Namespace == "*" || p.Namespace == namespace
	projectMatch := p.Project == "*" || p.Project == project
	resourceMatch := p.Resource == model.ResourceTypeAll || p.Resource == resource || resource == model.ResourceTypeAny
	actionMatch := p.Action == model.ActionAll || p.Action == action

	return namespaceMatch && projectMatch && resourceMatch && actionMatch
}

// matchAdmin checks if an AdminPermission matches the given criteria
func (c *PermissionChecker) matchAdmin(p model.AdminPermission, section model.SectionType, action model.ActionType) bool {
	sectionMatch := p.Section == model.AdminSectionAll || p.Section == section
	actionMatch := p.Action == model.ActionAll || p.Action == action

	return sectionMatch && actionMatch
}

// --- Query filtering methods ---

// FilterQueryByNamespace adds WHERE conditions to filter by namespace based on permissions.
// Uses ColumnNamespaceCode constant for the column name.
// If user has * namespace permission, returns query unchanged.
// If user has no matching permissions, adds WHERE 1=0 to return no results.
func (c *PermissionChecker) FilterQueryByNamespace(query *gorm.DB, permissions []model.ResourcePermission, action model.ActionType) *gorm.DB {
	allowedNamespaces := c.extractAllowedNamespaces(permissions, action)

	if allowedNamespaces == nil {
		return query.Where("1 = 0")
	}

	if len(allowedNamespaces) == 0 {
		return query
	}

	return query.Where(ColumnNamespaceCode+" IN ?", allowedNamespaces)
}

// FilterQueryByProject adds WHERE conditions to filter projects within a specific namespace.
// Uses ColumnProjectCode constant for the column name.
// Handles wildcards: * namespace or * project = full access within namespace.
// If user has no matching permissions for this namespace, adds WHERE 1=0 to return no results.
func (c *PermissionChecker) FilterQueryByProject(query *gorm.DB, permissions []model.ResourcePermission, namespace string, action model.ActionType) *gorm.DB {
	filtered := c.filterPermissionsByAction(permissions, action)

	if len(filtered) == 0 {
		return query.Where("1 = 0")
	}

	// Collect allowed projects for this namespace
	var allowedProjects []string
	hasFullAccess := false

	for _, p := range filtered {
		// Check if permission applies to this namespace
		if p.Namespace != "*" && p.Namespace != namespace {
			continue
		}

		// Full access: * namespace or * project for this namespace
		if p.Project == "*" {
			hasFullAccess = true
			break
		}

		// Specific project access
		allowedProjects = append(allowedProjects, p.Project)
	}

	if hasFullAccess {
		return query.Where(ColumnNamespaceCode+" = ?", namespace)
	}

	if len(allowedProjects) == 0 {
		return query.Where("1 = 0")
	}

	return query.Where(ColumnNamespaceCode+" = ? AND "+ColumnProjectCode+" IN ?", namespace, allowedProjects)
}

// FilterQueryByNamespaceProject adds WHERE conditions to filter by namespace and project based on permissions.
// Uses ColumnNamespaceCode and ColumnProjectCode constants for column names.
// Handles wildcards: * namespace = full access, * project = full namespace access.
// If user has no matching permissions, adds WHERE 1=0 to return no results.
func (c *PermissionChecker) FilterQueryByNamespaceProject(query *gorm.DB, permissions []model.ResourcePermission, action model.ActionType) *gorm.DB {
	filtered := c.filterPermissionsByAction(permissions, action)

	if len(filtered) == 0 {
		return query.Where("1 = 0")
	}

	// Check for full access (namespace = *)
	for _, p := range filtered {
		if p.Namespace == "*" {
			return query
		}
	}

	// Track namespaces with full project access (project = *)
	fullNamespaceAccess := make(map[string]bool)
	// Track specific namespace+project pairs
	specificAccess := make(map[string]map[string]bool)

	for _, p := range filtered {
		if p.Project == "*" {
			fullNamespaceAccess[p.Namespace] = true
		} else {
			// Only add specific project if namespace doesn't already have full access
			if !fullNamespaceAccess[p.Namespace] {
				if specificAccess[p.Namespace] == nil {
					specificAccess[p.Namespace] = make(map[string]bool)
				}
				specificAccess[p.Namespace][p.Project] = true
			}
		}
	}

	// Remove specific accesses for namespaces that got full access
	for ns := range fullNamespaceAccess {
		delete(specificAccess, ns)
	}

	// Build OR conditions
	var conditions []string
	var args []interface{}

	// Add conditions for namespaces with full project access
	if len(fullNamespaceAccess) > 0 {
		namespaces := make([]string, 0, len(fullNamespaceAccess))
		for ns := range fullNamespaceAccess {
			namespaces = append(namespaces, ns)
		}
		conditions = append(conditions, ColumnNamespaceCode+" IN ?")
		args = append(args, namespaces)
	}

	// Add conditions for specific namespace+project pairs
	for ns, projects := range specificAccess {
		projectList := make([]string, 0, len(projects))
		for proj := range projects {
			projectList = append(projectList, proj)
		}
		conditions = append(conditions, "("+ColumnNamespaceCode+" = ? AND "+ColumnProjectCode+" IN ?)")
		args = append(args, ns, projectList)
	}

	// Join conditions with OR
	if len(conditions) == 1 {
		return query.Where(conditions[0], args...)
	}

	// Multiple conditions - use raw SQL with OR
	combined := "(" + conditions[0]
	for i := 1; i < len(conditions); i++ {
		combined += " OR " + conditions[i]
	}
	combined += ")"

	return query.Where(combined, args...)
}

// extractAllowedNamespaces returns the list of allowed namespaces for the given action.
// Returns nil if no permissions match (should filter to nothing).
// Returns empty slice if user has * namespace access (full access).
func (c *PermissionChecker) extractAllowedNamespaces(permissions []model.ResourcePermission, action model.ActionType) []string {
	filtered := c.filterPermissionsByAction(permissions, action)

	if len(filtered) == 0 {
		return nil
	}

	namespaces := make(map[string]bool)
	for _, p := range filtered {
		if p.Namespace == "*" {
			return []string{}
		}
		namespaces[p.Namespace] = true
	}

	result := make([]string, 0, len(namespaces))
	for ns := range namespaces {
		result = append(result, ns)
	}
	return result
}

// filterPermissionsByAction returns only permissions that match the given action
func (c *PermissionChecker) filterPermissionsByAction(permissions []model.ResourcePermission, action model.ActionType) []model.ResourcePermission {
	result := make([]model.ResourcePermission, 0, len(permissions))
	for _, p := range permissions {
		if p.Action == model.ActionAll || p.Action == action {
			result = append(result, p)
		}
	}
	return result
}
