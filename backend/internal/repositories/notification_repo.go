package repositories

import (
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
)

type NotificationRepository interface {
	Create(n *models.Notification) error
	FindByID(id uuid.UUID) (*models.Notification, error)
	List(unreadOnly bool, page, pageSize int) ([]models.Notification, int64, error)
	MarkRead(id uuid.UUID) error
	MarkAllRead() error
	// ExistsByType checks for an existing unread notification of a given type
	// (optionally scoped to an entity ID). Used to deduplicate warnings.
	ExistsByType(notifType string, entityID *uuid.UUID) (bool, error)
	CountUnread() (int64, error)
}

type notificationRepository struct{ db *gorm.DB }

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(n *models.Notification) error {
	return r.db.Create(n).Error
}

func (r *notificationRepository) FindByID(id uuid.UUID) (*models.Notification, error) {
	var n models.Notification
	err := r.db.First(&n, "id = ?", id).Error
	return &n, err
}

func (r *notificationRepository) List(unreadOnly bool, page, pageSize int) ([]models.Notification, int64, error) {
	var notifs []models.Notification
	var total int64
	q := r.db.Model(&models.Notification{})
	if unreadOnly {
		q = q.Where("is_read = false")
	}
	q.Count(&total)
	err := q.Order("created_at DESC").
		Offset((page-1)*pageSize).Limit(pageSize).Find(&notifs).Error
	return notifs, total, err
}

func (r *notificationRepository) MarkRead(id uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&models.Notification{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_read":    true,
			"read_at":    now,
			"updated_at": now,
		}).Error
}

func (r *notificationRepository) MarkAllRead() error {
	now := time.Now()
	return r.db.Model(&models.Notification{}).Where("is_read = false").
		Updates(map[string]interface{}{
			"is_read":    true,
			"read_at":    now,
			"updated_at": now,
		}).Error
}

func (r *notificationRepository) ExistsByType(notifType string, entityID *uuid.UUID) (bool, error) {
	var count int64
	q := r.db.Model(&models.Notification{}).Where("type = ? AND is_read = false", notifType)
	if entityID != nil {
		q = q.Where("entity_id = ?", entityID)
	}
	err := q.Count(&count).Error
	return count > 0, err
}

func (r *notificationRepository) CountUnread() (int64, error) {
	var count int64
	err := r.db.Model(&models.Notification{}).Where("is_read = false").Count(&count).Error
	return count, err
}
