package model

import (
	"time"

	"gorm.io/gorm"
)

// User stores user identity and role/tier/company for auth (USER_SPEC, RBAC_SPEC).
// Tenant-scoped: each user belongs to one company (company_id).
type User struct {
	ID          string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	CompanyID   string         `gorm:"type:varchar(36);not null;index" json:"company_id"` // tenant scope
	Email       string         `gorm:"type:varchar(256);uniqueIndex" json:"email"`
	DisplayName string         `gorm:"type:varchar(256)" json:"display_name"`
	Role        string         `gorm:"type:varchar(32);not null" json:"role"`   // RBAC role
	Tier        string         `gorm:"type:varchar(16);not null" json:"tier"`  // free / paid
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}
