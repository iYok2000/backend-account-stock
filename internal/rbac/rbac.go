package rbac

import (
	"account-stock-be/internal/auth"
	"slices"
)

// Permission string format: resource:action (e.g. inventory:read).
// Matches RBAC_SPEC and frontend lib/rbac/constants.
const (
	PermDashboardRead = "dashboard:read"

	PermInventoryRead   = "inventory:read"
	PermInventoryCreate = "inventory:create"
	PermInventoryUpdate = "inventory:update"
	PermInventoryDelete = "inventory:delete"
	PermInventoryExport = "inventory:export"

	PermOrdersRead   = "orders:read"
	PermOrdersCreate = "orders:create"
	PermOrdersUpdate = "orders:update"
	PermOrdersExport = "orders:export"

	PermSuppliersRead   = "suppliers:read"
	PermSuppliersCreate = "suppliers:create"
	PermSuppliersUpdate = "suppliers:update"
	PermSuppliersDelete = "suppliers:delete"

	PermShopsRead   = "shops:read"
	PermShopsCreate = "shops:create"
	PermShopsUpdate = "shops:update"
	PermShopsDelete = "shops:delete"

	PermPromotionsRead   = "promotions:read"
	PermPromotionsCreate = "promotions:create"
	PermPromotionsUpdate = "promotions:update"
	PermPromotionsDelete = "promotions:delete"
	PermPromotionsExport = "promotions:export"

	PermAnalysisRead   = "analysis:read"
	PermAnalysisExport = "analysis:export"

	PermAgentsRead   = "agents:read"
	PermAgentsCreate = "agents:create"
	PermAgentsUpdate = "agents:update"
	PermAgentsDelete = "agents:delete"

	PermSettingsRead   = "settings:read"
	PermSettingsUpdate = "settings:update"

	PermUsersRead   = "users:read"
	PermUsersCreate = "users:create"
	PermUsersUpdate = "users:update"
	PermUsersDelete = "users:delete"
	PermUsersExport = "users:export"
)

// rolePermissions maps role to permissions (RBAC_SPEC §5 Role–Permission Matrix).
// Admin = all except users; SuperAdmin = all including users.
var rolePermissions = map[auth.Role][]string{
	auth.RoleViewer: {
		PermDashboardRead, PermInventoryRead, PermOrdersRead, PermSuppliersRead,
	},
	auth.RoleStaff: {
		PermDashboardRead, PermInventoryRead, PermInventoryCreate, PermInventoryUpdate, PermInventoryDelete, PermInventoryExport,
		PermOrdersRead, PermOrdersCreate, PermOrdersUpdate, PermOrdersExport,
		PermSuppliersRead,
		PermShopsRead, PermPromotionsRead, PermAnalysisRead,
	},
	auth.RoleManager: {
		PermDashboardRead, PermInventoryRead, PermInventoryCreate, PermInventoryUpdate, PermInventoryDelete, PermInventoryExport,
		PermOrdersRead, PermOrdersCreate, PermOrdersUpdate, PermOrdersExport,
		PermSuppliersRead, PermSuppliersCreate, PermSuppliersUpdate, PermSuppliersDelete,
		PermShopsRead, PermShopsCreate, PermShopsUpdate, PermShopsDelete,
		PermPromotionsRead, PermPromotionsCreate, PermPromotionsUpdate, PermPromotionsDelete, PermPromotionsExport,
		PermAnalysisRead, PermAnalysisExport,
		PermAgentsRead, PermSettingsRead,
	},
	auth.RoleAdmin: {
		PermDashboardRead,
		PermInventoryRead, PermInventoryCreate, PermInventoryUpdate, PermInventoryDelete, PermInventoryExport,
		PermOrdersRead, PermOrdersCreate, PermOrdersUpdate, PermOrdersExport,
		PermSuppliersRead, PermSuppliersCreate, PermSuppliersUpdate, PermSuppliersDelete,
		PermShopsRead, PermShopsCreate, PermShopsUpdate, PermShopsDelete,
		PermPromotionsRead, PermPromotionsCreate, PermPromotionsUpdate, PermPromotionsDelete, PermPromotionsExport,
		PermAnalysisRead, PermAnalysisExport,
		PermAgentsRead, PermAgentsCreate, PermAgentsUpdate, PermAgentsDelete,
		PermSettingsRead, PermSettingsUpdate,
		// Admin does NOT have users:* per RBAC_SPEC
	},
	auth.RoleSuperAdmin: {
		PermDashboardRead,
		PermInventoryRead, PermInventoryCreate, PermInventoryUpdate, PermInventoryDelete, PermInventoryExport,
		PermOrdersRead, PermOrdersCreate, PermOrdersUpdate, PermOrdersExport,
		PermSuppliersRead, PermSuppliersCreate, PermSuppliersUpdate, PermSuppliersDelete,
		PermShopsRead, PermShopsCreate, PermShopsUpdate, PermShopsDelete,
		PermPromotionsRead, PermPromotionsCreate, PermPromotionsUpdate, PermPromotionsDelete, PermPromotionsExport,
		PermAnalysisRead, PermAnalysisExport,
		PermAgentsRead, PermAgentsCreate, PermAgentsUpdate, PermAgentsDelete,
		PermSettingsRead, PermSettingsUpdate,
		PermUsersRead, PermUsersCreate, PermUsersUpdate, PermUsersDelete, PermUsersExport,
	},
}

// PermissionsForRole returns permissions for the given role (Deny by Default).
func PermissionsForRole(role auth.Role) []string {
	p, ok := rolePermissions[role]
	if !ok {
		return nil
	}
	return slices.Clone(p)
}

// HasPermission returns true if permissions list contains the required permission string.
func HasPermission(permissions []string, required string) bool {
	for _, p := range permissions {
		if p == required {
			return true
		}
	}
	return false
}
