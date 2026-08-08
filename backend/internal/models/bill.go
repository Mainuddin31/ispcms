package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MonthlyBill is one bill per customer per billing month.
// Composite unique index on (internet_account_id, billing_month, billing_year)
// enforces the no-duplicate rule at the database level.
type MonthlyBill struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	BillNumber        string    `gorm:"uniqueIndex;not null" json:"bill_number"`
	InternetAccountID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_bill_account_month,priority:1" json:"internet_account_id"`
	PackageID         uuid.UUID `gorm:"type:uuid;not null;index" json:"package_id"`
	SubscriptionID    uuid.UUID `gorm:"type:uuid;not null" json:"subscription_id"`
	BillingMonth      int       `gorm:"not null;uniqueIndex:idx_bill_account_month,priority:2" json:"billing_month"` // 1–12
	BillingYear       int       `gorm:"not null;uniqueIndex:idx_bill_account_month,priority:3" json:"billing_year"`
	MonthlyCharge     float64   `gorm:"type:decimal(10,2);not null" json:"monthly_charge"`
	Discount          float64   `gorm:"type:decimal(10,2);default:0" json:"discount"`
	Fine              float64   `gorm:"type:decimal(10,2);default:0" json:"fine"`
	VAT               float64   `gorm:"type:decimal(10,2);default:0" json:"vat"`
	TotalAmount       float64   `gorm:"type:decimal(10,2);not null" json:"total_amount"`
	PaidAmount        float64   `gorm:"type:decimal(10,2);default:0" json:"paid_amount"`
	DueAmount         float64   `gorm:"type:decimal(10,2);not null" json:"due_amount"`
	// pending | due | partial | paid | cancelled
	Status      string     `gorm:"default:'pending';index" json:"status"`
	DueDate     *time.Time `json:"due_date"`
	Notes       string     `json:"notes"`
	GeneratedAt time.Time  `json:"generated_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	InternetAccount InternetAccount      `gorm:"foreignKey:InternetAccountID" json:"internet_account,omitempty"`
	Package         Package              `gorm:"foreignKey:PackageID" json:"package,omitempty"`
	Subscription    CustomerSubscription `gorm:"foreignKey:SubscriptionID" json:"subscription,omitempty"`
}

func (MonthlyBill) TableName() string { return "monthly_bills" }

func (b *MonthlyBill) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// BillSkipDetail records why a customer was skipped during bill generation.
type BillSkipDetail struct {
	AccountID string `json:"account_id"`
	Username  string `json:"username"`
	Reason    string `json:"reason"`
}

// BillGenerationLog records each bill-generation run.
type BillGenerationLog struct {
	ID             uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	BillingMonth   int              `gorm:"not null;index" json:"billing_month"`
	BillingYear    int              `gorm:"not null;index" json:"billing_year"`
	TotalAccounts  int              `json:"total_accounts"`
	BillsGenerated int              `json:"bills_generated"`
	BillsSkipped   int              `json:"bills_skipped"`
	SkipDetails    []BillSkipDetail `gorm:"serializer:json" json:"skip_details"`
	Status         string           `json:"status"` // completed | partial | failed
	GeneratedByID  *uuid.UUID       `gorm:"type:uuid" json:"generated_by_id"`
	GeneratedAt    time.Time        `json:"generated_at"`
	CreatedAt      time.Time        `json:"created_at"`
}

func (BillGenerationLog) TableName() string { return "bill_generation_logs" }

func (b *BillGenerationLog) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
