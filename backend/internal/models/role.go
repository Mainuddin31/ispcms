package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StringSlice stores a []string as a JSON text column.
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	return string(b), err
}

func (s *StringSlice) Scan(value interface{}) error {
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("StringSlice: cannot scan type %T", value)
	}
	return json.Unmarshal(bytes, s)
}

type Role struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name            string         `gorm:"uniqueIndex;not null" json:"name"`
	DisplayName     string         `json:"display_name"`
	Description     string         `json:"description"`
	// AccountPrefixes restricts which Internet Account usernames this role can see.
	// If non-empty, only accounts whose username starts with one of these prefixes
	// are visible. Empty = no accounts visible (for non-admin roles).
	AccountPrefixes StringSlice    `gorm:"type:text;default:'[]'" json:"account_prefixes"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	Permissions     []Permission   `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

func (r *Role) BeforeCreate(tx *gorm.DB) error {
	r.ID = uuid.New()
	return nil
}

type Permission struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;not null" json:"name"`
	Module    string    `gorm:"not null" json:"module"`
	Action    string    `gorm:"not null" json:"action"`
	CreatedAt time.Time `json:"created_at"`
}

func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	p.ID = uuid.New()
	return nil
}

type RolePermission struct {
	RoleID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	PermissionID uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt    time.Time
}
