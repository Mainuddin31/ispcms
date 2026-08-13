package repositories

import (
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
)

type RouterRepository interface {
	FindAll(page, pageSize int, search string) ([]models.Router, int64, error)
	FindByID(id uuid.UUID) (*models.Router, error)
	FindActiveWithInterval() ([]models.Router, error) // for scheduler
	Create(r *models.Router) error
	Update(r *models.Router) error
	Delete(id uuid.UUID) error
	UpdateConnectionStatus(id uuid.UUID, status string) error
	UpdateSyncTime(id uuid.UUID) error
	CountByStatus() (total, online, offline int64, err error)
}

type routerRepository struct{ db *gorm.DB }

func NewRouterRepository(db *gorm.DB) RouterRepository { return &routerRepository{db: db} }

func (r *routerRepository) FindActiveWithInterval() ([]models.Router, error) {
	var routers []models.Router
	err := r.db.Where("status = 'active' AND sync_interval > 0").Find(&routers).Error
	return routers, err
}

func (r *routerRepository) FindAll(page, pageSize int, search string) ([]models.Router, int64, error) {
	var routers []models.Router
	var total int64
	q := r.db.Model(&models.Router{})
	if search != "" {
		q = q.Where("name ILIKE ? OR ip_address ILIKE ? OR pop_name ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	q.Count(&total)
	err := q.Offset((page - 1) * pageSize).Limit(pageSize).Find(&routers).Error
	return routers, total, err
}
func (r *routerRepository) FindByID(id uuid.UUID) (*models.Router, error) {
	var router models.Router
	err := r.db.First(&router, "id = ?", id).Error
	return &router, err
}
func (r *routerRepository) Create(router *models.Router) error { return r.db.Create(router).Error }
func (r *routerRepository) Update(router *models.Router) error { return r.db.Save(router).Error }
func (r *routerRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Router{}, "id = ?", id).Error
}
func (r *routerRepository) UpdateConnectionStatus(id uuid.UUID, status string) error {
	updates := map[string]interface{}{"connection_status": status}
	if status == "connected" {
		updates["last_connected"] = gorm.Expr("NOW()")
	}
	return r.db.Model(&models.Router{}).Where("id = ?", id).Updates(updates).Error
}
func (r *routerRepository) UpdateSyncTime(id uuid.UUID) error {
	return r.db.Model(&models.Router{}).Where("id = ?", id).Update("last_sync_time", gorm.Expr("NOW()")).Error
}
func (r *routerRepository) CountByStatus() (total, online, offline int64, err error) {
	r.db.Model(&models.Router{}).Count(&total)
	r.db.Model(&models.Router{}).Where("connection_status = ?", "connected").Count(&online)
	r.db.Model(&models.Router{}).Where("connection_status != ?", "connected").Count(&offline)
	return
}
