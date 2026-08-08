package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProfileMapping links a MikroTik PPPoE profile name to a billing Package.
// Multiple MikroTik profiles can map to the same Package.
type ProfileMapping struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	MikrotikProfile string    `gorm:"uniqueIndex;not null" json:"mikrotik_profile"`
	PackageID       uuid.UUID `gorm:"type:uuid;not null;index" json:"package_id"`
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Package         Package   `gorm:"foreignKey:PackageID" json:"package,omitempty"`
}

func (ProfileMapping) TableName() string { return "profile_mappings" }

func (p *ProfileMapping) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
