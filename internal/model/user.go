package model

import (
	"time"

	"gorm.io/gorm"
)

// User stores user identity and role/tier for auth (SHOPS_AND_ROLES_SPEC, USER_SPEC).
// Tenant-scoped: each user belongs to one shop (shop_id); Root has shop_id null.
// Email is unique globally. 1 user : 1 shop.
type User struct {
	ID             string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	CompanyID      string         `gorm:"type:varchar(36);not null;index" json:"company_id"` // tenant scope
	ShopID         *string        `gorm:"type:varchar(36);index" json:"shop_id"`             // null for Root only
	Email          string         `gorm:"type:varchar(256);uniqueIndex" json:"email"`
	PasswordHash   string         `gorm:"type:varchar(256)" json:"-"`
	DisplayName    string         `gorm:"type:varchar(256)" json:"display_name"`
	Role           string         `gorm:"type:varchar(32);not null" json:"role"` // Root | SuperAdmin | Admin | Affiliate
	Tier           string         `gorm:"type:varchar(16);not null" json:"tier"`
	TierStartedAt  *time.Time     `gorm:"type:timestamptz" json:"tier_started_at,omitempty"`
	TierExpiresAt  *time.Time     `gorm:"type:timestamptz" json:"tier_expires_at,omitempty"`
	InviteCodeUsed string         `gorm:"type:varchar(36)" json:"invite_code_used,omitempty"`
	InviteSlots    int            `gorm:"type:int;not null;default:0" json:"invite_slots"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}
