package rbac

import (
	"account-stock-be/internal/auth"
	"slices"
)

// Permission string format: resource:action (e.g. inventory:read).
// Matches SHOPS_AND_ROLES_SPEC and frontend lib/rbac/constants.
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

	PermAnalyticsRead = "analytics:read"

	PermSettingsRead   = "settings:read"
	PermSettingsUpdate = "settings:update"

	PermUsersRead   = "users:read"
	PermUsersCreate = "users:create"
	PermUsersUpdate = "users:update"
	PermUsersDelete = "users:delete"
	PermUsersExport = "users:export"

	PermInvitesRead   = "invites:read"
	PermInvitesCreate = "invites:create"
	PermInvitesUpdate = "invites:update"
	PermInvitesDelete = "invites:delete"

	PermConfigRead   = "config:read"
	PermConfigUpdate = "config:update"
)

// rolePermissions maps role to permissions (SHOPS_AND_ROLES_SPEC, RBAC_SPEC §5).
var rolePermissions = map[auth.Role][]string{
	auth.RoleRoot: {
		// Root = superuser; has ALL permissions (platform-level access to everything)
		PermDashboardRead,
		PermInventoryRead, PermInventoryCreate, PermInventoryUpdate, PermInventoryDelete, PermInventoryExport,
		PermOrdersRead, PermOrdersCreate, PermOrdersUpdate, PermOrdersExport,
		PermSuppliersRead, PermSuppliersCreate, PermSuppliersUpdate, PermSuppliersDelete,
		PermShopsRead, PermShopsCreate, PermShopsUpdate, PermShopsDelete,
		PermPromotionsRead, PermPromotionsCreate, PermPromotionsUpdate, PermPromotionsDelete, PermPromotionsExport,
		PermAnalysisRead, PermAnalysisExport,
		PermAgentsRead, PermAgentsCreate, PermAgentsUpdate, PermAgentsDelete,
		PermAnalyticsRead,
		PermSettingsRead, PermSettingsUpdate,
		PermUsersRead, PermUsersCreate, PermUsersUpdate, PermUsersDelete, PermUsersExport,
		PermInvitesRead, PermInvitesCreate, PermInvitesUpdate, PermInvitesDelete,
		PermConfigRead, PermConfigUpdate,
	},
	auth.RoleAffiliate: {
		PermDashboardRead,
		PermInventoryCreate, // import (affiliate) only
		PermAnalysisRead,    // calculator, tax, reports
		PermAnalyticsRead,
	},
	auth.RoleAdmin: {
		// Admin = all shop features EXCEPT users:* and shops:update (SHOPS_AND_ROLES_SPEC §1, RBAC_BACKEND_SPEC §4)
		PermDashboardRead,
		PermInventoryRead, PermInventoryCreate, PermInventoryUpdate, PermInventoryDelete, PermInventoryExport,
		PermOrdersRead, PermOrdersCreate, PermOrdersUpdate, PermOrdersExport,
		PermSuppliersRead, PermSuppliersCreate, PermSuppliersUpdate, PermSuppliersDelete,
		PermShopsRead,
		PermPromotionsRead, PermPromotionsCreate, PermPromotionsUpdate, PermPromotionsDelete, PermPromotionsExport,
		PermAnalysisRead, PermAnalysisExport,
		PermAgentsRead, PermAgentsCreate, PermAgentsUpdate, PermAgentsDelete,
		PermSettingsRead, PermSettingsUpdate,
	},
	auth.RoleSuperAdmin: {
		PermDashboardRead,
		PermInventoryRead, PermInventoryCreate, PermInventoryUpdate, PermInventoryDelete, PermInventoryExport,
		PermOrdersRead, PermOrdersCreate, PermOrdersUpdate, PermOrdersExport,
		PermSuppliersRead, PermSuppliersCreate, PermSuppliersUpdate, PermSuppliersDelete,
		PermShopsRead, PermShopsUpdate,
		PermPromotionsRead, PermPromotionsCreate, PermPromotionsUpdate, PermPromotionsDelete, PermPromotionsExport,
		PermAnalysisRead, PermAnalysisExport,
		PermAgentsRead, PermAgentsCreate, PermAgentsUpdate, PermAgentsDelete,
		PermSettingsRead, PermSettingsUpdate,
		PermUsersRead, PermUsersCreate, PermUsersUpdate, PermUsersDelete, PermUsersExport,
		PermInvitesRead, PermInvitesCreate, PermInvitesUpdate, PermInvitesDelete,
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
