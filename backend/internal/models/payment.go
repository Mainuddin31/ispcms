package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentRecord tracks each individual payment transaction against a monthly bill.
// Multiple payments can exist per bill (partial payments over time).
type PaymentRecord struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	BillID            uuid.UUID  `gorm:"type:uuid;not null;index" json:"bill_id"`
	InternetAccountID uuid.UUID  `gorm:"type:uuid;not null;index" json:"internet_account_id"`
	Amount            float64    `gorm:"type:decimal(10,2);not null" json:"amount"`
	Notes             string     `gorm:"type:text" json:"notes"`
	PaymentMethod     string     `gorm:"default:'cash'" json:"payment_method"` // cash | bkash | bank | card | other
	ReceiptNumber     string     `json:"receipt_number"`
	ReceivedByID      *uuid.UUID `gorm:"type:uuid" json:"received_by_id"`
	PaidAt            time.Time  `gorm:"not null" json:"paid_at"`
	CreatedAt         time.Time  `json:"created_at"`

	Bill       MonthlyBill `gorm:"foreignKey:BillID" json:"bill,omitempty"`
	ReceivedBy *User       `gorm:"foreignKey:ReceivedByID" json:"received_by,omitempty"`
}

func (PaymentRecord) TableName() string { return "payment_records" }

func (p *PaymentRecord) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	if p.PaidAt.IsZero() {
		p.PaidAt = time.Now()
	}
	return nil
}
