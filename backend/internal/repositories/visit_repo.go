package repositories

import (
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
)

type VisitFilter struct {
	AssignedStaffID *uuid.UUID
	Status          string
	DateFrom        string
	DateTo          string
	Search          string
	BillingStatus   string // pending | paid (filters bill.status)
}

type VisitRepository interface {
	Create(v *models.Visit) error
	Update(v *models.Visit) error
	FindByID(id uuid.UUID) (*models.Visit, error)
	List(filter VisitFilter, page, pageSize int) ([]models.Visit, int64, error)
	// HasActiveVisitForBill returns the existing active (Scheduled/Rescheduled) visit for a bill.
	HasActiveVisitForBill(billID uuid.UUID) (*models.Visit, error)
	// TodayVisits returns visits scheduled for today; if staffID is non-nil, filters to that staff.
	TodayVisits(staffID *uuid.UUID) ([]models.Visit, error)
	// TodayCount returns the count of today's visits (scoped to staff if given).
	TodayCount(staffID *uuid.UUID) (int64, error)
	// PendingCustomers returns accounts that have a pending/due bill for the current billing
	// month and whose current bill is not fully paid, along with any active visit info.
	PendingCustomers(month, year int) ([]models.PendingVisitCustomer, error)
	// VisitsByAccount returns all visits for an internet account (history).
	VisitsByAccount(internetAccountID uuid.UUID) ([]models.Visit, error)
}

type visitRepository struct{ db *gorm.DB }

func NewVisitRepository(db *gorm.DB) VisitRepository {
	return &visitRepository{db: db}
}

func (r *visitRepository) Create(v *models.Visit) error {
	return r.db.Omit("InternetAccount", "Bill", "AssignedStaff").Create(v).Error
}

func (r *visitRepository) Update(v *models.Visit) error {
	return r.db.Omit("InternetAccount", "Bill", "AssignedStaff").Save(v).Error
}

func (r *visitRepository) FindByID(id uuid.UUID) (*models.Visit, error) {
	var v models.Visit
	err := r.db.
		Preload("InternetAccount").
		Preload("Bill.Package").
		Preload("AssignedStaff").
		First(&v, "id = ?", id).Error
	return &v, err
}

func (r *visitRepository) List(filter VisitFilter, page, pageSize int) ([]models.Visit, int64, error) {
	q := r.db.Model(&models.Visit{}).
		Preload("InternetAccount").
		Preload("Bill.Package").
		Preload("AssignedStaff")

	if filter.Status != "" && filter.Status != "all" {
		q = q.Where("visits.status = ?", filter.Status)
	}
	if filter.AssignedStaffID != nil {
		q = q.Where("visits.assigned_staff_id = ?", filter.AssignedStaffID)
	}
	if filter.DateFrom != "" {
		q = q.Where("visits.scheduled_date::date >= ?::date", filter.DateFrom)
	}
	if filter.DateTo != "" {
		q = q.Where("visits.scheduled_date::date <= ?::date", filter.DateTo)
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		q = q.Joins("LEFT JOIN internet_accounts ia ON ia.id = visits.internet_account_id").
			Where("ia.username ILIKE ? OR ia.comment ILIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var visits []models.Visit
	offset := (page - 1) * pageSize
	err := q.Order("visits.scheduled_date ASC, visits.scheduled_time ASC").
		Offset(offset).Limit(pageSize).Find(&visits).Error
	return visits, total, err
}

func (r *visitRepository) HasActiveVisitForBill(billID uuid.UUID) (*models.Visit, error) {
	var v models.Visit
	err := r.db.
		Preload("AssignedStaff").
		Where("bill_id = ? AND status IN ('Scheduled','Rescheduled') AND deleted_at IS NULL", billID).
		First(&v).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &v, err
}

func (r *visitRepository) TodayVisits(staffID *uuid.UUID) ([]models.Visit, error) {
	today := time.Now().Format("2006-01-02")
	q := r.db.
		Preload("InternetAccount").
		Preload("Bill.Package").
		Preload("AssignedStaff").
		Where("visits.scheduled_date::date = ?::date AND visits.status IN ('Scheduled','Rescheduled') AND visits.deleted_at IS NULL", today)
	if staffID != nil {
		q = q.Where("visits.assigned_staff_id = ?", staffID)
	}
	var visits []models.Visit
	err := q.Order("visits.scheduled_time ASC").Find(&visits).Error
	return visits, err
}

func (r *visitRepository) TodayCount(staffID *uuid.UUID) (int64, error) {
	today := time.Now().Format("2006-01-02")
	q := r.db.Model(&models.Visit{}).
		Where("scheduled_date::date = ?::date AND status IN ('Scheduled','Rescheduled') AND deleted_at IS NULL", today)
	if staffID != nil {
		q = q.Where("assigned_staff_id = ?", staffID)
	}
	var count int64
	err := q.Count(&count).Error
	return count, err
}

func (r *visitRepository) PendingCustomers(month, year int) ([]models.PendingVisitCustomer, error) {
	rows, err := r.db.Raw(`
		SELECT
			ia.id              AS internet_account_id,
			ia.username        AS username,
			ia.comment         AS comment,
			p.display_name     AS package_name,
			mb.id              AS bill_id,
			mb.billing_month   AS billing_month,
			mb.billing_year    AS billing_year,
			mb.total_amount    AS total_amount,
			mb.paid_amount     AS paid_amount,
			mb.due_amount      AS due_amount,
			mb.status          AS bill_status,
			v.id               AS existing_visit_id,
			v.scheduled_date   AS scheduled_date,
			v.scheduled_time   AS scheduled_time,
			u.full_name        AS assigned_staff_name,
			v.status           AS visit_status
		FROM monthly_bills mb
		JOIN internet_accounts ia ON ia.id = mb.internet_account_id
		JOIN packages p           ON p.id  = mb.package_id
		LEFT JOIN visits v ON (
			v.bill_id    = mb.id
			AND v.status IN ('Scheduled', 'Rescheduled')
			AND v.deleted_at IS NULL
		)
		LEFT JOIN users u ON u.id = v.assigned_staff_id
		WHERE mb.billing_month = ?
		  AND mb.billing_year  = ?
		  AND mb.status        IN ('pending', 'due', 'partial')
		  AND ia.archived_at   IS NULL
		ORDER BY mb.due_amount DESC, ia.username ASC
	`, month, year).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.PendingVisitCustomer
	for rows.Next() {
		var pc models.PendingVisitCustomer
		if err := r.db.ScanRows(rows, &pc); err != nil {
			return nil, err
		}
		result = append(result, pc)
	}
	return result, nil
}

func (r *visitRepository) VisitsByAccount(internetAccountID uuid.UUID) ([]models.Visit, error) {
	var visits []models.Visit
	err := r.db.
		Preload("AssignedStaff").
		Preload("Bill.Package").
		Where("internet_account_id = ? AND deleted_at IS NULL", internetAccountID).
		Order("scheduled_date DESC, scheduled_time DESC").
		Find(&visits).Error
	return visits, err
}
