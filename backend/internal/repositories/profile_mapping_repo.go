package repositories

import (
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
)

type ProfileMappingFilter struct {
	PackageID string
	Search    string
}

type ProfileMappingRepository interface {
	Create(m *models.ProfileMapping) error
	Update(m *models.ProfileMapping) error
	Delete(id uuid.UUID) error
	FindByID(id uuid.UUID) (*models.ProfileMapping, error)
	FindByProfile(mikrotikProfile string) (*models.ProfileMapping, error)
	List(filter ProfileMappingFilter, page, pageSize int) ([]models.ProfileMapping, int64, error)
	ListAll() ([]models.ProfileMapping, error)
	// UnmappedProfiles returns distinct profiles in internet_accounts not in profile_mappings.
	UnmappedProfiles() ([]string, error)
	Count() (int64, error)
}

type profileMappingRepository struct{ db *gorm.DB }

func NewProfileMappingRepository(db *gorm.DB) ProfileMappingRepository {
	return &profileMappingRepository{db: db}
}

func (r *profileMappingRepository) Create(m *models.ProfileMapping) error {
	return r.db.Create(m).Error
}

func (r *profileMappingRepository) Update(m *models.ProfileMapping) error {
	return r.db.Save(m).Error
}

func (r *profileMappingRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.ProfileMapping{}, "id = ?", id).Error
}

func (r *profileMappingRepository) FindByID(id uuid.UUID) (*models.ProfileMapping, error) {
	var m models.ProfileMapping
	err := r.db.Preload("Package").First(&m, "id = ?", id).Error
	return &m, err
}

func (r *profileMappingRepository) FindByProfile(mikrotikProfile string) (*models.ProfileMapping, error) {
	var m models.ProfileMapping
	err := r.db.Preload("Package").First(&m, "mikrotik_profile = ?", mikrotikProfile).Error
	return &m, err
}

func (r *profileMappingRepository) List(filter ProfileMappingFilter, page, pageSize int) ([]models.ProfileMapping, int64, error) {
	var mappings []models.ProfileMapping
	var total int64
	q := r.db.Model(&models.ProfileMapping{})
	if filter.PackageID != "" {
		q = q.Where("package_id = ?", filter.PackageID)
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		q = q.Where("mikrotik_profile ILIKE ? OR notes ILIKE ?", like, like)
	}
	q.Count(&total)
	err := q.Preload("Package").Order("mikrotik_profile ASC").
		Offset((page-1)*pageSize).Limit(pageSize).Find(&mappings).Error
	return mappings, total, err
}

func (r *profileMappingRepository) ListAll() ([]models.ProfileMapping, error) {
	var mappings []models.ProfileMapping
	err := r.db.Preload("Package").Order("mikrotik_profile ASC").Find(&mappings).Error
	return mappings, err
}

func (r *profileMappingRepository) UnmappedProfiles() ([]string, error) {
	var profiles []string
	err := r.db.Raw(`
		SELECT DISTINCT profile FROM internet_accounts
		WHERE profile != '' AND profile IS NOT NULL AND archived_at IS NULL
		  AND profile NOT IN (SELECT mikrotik_profile FROM profile_mappings)
		ORDER BY profile
	`).Scan(&profiles).Error
	return profiles, err
}

func (r *profileMappingRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.ProfileMapping{}).Count(&count).Error
	return count, err
}
