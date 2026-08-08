package repositories

import (
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PaymentRepository interface {
	Create(p *models.PaymentRecord) error
	ListByAccount(internetAccountID uuid.UUID) ([]models.PaymentRecord, error)
	ListByBill(billID uuid.UUID) ([]models.PaymentRecord, error)
}

type paymentRepository struct{ db *gorm.DB }

func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(p *models.PaymentRecord) error {
	// Omit(clause.Associations) prevents GORM from cascading into p.Bill (MonthlyBill),
	// which would cascade into InternetAccount → Router → phantom router INSERT.
	// The FK fields (BillID, InternetAccountID) are already set explicitly — no cascade needed.
	return r.db.Omit(clause.Associations).Create(p).Error
}

func (r *paymentRepository) ListByAccount(internetAccountID uuid.UUID) ([]models.PaymentRecord, error) {
	var records []models.PaymentRecord
	err := r.db.
		Preload("Bill.Package").
		Preload("ReceivedBy").
		Where("internet_account_id = ?", internetAccountID).
		Order("paid_at DESC").
		Find(&records).Error
	return records, err
}

func (r *paymentRepository) ListByBill(billID uuid.UUID) ([]models.PaymentRecord, error) {
	var records []models.PaymentRecord
	err := r.db.
		Preload("ReceivedBy").
		Where("bill_id = ?", billID).
		Order("paid_at DESC").
		Find(&records).Error
	return records, err
}
