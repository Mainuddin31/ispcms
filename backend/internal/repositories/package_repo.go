package repositories

import (
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
)

type PackageFilter struct {
	Status string
	Search string
}

type PackageRepository interface {
	Create(p *models.Package) error
	Update(p *models.Package) error
	Delete(id uuid.UUID) error
	FindByID(id uuid.UUID) (*models.Package, error)
	FindByName(name string) (*models.Package, error)
	List(filter PackageFilter, page, pageSize int) ([]models.Package, int64, error)
	ListActive() ([]models.Package, error)
	Count() (total, active int64, err error)
}

type packageRepository struct{ db *gorm.DB }

func NewPackageRepository(db *gorm.DB) PackageRepository {
	return &packageRepository{db: db}
}

func (r *packageRepository) Create(p *models.Package) error {
	return r.db.Create(p).Error
}

func (r *packageRepository) Update(p *models.Package) error {
	return r.db.Save(p).Error
}

func (r *packageRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Package{}, "id = ?", id).Error
}

func (r *packageRepository) FindByID(id uuid.UUID) (*models.Package, error) {
	var p models.Package
	err := r.db.First(&p, "id = ?", id).Error
	return &p, err
}

func (r *packageRepository) FindByName(name string) (*models.Package, error) {
	var p models.Package
	err := r.db.First(&p, "package_name = ?", name).Error
	return &p, err
}

func (r *packageRepository) List(filter PackageFilter, page, pageSize int) ([]models.Package, int64, error) {
	var packages []models.Package
	var total int64
	q := r.db.Model(&models.Package{})
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		q = q.Where("package_name ILIKE ? OR display_name ILIKE ?", like, like)
	}
	q.Count(&total)
	err := q.Order("package_name ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&packages).Error
	return packages, total, err
}

func (r *packageRepository) ListActive() ([]models.Package, error) {
	var packages []models.Package
	err := r.db.Where("status = 'active'").Order("package_name ASC").Find(&packages).Error
	return packages, err
}

func (r *packageRepository) Count() (total, active int64, err error) {
	r.db.Model(&models.Package{}).Count(&total)
	r.db.Model(&models.Package{}).Where("status = 'active'").Count(&active)
	return
}
