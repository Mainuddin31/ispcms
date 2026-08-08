package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PPPoESecret struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	RouterID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"router_id"`
	RouterOSID    string         `gorm:"column:routeros_id" json:"routeros_id"`
	Username      string         `gorm:"not null;index" json:"username"`
	Password      string         `json:"password"`
	Profile       string         `json:"profile"`
	Service       string         `json:"service"`
	LocalAddress  string         `gorm:"column:local_address" json:"local_address"`
	RemoteAddress string         `gorm:"column:remote_address" json:"remote_address"`
	CallerID      string         `gorm:"column:caller_id" json:"caller_id"`
	Disabled      bool           `json:"disabled"`
	Comment       string         `json:"comment"`
	LastSeen      *time.Time     `gorm:"column:last_seen" json:"last_seen"`
	SyncTime      time.Time      `gorm:"column:sync_time" json:"sync_time"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Router        Router         `gorm:"foreignKey:RouterID" json:"router,omitempty"`
}

func (PPPoESecret) TableName() string { return "pppoe_secrets" }

func (p *PPPoESecret) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type PPPoEActiveSession struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	RouterID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"router_id"`
	RouterOSID     string     `gorm:"column:routeros_id" json:"routeros_id"`
	Username       string     `gorm:"not null;index" json:"username"`
	CurrentIP      string     `gorm:"column:current_ip" json:"current_ip"`
	Uptime         string     `json:"uptime"`
	SessionID      string     `gorm:"column:session_id" json:"session_id"`
	ConnectedSince *time.Time `gorm:"column:connected_since" json:"connected_since"`
	SyncTime       time.Time  `gorm:"column:sync_time" json:"sync_time"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (PPPoEActiveSession) TableName() string { return "pppoe_active_sessions" }

func (p *PPPoEActiveSession) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
