package repositories

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BillFilter struct {
	InternetAccountID *uuid.UUID
	PackageID         *uuid.UUID
	Status            string
	Month             int
	Year              int
	Search            string
}

type BillRepository interface {
	Create(b *models.MonthlyBill) error
	Update(b *models.MonthlyBill) error
	FindByID(id uuid.UUID) (*models.MonthlyBill, error)
	ExistsByAccountMonth(internetAccountID uuid.UUID, month, year int) (bool, error)
	List(filter BillFilter, page, pageSize int) ([]models.MonthlyBill, int64, error)
	GenerateNextBillNumber(month, year int) (string, error)
	CreateGenerationLog(log *models.BillGenerationLog) error
	ListGenerationLogs(month, year, limit int) ([]models.BillGenerationLog, error)
	// SummarizeBilling returns how many bills have been generated and how many
	// active subscriptions still have no bill for the given month.
	SummarizeBilling(month, year int) (generated, pending int64, err error)
	CountThisMonth() (int64, error)
	// FindUnpaidByAccount returns all unpaid/partial bills for an account, oldest first.
	FindUnpaidByAccount(internetAccountID uuid.UUID) ([]models.MonthlyBill, error)
}

type billRepository struct{ db *gorm.DB }

func NewBillRepository(db *gorm.DB) BillRepository {
	return &billRepository{db: db}
}

func (r *billRepository) Create(b *models.MonthlyBill) error {
	// Omit(clause.Associations) prevents GORM from cascading into the zero-value
	// InternetAccount/Package/Subscription fields, which could cascade further into
	// Router and create phantom rows. All FK fields are already set explicitly.
	return r.db.Omit(clause.Associations).Create(b).Error
}

func (r *billRepository) Update(b *models.MonthlyBill) error {
	// Omit(clause.Associations) prevents GORM from cascading the Save into
	// InternetAccount → Router, which would trigger Router.BeforeCreate and
	// INSERT phantom router rows on every payment update.
	return r.db.Omit(clause.Associations).Save(b).Error
}

func (r *billRepository) FindByID(id uuid.UUID) (*models.MonthlyBill, error) {
	var b models.MonthlyBill
	err := r.db.
		Preload("InternetAccount.Router").
		Preload("Package").
		First(&b, "id = ?", id).Error
	return &b, err
}

func (r *billRepository) ExistsByAccountMonth(internetAccountID uuid.UUID, month, year int) (bool, error) {
	var count int64
	err := r.db.Model(&models.MonthlyBill{}).
		Where("internet_account_id = ? AND billing_month = ? AND billing_year = ? AND status != 'cancelled'",
			internetAccountID, month, year).
		Count(&count).Error
	return count > 0, err
}

func (r *billRepository) List(filter BillFilter, page, pageSize int) ([]models.MonthlyBill, int64, error) {
	var bills []models.MonthlyBill
	var total int64
	q := r.db.Model(&models.MonthlyBill{}).
		Preload("InternetAccount.Router").Preload("Package")
	if filter.InternetAccountID != nil {
		q = q.Where("internet_account_id = ?", filter.InternetAccountID)
	}
	if filter.PackageID != nil {
		q = q.Where("package_id = ?", filter.PackageID)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.Month > 0 {
		q = q.Where("billing_month = ?", filter.Month)
	}
	if filter.Year > 0 {
		q = q.Where("billing_year = ?", filter.Year)
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		q = q.Joins("JOIN internet_accounts ia ON ia.id = monthly_bills.internet_account_id").
			Where("ia.username ILIKE ? OR monthly_bills.bill_number ILIKE ?", like, like)
	}
	q.Count(&total)
	err := q.Order("generated_at DESC").
		Offset((page-1)*pageSize).Limit(pageSize).Find(&bills).Error
	return bills, total, err
}

func (r *billRepository) GenerateNextBillNumber(month, year int) (string, error) {
	prefix := fmt.Sprintf("BILL-%d%02d-", year, month)
	var count int64
	r.db.Model(&models.MonthlyBill{}).
		Where("bill_number LIKE ?", prefix+"%").
		Count(&count)
	return fmt.Sprintf("%s%04d", prefix, count+1), nil
}

func (r *billRepository) CreateGenerationLog(log *models.BillGenerationLog) error {
	return r.db.Create(log).Error
}

func (r *billRepository) ListGenerationLogs(month, year, limit int) ([]models.BillGenerationLog, error) {
	var logs []models.BillGenerationLog
	q := r.db.Model(&models.BillGenerationLog{})
	if month > 0 {
		q = q.Where("billing_month = ? AND billing_year = ?", month, year)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Order("generated_at DESC").Find(&logs).Error
	return logs, err
}

func (r *billRepository) SummarizeBilling(month, year int) (generated, pending int64, err error) {
	r.db.Model(&models.MonthlyBill{}).
		Where("billing_month = ? AND billing_year = ? AND status != 'cancelled'", month, year).
		Count(&generated)
	r.db.Raw(`
		SELECT COUNT(*) FROM customer_subscriptions cs
		WHERE cs.is_active = true
		AND NOT EXISTS (
			SELECT 1 FROM monthly_bills mb
			WHERE mb.internet_account_id = cs.internet_account_id
			  AND mb.billing_month = ?
			  AND mb.billing_year = ?
			  AND mb.status != 'cancelled'
		)
	`, month, year).Scan(&pending)
	return
}

func (r *billRepository) FindUnpaidByAccount(internetAccountID uuid.UUID) ([]models.MonthlyBill, error) {
	var bills []models.MonthlyBill
	err := r.db.Model(&models.MonthlyBill{}).
		Preload("Package").
		Where("internet_account_id = ? AND status IN ('pending','due','partial')", internetAccountID).
		Order("billing_year ASC, billing_month ASC").
		Find(&bills).Error
	return bills, err
}

func (r *billRepository) CountThisMonth() (int64, error) {
	now := time.Now()
	var count int64
	err := r.db.Model(&models.MonthlyBill{}).
		Where("billing_month = ? AND billing_year = ? AND status != 'cancelled'",
			int(now.Month()), now.Year()).
		Count(&count).Error
	return count, err
}
