package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Notification is a system-level alert sent to one or more roles.
type Notification struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Type           string     `gorm:"not null;index" json:"type"` // package_mapping_missing | monthly_bills_not_generated | bill_generation_completed | bill_generation_failed
	Title          string     `gorm:"not null" json:"title"`
	Message        string     `gorm:"type:text;not null" json:"message"`
	Severity       string     `gorm:"default:'info'" json:"severity"` // info | warning | error
	EntityType     string     `json:"entity_type"`                    // package | bill | profile_mapping
	EntityID       *uuid.UUID `gorm:"type:uuid" json:"entity_id"`
	RecipientRoles []string   `gorm:"serializer:json" json:"recipient_roles"`
	IsRead         bool       `gorm:"default:false;index" json:"is_read"`
	ReadAt         *time.Time `json:"read_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (Notification) TableName() string { return "notifications" }

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}
