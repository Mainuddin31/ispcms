package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CustomerSubscription tracks which Package an InternetAccount is subscribed to
// and the price locked in at the time of subscription.
// Only one subscription per account should have IsActive=true at any time.
// Historical subscriptions are kept for billing audit.
type CustomerSubscription struct {
	ID                uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	InternetAccountID uuid.UUID       `gorm:"type:uuid;not null;index" json:"internet_account_id"`
	PackageID         uuid.UUID       `gorm:"type:uuid;not null;index" json:"package_id"`
	MonthlyPrice      float64         `gorm:"type:decimal(10,2);not null" json:"monthly_price"` // locked at subscription time
	EffectiveFrom     time.Time       `json:"effective_from"`
	EffectiveUntil    *time.Time      `json:"effective_until"` // null = still active
	IsActive          bool            `gorm:"default:true;index" json:"is_active"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	InternetAccount   InternetAccount `gorm:"foreignKey:InternetAccountID" json:"internet_account,omitempty"`
	Package           Package         `gorm:"foreignKey:PackageID" json:"package,omitempty"`
}

func (CustomerSubscription) TableName() string { return "customer_subscriptions" }

func (c *CustomerSubscription) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
