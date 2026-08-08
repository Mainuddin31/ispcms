package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Expense Category ──────────────────────────────────────────────────────────

type ExpenseCategory struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string    `gorm:"type:varchar(100);not null;uniqueIndex" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Status      string    `gorm:"type:varchar(20);default:'active'" json:"status"` // active | inactive
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (ExpenseCategory) TableName() string { return "expense_categories" }

func (c *ExpenseCategory) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// ── Expense ───────────────────────────────────────────────────────────────────

type Expense struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	ExpenseNumber   string     `gorm:"type:varchar(30);not null;uniqueIndex" json:"expense_number"`
	ExpenseDate     time.Time  `gorm:"type:date;not null" json:"expense_date"`
	CategoryID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"category_id"`
	Amount          float64    `gorm:"type:decimal(12,2);not null" json:"amount"`
	PaymentMethod   string     `gorm:"type:varchar(30);not null;default:'cash'" json:"payment_method"` // cash | bank | mobile | cheque | card | other
	Vendor          string     `gorm:"type:varchar(200)" json:"vendor"`
	ReferenceNumber string     `gorm:"type:varchar(100)" json:"reference_number"`
	Description     string     `gorm:"type:text" json:"description"`
	AttachmentPath  string     `gorm:"type:varchar(500)" json:"attachment_path"`
	CreatedByID     *uuid.UUID `gorm:"type:uuid;column:created_by" json:"created_by_id"`
	UpdatedByID     *uuid.UUID `gorm:"type:uuid;column:updated_by" json:"updated_by_id"`
	DeletedByID     *uuid.UUID `gorm:"type:uuid;column:deleted_by" json:"deleted_by_id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `gorm:"index" json:"deleted_at,omitempty"`

	// Associations (preload only — never cascade-create)
	Category  *ExpenseCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	CreatedBy *User            `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`
	UpdatedBy *User            `gorm:"foreignKey:UpdatedByID" json:"updated_by,omitempty"`
	DeletedBy *User            `gorm:"foreignKey:DeletedByID" json:"deleted_by,omitempty"`
}

func (Expense) TableName() string { return "expenses" }

func (e *Expense) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

// ExpenseSummary is returned by the stats/dashboard endpoint.
type ExpenseSummary struct {
	TodayTotal     float64          `json:"today_total"`
	WeekTotal      float64          `json:"week_total"`
	MonthTotal     float64          `json:"month_total"`
	YearTotal      float64          `json:"year_total"`
	AllTimeTotal   float64          `json:"all_time_total"`
	CategoryTotals []CategoryTotal  `json:"category_totals"`
}

type CategoryTotal struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Total        float64 `json:"total"`
}

// GenerateExpenseNumber builds EXP-YYYYMM-NNNNNN
func GenerateExpenseNumber(t time.Time, seq int64) string {
	return fmt.Sprintf("EXP-%d%02d-%06d", t.Year(), int(t.Month()), seq)
}
