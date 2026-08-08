package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Package represents a billable internet service package.
type Package struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	PackageName        string    `gorm:"uniqueIndex;not null" json:"package_name"`
	DisplayName        string    `gorm:"not null" json:"display_name"`
	Speed              string    `json:"speed"`
	MonthlyPrice       float64   `gorm:"type:decimal(10,2);not null" json:"monthly_price"`
	VATPercent         float64   `gorm:"type:decimal(5,2);default:0" json:"vat_percent"`
	InstallationCharge float64   `gorm:"type:decimal(10,2);default:0" json:"installation_charge"`
	Description        string    `json:"description"`
	Status             string    `gorm:"default:'active'" json:"status"` // active | inactive
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (Package) TableName() string { return "packages" }

func (p *Package) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
