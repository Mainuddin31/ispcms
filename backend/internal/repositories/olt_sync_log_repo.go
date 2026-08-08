package repositories

import (
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
)

type OLTSyncLogRepository interface {
	Create(l *models.OLTSyncLog) error
	Update(l *models.OLTSyncLog) error
	ListByOLT(oltID uuid.UUID, limit int) ([]models.OLTSyncLog, error)
	ListRecent(limit int) ([]models.OLTSyncLog, error)
}

type oltSyncLogRepository struct{ db *gorm.DB }

func NewOLTSyncLogRepository(db *gorm.DB) OLTSyncLogRepository {
	return &oltSyncLogRepository{db: db}
}

func (r *oltSyncLogRepository) Create(l *models.OLTSyncLog) error {
	return r.db.Create(l).Error
}

func (r *oltSyncLogRepository) Update(l *models.OLTSyncLog) error {
	return r.db.Save(l).Error
}

func (r *oltSyncLogRepository) ListByOLT(oltID uuid.UUID, limit int) ([]models.OLTSyncLog, error) {
	var logs []models.OLTSyncLog
	q := r.db.Where("olt_id = ?", oltID).Order("started_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&logs).Error
	return logs, err
}

func (r *oltSyncLogRepository) ListRecent(limit int) ([]models.OLTSyncLog, error) {
	var logs []models.OLTSyncLog
	q := r.db.Preload("OLT").Order("started_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&logs).Error
	return logs, err
}
