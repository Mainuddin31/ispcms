package repositories

import (
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
)

type PONPortRepository interface {
	ListByOLT(oltID uuid.UUID) ([]models.PONPort, error)
	FindByOLTAndIndex(oltID uuid.UUID, portIndex int) (*models.PONPort, error)
	Upsert(p *models.PONPort) error
	ArchiveMissing(oltID uuid.UUID, activeIndexes []int) (int64, error)
	UpdateONUCount(id uuid.UUID, count int) error
}

type ponPortRepository struct{ db *gorm.DB }

func NewPONPortRepository(db *gorm.DB) PONPortRepository {
	return &ponPortRepository{db: db}
}

func (r *ponPortRepository) ListByOLT(oltID uuid.UUID) ([]models.PONPort, error) {
	var ports []models.PONPort
	err := r.db.Where("olt_id = ? AND archived_at IS NULL", oltID).Order("port_index").Find(&ports).Error
	return ports, err
}

func (r *ponPortRepository) FindByOLTAndIndex(oltID uuid.UUID, portIndex int) (*models.PONPort, error) {
	var p models.PONPort
	err := r.db.Where("olt_id = ? AND port_index = ?", oltID, portIndex).First(&p).Error
	return &p, err
}

func (r *ponPortRepository) Upsert(p *models.PONPort) error {
	existing, err := r.FindByOLTAndIndex(p.OLTID, p.PortIndex)
	if err != nil {
		// not found — create
		if p.ID == uuid.Nil {
			p.ID = uuid.New()
		}
		return r.db.Create(p).Error
	}
	// update
	existing.PortName = p.PortName
	existing.Status = p.Status
	existing.ArchivedAt = nil
	return r.db.Save(existing).Error
}

func (r *ponPortRepository) ArchiveMissing(oltID uuid.UUID, activeIndexes []int) (int64, error) {
	q := r.db.Model(&models.PONPort{}).Where("olt_id = ? AND archived_at IS NULL", oltID)
	if len(activeIndexes) > 0 {
		q = q.Where("port_index NOT IN ?", activeIndexes)
	}
	result := q.Update("archived_at", "NOW()")
	return result.RowsAffected, result.Error
}

func (r *ponPortRepository) UpdateONUCount(id uuid.UUID, count int) error {
	return r.db.Model(&models.PONPort{}).Where("id = ?", id).Update("onu_count", count).Error
}
