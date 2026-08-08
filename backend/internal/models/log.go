package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SyncLog struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	RouterID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"router_id"`
	Status           string     `json:"status"`
	SecretsTotal     int        `json:"secrets_total"`
	SecretsCreated   int        `json:"secrets_created"`   // legacy field, kept for compat
	SecretsUpdated   int        `json:"secrets_updated"`   // legacy field, kept for compat
	SecretsDeleted   int        `json:"secrets_deleted"`   // legacy field, kept for compat
	SessionsTotal    int        `json:"sessions_total"`
	// Internet account sync counts
	NewAccounts      int        `gorm:"default:0" json:"new_accounts"`
	UpdatedAccounts  int        `gorm:"default:0" json:"updated_accounts"`
	ArchivedAccounts int        `gorm:"default:0" json:"archived_accounts"`
	OnlineCount      int        `gorm:"default:0" json:"online_count"`
	OfflineCount     int        `gorm:"default:0" json:"offline_count"`
	ErrorMessage     string     `json:"error_message"`
	Duration         int64      `json:"duration_ms"`
	StartedAt        time.Time  `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at"`
	Router           Router     `gorm:"foreignKey:RouterID" json:"router,omitempty"`
}

func (s *SyncLog) BeforeCreate(tx *gorm.DB) error {
	s.ID = uuid.New()
	return nil
}

type ActivityLog struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID        *uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	Module        string     `gorm:"type:varchar(50);index" json:"module"`           // billing | payment | expense | router | customer | user
	ActivityType  string     `gorm:"type:varchar(50)" json:"activity_type"`          // payment_received | expense_recorded | bill_generated | sync_completed | user_created | etc.
	Title         string     `gorm:"type:varchar(200)" json:"title"`
	Description   string     `gorm:"type:text" json:"description"`
	ReferenceType string     `gorm:"type:varchar(50)" json:"reference_type"`         // payment | expense | bill | sync_log | user
	ReferenceID   string     `gorm:"type:varchar(36);index" json:"reference_id"`
	// Legacy fields (kept for compat)
	Action     string         `gorm:"type:varchar(50)" json:"action"`
	ResourceID string         `gorm:"type:varchar(36)" json:"resource_id"`
	Details    string         `gorm:"type:text" json:"details"`
	IPAddress  string         `gorm:"type:varchar(45)" json:"ip_address"`
	CreatedAt  time.Time      `json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (a *ActivityLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
