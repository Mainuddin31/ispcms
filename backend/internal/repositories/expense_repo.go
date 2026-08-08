package repositories

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ── Expense Category Repository ───────────────────────────────────────────────

type ExpenseCategoryRepository interface {
	Create(c *models.ExpenseCategory) error
	Update(c *models.ExpenseCategory) error
	Delete(id uuid.UUID) error
	FindByID(id uuid.UUID) (*models.ExpenseCategory, error)
	FindByName(name string) (*models.ExpenseCategory, error)
	List(status string) ([]models.ExpenseCategory, error)
}

type expenseCategoryRepository struct{ db *gorm.DB }

func NewExpenseCategoryRepository(db *gorm.DB) ExpenseCategoryRepository {
	return &expenseCategoryRepository{db: db}
}

func (r *expenseCategoryRepository) Create(c *models.ExpenseCategory) error {
	return r.db.Omit(clause.Associations).Create(c).Error
}

func (r *expenseCategoryRepository) Update(c *models.ExpenseCategory) error {
	return r.db.Omit(clause.Associations).Save(c).Error
}

func (r *expenseCategoryRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.ExpenseCategory{}, "id = ?", id).Error
}

func (r *expenseCategoryRepository) FindByID(id uuid.UUID) (*models.ExpenseCategory, error) {
	var c models.ExpenseCategory
	err := r.db.First(&c, "id = ?", id).Error
	return &c, err
}

func (r *expenseCategoryRepository) FindByName(name string) (*models.ExpenseCategory, error) {
	var c models.ExpenseCategory
	err := r.db.First(&c, "name = ?", name).Error
	return &c, err
}

func (r *expenseCategoryRepository) List(status string) ([]models.ExpenseCategory, error) {
	var cats []models.ExpenseCategory
	q := r.db.Model(&models.ExpenseCategory{})
	if status != "" && status != "all" {
		q = q.Where("status = ?", status)
	}
	err := q.Order("name ASC").Find(&cats).Error
	return cats, err
}

// ── Expense Repository ────────────────────────────────────────────────────────

type ExpenseFilter struct {
	Search        string
	CategoryID    *uuid.UUID
	PaymentMethod string
	UserID        *uuid.UUID
	DateFrom      *time.Time
	DateTo        *time.Time
	AmountMin     *float64
	AmountMax     *float64
	SortBy        string // date | amount | category | vendor | created_at
	SortDir       string // asc | desc
}

type ExpenseRepository interface {
	Create(e *models.Expense) error
	Update(e *models.Expense) error
	SoftDelete(id uuid.UUID, deletedByID uuid.UUID) error
	FindByID(id uuid.UUID) (*models.Expense, error)
	List(f ExpenseFilter, page, pageSize int) ([]models.Expense, int64, error)
	NextSequence(year, month int) (int64, error)
	Summary(from, to *time.Time, categoryID *uuid.UUID) (float64, error)
	CategoryTotals(from, to *time.Time) ([]models.CategoryTotal, error)
}

type expenseRepository struct{ db *gorm.DB }

func NewExpenseRepository(db *gorm.DB) ExpenseRepository {
	return &expenseRepository{db: db}
}

func (r *expenseRepository) Create(e *models.Expense) error {
	return r.db.Omit(clause.Associations).Create(e).Error
}

func (r *expenseRepository) Update(e *models.Expense) error {
	return r.db.Omit(clause.Associations).Save(e).Error
}

func (r *expenseRepository) SoftDelete(id uuid.UUID, deletedByID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&models.Expense{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"deleted_at": now,
			"deleted_by": deletedByID,
		}).Error
}

func (r *expenseRepository) FindByID(id uuid.UUID) (*models.Expense, error) {
	var e models.Expense
	err := r.db.
		Preload("Category").
		Preload("CreatedBy").
		Preload("UpdatedBy").
		Preload("DeletedBy").
		First(&e, "id = ? AND deleted_at IS NULL", id).Error
	return &e, err
}

func (r *expenseRepository) List(f ExpenseFilter, page, pageSize int) ([]models.Expense, int64, error) {
	var expenses []models.Expense
	var total int64

	q := r.db.Model(&models.Expense{}).
		Preload("Category").
		Preload("CreatedBy").
		Where("expenses.deleted_at IS NULL")

	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("expense_number ILIKE ? OR vendor ILIKE ? OR description ILIKE ?", like, like, like)
	}
	if f.CategoryID != nil {
		q = q.Where("category_id = ?", f.CategoryID)
	}
	if f.PaymentMethod != "" {
		q = q.Where("payment_method = ?", f.PaymentMethod)
	}
	if f.UserID != nil {
		q = q.Where("created_by = ?", f.UserID)
	}
	if f.DateFrom != nil {
		q = q.Where("expense_date >= ?", f.DateFrom)
	}
	if f.DateTo != nil {
		q = q.Where("expense_date <= ?", f.DateTo)
	}
	if f.AmountMin != nil {
		q = q.Where("amount >= ?", f.AmountMin)
	}
	if f.AmountMax != nil {
		q = q.Where("amount <= ?", f.AmountMax)
	}

	q.Count(&total)

	// Sorting
	sortCol := "expense_date"
	switch f.SortBy {
	case "amount":
		sortCol = "amount"
	case "category":
		sortCol = "category_id"
	case "vendor":
		sortCol = "vendor"
	case "created_at":
		sortCol = "created_at"
	}
	sortDir := "DESC"
	if f.SortDir == "asc" {
		sortDir = "ASC"
	}

	err := q.Order(sortCol + " " + sortDir).
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&expenses).Error
	return expenses, total, err
}

func (r *expenseRepository) NextSequence(year, month int) (int64, error) {
	var count int64
	prefix := fmt.Sprintf("EXP-%d%02d-", year, month)
	r.db.Model(&models.Expense{}).
		Where("expense_number LIKE ?", prefix+"%").
		Count(&count)
	return count + 1, nil
}

func (r *expenseRepository) Summary(from, to *time.Time, categoryID *uuid.UUID) (float64, error) {
	var total float64
	q := r.db.Model(&models.Expense{}).Where("deleted_at IS NULL").Select("COALESCE(SUM(amount), 0)")
	if from != nil {
		q = q.Where("expense_date >= ?", from)
	}
	if to != nil {
		q = q.Where("expense_date <= ?", to)
	}
	if categoryID != nil {
		q = q.Where("category_id = ?", categoryID)
	}
	err := q.Scan(&total).Error
	return total, err
}

func (r *expenseRepository) CategoryTotals(from, to *time.Time) ([]models.CategoryTotal, error) {
	var results []models.CategoryTotal
	q := r.db.Table("expenses e").
		Select("e.category_id::text, ec.name AS category_name, COALESCE(SUM(e.amount), 0) AS total").
		Joins("JOIN expense_categories ec ON ec.id = e.category_id").
		Where("e.deleted_at IS NULL")
	if from != nil {
		q = q.Where("e.expense_date >= ?", from)
	}
	if to != nil {
		q = q.Where("e.expense_date <= ?", to)
	}
	err := q.Group("e.category_id, ec.name").Order("total DESC").Scan(&results).Error
	return results, err
}
