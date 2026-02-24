package model

import (
	"time"

	"gorm.io/gorm"
)

// Company is the tenant (เจ้า) per USER_SPEC. Not tenant-scoped (no company_id on self).
type Company struct {
	ID        string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	Name      string         `gorm:"type:varchar(256);not null" json:"name"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Company) TableName() string {
	return "companies"
}
