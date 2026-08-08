package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	FullName       string         `gorm:"not null" json:"full_name"`
	Username       string         `gorm:"uniqueIndex;not null" json:"username"`
	Email          string         `gorm:"uniqueIndex;not null" json:"email"`
	Phone          string         `json:"phone"`
	Password       string         `gorm:"not null" json:"-"`
	Status         string         `gorm:"default:'active'" json:"status"`
	Avatar         string         `json:"avatar"`
	LastLogin      *time.Time     `json:"last_login"`
	FailedAttempts int            `gorm:"default:0" json:"-"`
	LockedUntil    *time.Time     `json:"-"`
	CreatedBy      *uuid.UUID     `json:"created_by"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	Roles          []Role         `gorm:"many2many:user_roles;" json:"roles,omitempty"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.ID = uuid.New()
	return nil
}

type UserRole struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	RoleID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time
}
