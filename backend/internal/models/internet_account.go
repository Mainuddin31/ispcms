package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InternetAccount is a unified view of a PPPoE secret plus its live session info.
// One row per (router_id, mikrotik_secret_id). Sync updates both secret fields and
// session fields atomically.
type InternetAccount struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	RouterID         uuid.UUID  `gorm:"type:uuid;not null;index:idx_ia_router_username,unique" json:"router_id"`
	Username         string     `gorm:"not null;index:idx_ia_router_username,unique" json:"username"`
	MikrotikSecretID string     `gorm:"index" json:"mikrotik_secret_id"`
	Password         string     `json:"password,omitempty"`
	Service          string     `json:"service"`
	Profile          string     `json:"profile"`
	LocalAddress     string     `json:"local_address"`
	RemoteAddress    string     `json:"remote_address"`
	CallerID         string     `json:"caller_id"`
	Comment          string     `json:"comment"`
	Disabled         bool       `json:"disabled"`

	// Live session fields (updated each sync)
	IsOnline       bool       `json:"is_online"`
	CurrentIP      string     `json:"current_ip"`
	SessionID      string     `json:"session_id"`
	Uptime         string     `json:"uptime"`
	ConnectedSince *time.Time `json:"connected_since"`
	LastSeen       *time.Time `json:"last_seen"`

	// Sync metadata
	SyncStatus          string `gorm:"default:'synced'" json:"sync_status"`                    // synced | missing
	PackageMappingStatus string `gorm:"default:'ok'" json:"package_mapping_status"`            // ok | mapping_required | no_profile
	LastSyncAt          *time.Time `json:"last_sync_at"`
	ArchivedAt          *time.Time `json:"archived_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Router Router `gorm:"foreignKey:RouterID" json:"router,omitempty"`
	ONU    *ONU   `gorm:"foreignKey:InternetAccountID;references:ID" json:"onu,omitempty"`
}

func (a *InternetAccount) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// TableName sets the table name explicitly.
func (InternetAccount) TableName() string { return "internet_accounts" }
