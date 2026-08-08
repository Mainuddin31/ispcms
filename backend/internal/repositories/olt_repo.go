package repositories

import (
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
)

type OLTFilter struct {
	Search string
	Status string
}

type OLTRepository interface {
	List(f OLTFilter) ([]models.OLT, error)
	FindByID(id uuid.UUID) (*models.OLT, error)
	FindActiveWithInterval() ([]models.OLT, error) // for scheduler
	Create(o *models.OLT) error
	Update(o *models.OLT) error
	UpdateLastSync(id uuid.UUID, t time.Time) error
	SoftDelete(id uuid.UUID, deletedBy uuid.UUID) error
	Stats() (*models.OLTStats, error)
}

type oltRepository struct{ db *gorm.DB }

func NewOLTRepository(db *gorm.DB) OLTRepository {
	return &oltRepository{db: db}
}

func (r *oltRepository) baseQuery() *gorm.DB {
	return r.db.Where("deleted_at IS NULL")
}

func (r *oltRepository) List(f OLTFilter) ([]models.OLT, error) {
	q := r.baseQuery().Preload("SNMPProfile")
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("name ILIKE ? OR management_ip ILIKE ? OR vendor ILIKE ? OR pop ILIKE ?", like, like, like, like)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	var olts []models.OLT
	err := q.Order("name").Find(&olts).Error
	return olts, err
}

func (r *oltRepository) FindByID(id uuid.UUID) (*models.OLT, error) {
	var o models.OLT
	err := r.baseQuery().Preload("SNMPProfile").First(&o, "id = ?", id).Error
	return &o, err
}

func (r *oltRepository) FindActiveWithInterval() ([]models.OLT, error) {
	var olts []models.OLT
	err := r.baseQuery().
		Preload("SNMPProfile").
		Where("status = 'active' AND sync_interval > 0").
		Find(&olts).Error
	return olts, err
}

func (r *oltRepository) Create(o *models.OLT) error {
	return r.db.Create(o).Error
}

func (r *oltRepository) Update(o *models.OLT) error {
	return r.db.Save(o).Error
}

func (r *oltRepository) UpdateLastSync(id uuid.UUID, t time.Time) error {
	return r.db.Model(&models.OLT{}).Where("id = ?", id).Update("last_sync_at", t).Error
}

func (r *oltRepository) SoftDelete(id uuid.UUID, deletedBy uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&models.OLT{}).Where("id = ?", id).Updates(map[string]interface{}{
		"deleted_at": now,
		"deleted_by": deletedBy,
	}).Error
}

func (r *oltRepository) Stats() (*models.OLTStats, error) {
	stats := &models.OLTStats{}
	r.db.Model(&models.OLT{}).Where("deleted_at IS NULL").Count(&stats.TotalOLTs)
	r.db.Model(&models.OLT{}).Where("deleted_at IS NULL AND status = 'active'").Count(&stats.ActiveOLTs)
	r.db.Model(&models.PONPort{}).Where("archived_at IS NULL").Count(&stats.TotalPONPorts)
	r.db.Model(&models.ONU{}).Where("archived_at IS NULL").Count(&stats.TotalONUs)
	r.db.Model(&models.ONU{}).Where("archived_at IS NULL AND status = 'online'").Count(&stats.OnlineONUs)
	r.db.Model(&models.ONU{}).Where("archived_at IS NULL AND status = 'offline'").Count(&stats.OfflineONUs)
	r.db.Model(&models.ONU{}).Where("archived_at IS NULL AND internet_account_id IS NULL").Count(&stats.UnassignedONUs)
	if stats.TotalPONPorts > 0 {
		var maxCapacity int64
		r.db.Model(&models.PONPort{}).Where("archived_at IS NULL").Select("COALESCE(SUM(max_onus), 0)").Scan(&maxCapacity)
		if maxCapacity > 0 {
			stats.PortUtilization = float64(stats.TotalONUs) / float64(maxCapacity) * 100
		}
	}
	return stats, nil
}
