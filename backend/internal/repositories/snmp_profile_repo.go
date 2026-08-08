package repositories

import (
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
)

type SNMPProfileRepository interface {
	List() ([]models.SNMPProfile, error)
	FindByID(id uuid.UUID) (*models.SNMPProfile, error)
	Create(p *models.SNMPProfile) error
	Update(p *models.SNMPProfile) error
	Delete(id uuid.UUID) error
}

type snmpProfileRepository struct{ db *gorm.DB }

func NewSNMPProfileRepository(db *gorm.DB) SNMPProfileRepository {
	return &snmpProfileRepository{db: db}
}

func (r *snmpProfileRepository) List() ([]models.SNMPProfile, error) {
	var profiles []models.SNMPProfile
	err := r.db.Order("vendor, name").Find(&profiles).Error
	return profiles, err
}

func (r *snmpProfileRepository) FindByID(id uuid.UUID) (*models.SNMPProfile, error) {
	var p models.SNMPProfile
	err := r.db.First(&p, "id = ?", id).Error
	return &p, err
}

func (r *snmpProfileRepository) Create(p *models.SNMPProfile) error {
	return r.db.Create(p).Error
}

func (r *snmpProfileRepository) Update(p *models.SNMPProfile) error {
	return r.db.Save(p).Error
}

func (r *snmpProfileRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.SNMPProfile{}, "id = ?", id).Error
}
