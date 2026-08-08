package repositories

import (
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
)

type SubscriptionRepository interface {
	Create(s *models.CustomerSubscription) error
	Update(s *models.CustomerSubscription) error
	FindByID(id uuid.UUID) (*models.CustomerSubscription, error)
	// FindActiveByAccount returns the currently active (open-ended) subscription.
	FindActiveByAccount(internetAccountID uuid.UUID) (*models.CustomerSubscription, error)
	// FindOnDate returns the subscription that covered the given date (used by billing).
	FindOnDate(internetAccountID uuid.UUID, date time.Time) (*models.CustomerSubscription, error)
	List(page, pageSize int, accountID *uuid.UUID, packageID *uuid.UUID, activeOnly bool) ([]models.CustomerSubscription, int64, error)
	// DeactivateForAccount closes all active subscriptions for an account.
	DeactivateForAccount(internetAccountID uuid.UUID, effectiveUntil time.Time) error
	CountActive() (int64, error)
}

type subscriptionRepository struct{ db *gorm.DB }

func NewSubscriptionRepository(db *gorm.DB) SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

func (r *subscriptionRepository) Create(s *models.CustomerSubscription) error {
	return r.db.Create(s).Error
}

func (r *subscriptionRepository) Update(s *models.CustomerSubscription) error {
	return r.db.Save(s).Error
}

func (r *subscriptionRepository) FindByID(id uuid.UUID) (*models.CustomerSubscription, error) {
	var s models.CustomerSubscription
	err := r.db.Preload("Package").Preload("InternetAccount.Router").First(&s, "id = ?", id).Error
	return &s, err
}

func (r *subscriptionRepository) FindActiveByAccount(internetAccountID uuid.UUID) (*models.CustomerSubscription, error) {
	var s models.CustomerSubscription
	err := r.db.Preload("Package").
		Where("internet_account_id = ? AND is_active = true", internetAccountID).
		Order("effective_from DESC").
		First(&s).Error
	return &s, err
}

// FindOnDate finds which subscription was active on a specific date.
// This intentionally does NOT filter by is_active so historical (closed) subscriptions
// are also found — needed for billing the correct package for a past month.
func (r *subscriptionRepository) FindOnDate(internetAccountID uuid.UUID, date time.Time) (*models.CustomerSubscription, error) {
	var s models.CustomerSubscription
	err := r.db.Preload("Package").
		Where(`internet_account_id = ?
			AND effective_from <= ?
			AND (effective_until IS NULL OR effective_until >= ?)`,
			internetAccountID, date, date).
		Order("effective_from DESC").
		First(&s).Error
	return &s, err
}

func (r *subscriptionRepository) List(page, pageSize int, accountID *uuid.UUID, packageID *uuid.UUID, activeOnly bool) ([]models.CustomerSubscription, int64, error) {
	var subs []models.CustomerSubscription
	var total int64
	q := r.db.Model(&models.CustomerSubscription{}).
		Preload("Package").Preload("InternetAccount.Router")
	if activeOnly {
		q = q.Where("is_active = true")
	}
	if accountID != nil {
		q = q.Where("internet_account_id = ?", accountID)
	}
	if packageID != nil {
		q = q.Where("package_id = ?", packageID)
	}
	q.Count(&total)
	err := q.Order("created_at DESC").
		Offset((page-1)*pageSize).Limit(pageSize).Find(&subs).Error
	return subs, total, err
}

func (r *subscriptionRepository) DeactivateForAccount(internetAccountID uuid.UUID, effectiveUntil time.Time) error {
	return r.db.Model(&models.CustomerSubscription{}).
		Where("internet_account_id = ? AND is_active = true", internetAccountID).
		Updates(map[string]interface{}{
			"is_active":       false,
			"effective_until": effectiveUntil,
			"updated_at":      time.Now(),
		}).Error
}

func (r *subscriptionRepository) CountActive() (int64, error) {
	var count int64
	err := r.db.Model(&models.CustomerSubscription{}).Where("is_active = true").Count(&count).Error
	return count, err
}
