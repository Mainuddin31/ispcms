package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Visit represents a scheduled field collection visit for a customer
// whose current-month bill is still pending.
type Visit struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	InternetAccountID uuid.UUID  `gorm:"type:uuid;not null;index" json:"internet_account_id"`
	BillID            uuid.UUID  `gorm:"type:uuid;not null;index" json:"bill_id"`
	BillingMonth      int        `gorm:"not null" json:"billing_month"`
	BillingYear       int        `gorm:"not null" json:"billing_year"`
	AssignedStaffID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"assigned_staff_id"`
	ScheduledDate     string     `gorm:"type:date;not null;index" json:"scheduled_date"` // "YYYY-MM-DD"
	ScheduledTime     string     `gorm:"type:varchar(8);not null" json:"scheduled_time"` // "HH:MM"
	// Scheduled | Completed | Rescheduled | Cancelled
	Status      string     `gorm:"type:varchar(20);not null;default:'Scheduled';index" json:"status"`
	Notes       string     `gorm:"type:text" json:"notes"`
	CreatedBy   uuid.UUID  `gorm:"type:uuid;not null" json:"created_by"`
	UpdatedBy   *uuid.UUID `gorm:"type:uuid" json:"updated_by"`
	CompletedBy *uuid.UUID `gorm:"type:uuid" json:"completed_by"`
	CompletedAt *time.Time `json:"completed_at"`
	RescheduledBy *uuid.UUID `gorm:"type:uuid" json:"rescheduled_by"`
	RescheduledAt *time.Time `json:"rescheduled_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Associations (preloaded).
	// NOTE: Creator and Completer *User associations are intentionally omitted.
	// User.CreatedBy == Visit.CreatedBy by name, which causes GORM AutoMigrate to
	// add a phantom fk_visits_creator FK on the users table (pointing users.created_by → visits.id).
	// The UUID fields (CreatedBy, CompletedBy) are still stored; resolve user info via separate query if needed.
	InternetAccount *InternetAccount `gorm:"foreignKey:InternetAccountID;constraint:false" json:"internet_account,omitempty"`
	Bill            *MonthlyBill     `gorm:"foreignKey:BillID;constraint:false" json:"bill,omitempty"`
	AssignedStaff   *User            `gorm:"foreignKey:AssignedStaffID;references:ID;constraint:false" json:"assigned_staff,omitempty"`
}

func (v *Visit) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}

func (Visit) TableName() string { return "visits" }

// PendingVisitCustomer is the read-model for the "Pending Customers" list —
// one row per internet account that has a pending/due bill for the current month.
type PendingVisitCustomer struct {
	InternetAccountID uuid.UUID  `json:"internet_account_id"`
	Username          string     `json:"username"`
	Comment           string     `json:"comment"`
	PackageName       string     `json:"package_name"`
	BillID            uuid.UUID  `json:"bill_id"`
	BillingMonth      int        `json:"billing_month"`
	BillingYear       int        `json:"billing_year"`
	TotalAmount       float64    `json:"total_amount"`
	PaidAmount        float64    `json:"paid_amount"`
	DueAmount         float64    `json:"due_amount"`
	BillStatus        string     `json:"bill_status"`
	// Nil when no active visit scheduled
	ExistingVisitID   *uuid.UUID `json:"existing_visit_id"`
	ScheduledDate     *string    `json:"scheduled_date"`
	ScheduledTime     *string    `json:"scheduled_time"`
	AssignedStaffName *string    `json:"assigned_staff_name"`
	VisitStatus       *string    `json:"visit_status"`
}
