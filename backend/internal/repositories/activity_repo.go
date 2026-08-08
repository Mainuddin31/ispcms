package repositories

import (
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ActivityFilter struct {
	Module string
	Period string // today | 7days | 30days | all
	Limit  int
}

type ActivityLogRepository interface {
	Create(a *models.ActivityLog) error
	List(f ActivityFilter) ([]models.ActivityLog, error)
}

type activityLogRepository struct{ db *gorm.DB }

func NewActivityLogRepository(db *gorm.DB) ActivityLogRepository {
	return &activityLogRepository{db: db}
}

func (r *activityLogRepository) Create(a *models.ActivityLog) error {
	return r.db.Omit(clause.Associations).Create(a).Error
}

func (r *activityLogRepository) List(f ActivityFilter) ([]models.ActivityLog, error) {
	var logs []models.ActivityLog
	q := r.db.Model(&models.ActivityLog{}).
		Preload("User").
		Where("activity_logs.deleted_at IS NULL")

	if f.Module != "" && f.Module != "all" {
		q = q.Where("module = ?", f.Module)
	}
	switch f.Period {
	case "today":
		now := time.Now()
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		q = q.Where("activity_logs.created_at >= ?", start)
	case "7days":
		q = q.Where("activity_logs.created_at >= ?", time.Now().AddDate(0, 0, -7))
	case "30days":
		q = q.Where("activity_logs.created_at >= ?", time.Now().AddDate(0, 0, -30))
	}

	limit := 20
	if f.Limit > 0 && f.Limit <= 100 {
		limit = f.Limit
	}

	err := q.Order("activity_logs.created_at DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

// LogEntry is a convenience helper used by services to build an ActivityLog.
type LogEntry struct {
	UserID        *uuid.UUID
	Module        string
	ActivityType  string
	Title         string
	Description   string
	ReferenceType string
	ReferenceID   string
}

func (r *activityLogRepository) Log(e LogEntry) {
	a := &models.ActivityLog{
		UserID:        e.UserID,
		Module:        e.Module,
		ActivityType:  e.ActivityType,
		Title:         e.Title,
		Description:   e.Description,
		ReferenceType: e.ReferenceType,
		ReferenceID:   e.ReferenceID,
		Action:        e.ActivityType,  // legacy compat
		Details:       e.Description,  // legacy compat
	}
	_ = r.Create(a)
}
