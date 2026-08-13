package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Router struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name             string         `gorm:"not null" json:"name"`
	IPAddress        string         `gorm:"not null" json:"ip_address"`
	APIPort          int            `gorm:"default:8728" json:"api_port"`
	Username         string         `gorm:"not null" json:"username"`
	Password         string         `gorm:"not null" json:"-"`
	Location         string         `json:"location"`
	POPName          string         `json:"pop_name"`
	Description      string         `json:"description"`
	Status           string         `gorm:"default:'active'" json:"status"`
	ConnectionStatus string         `gorm:"default:'disconnected'" json:"connection_status"`
	SyncInterval     int            `gorm:"default:60" json:"sync_interval"` // minutes; 0 = manual only
	LastConnected    *time.Time     `json:"last_connected"`
	LastSyncTime     *time.Time     `json:"last_sync_time"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (r *Router) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
