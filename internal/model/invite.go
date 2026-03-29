package model

import (
	"time"

	"gorm.io/gorm"
)

// InviteCode for tier management (admin creates, users redeem).
type InviteCode struct {
	ID               string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	Code             string         `gorm:"type:varchar(32);uniqueIndex;not null" json:"code"`
	GrantTier        string         `gorm:"type:varchar(16);not null" json:"grant_tier"` // FREE/STARTER/PRO/ENTERPRISE
	TierDurationDays *int           `gorm:"type:int" json:"tier_duration_days"`          // NULL = unlimited
	MaxUses          int            `gorm:"type:int;not null;default:1" json:"max_uses"`
	UsedCount        int            `gorm:"type:int;not null;default:0" json:"used_count"`
	IsActive         bool           `gorm:"type:boolean;not null;default:true" json:"is_active"`
	ExpiresAt        *time.Time     `gorm:"type:timestamptz" json:"expires_at"`
	Note             string         `gorm:"type:text" json:"note"`
	CreatedBy        *string        `gorm:"type:varchar(36)" json:"created_by"` // User ID
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (InviteCode) TableName() string {
	return "invite_codes"
}

// TierHistory tracks tier changes for audit.
type TierHistory struct {
	ID           string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID       string     `gorm:"type:varchar(36);not null;index" json:"user_id"`
	OldTier      string     `gorm:"type:varchar(16)" json:"old_tier"`
	NewTier      string     `gorm:"type:varchar(16);not null" json:"new_tier"`
	Reason       string     `gorm:"type:varchar(64)" json:"reason"` // invite_code, admin_grant, expired, etc.
	ChangedBy    *string    `gorm:"type:varchar(36)" json:"changed_by"`
	InviteCodeID *string    `gorm:"type:varchar(36)" json:"invite_code_id"`
	StartedAt    time.Time  `gorm:"type:timestamptz;not null" json:"started_at"`
	ExpiresAt    *time.Time `gorm:"type:timestamptz" json:"expires_at"`
	Note         string     `gorm:"type:text" json:"note"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (TierHistory) TableName() string {
	return "tier_history"
}

// SystemConfig for global settings (e.g., require_invite_code).
type SystemConfig struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	Key         string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"key"`
	Value       string    `gorm:"type:text;not null" json:"value"`
	Description string    `gorm:"type:text" json:"description"`
	UpdatedBy   *string   `gorm:"type:varchar(36)" json:"updated_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (SystemConfig) TableName() string {
	return "system_config"
}
