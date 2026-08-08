package services

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"github.com/ispcms/backend/internal/repositories"
)

// ── Expense Category Service ──────────────────────────────────────────────────

type ExpenseCategoryService interface {
	Create(name, description string) (*models.ExpenseCategory, error)
	Update(id uuid.UUID, name, description, status string) (*models.ExpenseCategory, error)
	Delete(id uuid.UUID) error
	Get(id uuid.UUID) (*models.ExpenseCategory, error)
	List(status string) ([]models.ExpenseCategory, error)
}

type expenseCategoryService struct {
	repo repositories.ExpenseCategoryRepository
}

func NewExpenseCategoryService(repo repositories.ExpenseCategoryRepository) ExpenseCategoryService {
	return &expenseCategoryService{repo: repo}
}

func (s *expenseCategoryService) Create(name, description string) (*models.ExpenseCategory, error) {
	if name == "" {
		return nil, errors.New("category name is required")
	}
	if _, err := s.repo.FindByName(name); err == nil {
		return nil, errors.New("category name already exists")
	}
	c := &models.ExpenseCategory{
		Name:        name,
		Description: description,
		Status:      "active",
	}
	if err := s.repo.Create(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *expenseCategoryService) Update(id uuid.UUID, name, description, status string) (*models.ExpenseCategory, error) {
	c, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("category not found")
	}
	if name != "" && name != c.Name {
		if _, err := s.repo.FindByName(name); err == nil {
			return nil, errors.New("category name already exists")
		}
		c.Name = name
	}
	if description != "" {
		c.Description = description
	}
	if status == "active" || status == "inactive" {
		c.Status = status
	}
	if err := s.repo.Update(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *expenseCategoryService) Delete(id uuid.UUID) error {
	if _, err := s.repo.FindByID(id); err != nil {
		return errors.New("category not found")
	}
	return s.repo.Delete(id)
}

func (s *expenseCategoryService) Get(id uuid.UUID) (*models.ExpenseCategory, error) {
	return s.repo.FindByID(id)
}

func (s *expenseCategoryService) List(status string) ([]models.ExpenseCategory, error) {
	return s.repo.List(status)
}

// ── Expense Service ───────────────────────────────────────────────────────────

type CreateExpenseInput struct {
	ExpenseDate     time.Time
	CategoryID      uuid.UUID
	Amount          float64
	PaymentMethod   string
	Vendor          string
	ReferenceNumber string
	Description     string
	AttachmentPath  string
	CreatedByID     uuid.UUID
}

type UpdateExpenseInput struct {
	ExpenseDate     *time.Time
	CategoryID      *uuid.UUID
	Amount          *float64
	PaymentMethod   string
	Vendor          string
	ReferenceNumber string
	Description     string
	AttachmentPath  string
	UpdatedByID     uuid.UUID
}

type ExpenseService interface {
	Create(input CreateExpenseInput) (*models.Expense, error)
	Update(id uuid.UUID, input UpdateExpenseInput) (*models.Expense, error)
	Delete(id uuid.UUID, deletedByID uuid.UUID) error
	Get(id uuid.UUID) (*models.Expense, error)
	List(f repositories.ExpenseFilter, page, pageSize int) ([]models.Expense, int64, error)
	GetSummary() (*models.ExpenseSummary, error)
}

type expenseService struct {
	repo     repositories.ExpenseRepository
	catRepo  repositories.ExpenseCategoryRepository
}

func NewExpenseService(repo repositories.ExpenseRepository, catRepo repositories.ExpenseCategoryRepository) ExpenseService {
	return &expenseService{repo: repo, catRepo: catRepo}
}

func (s *expenseService) Create(input CreateExpenseInput) (*models.Expense, error) {
	if input.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	if input.CategoryID == uuid.Nil {
		return nil, errors.New("category is required")
	}
	if input.ExpenseDate.IsZero() {
		return nil, errors.New("expense date is required")
	}
	if _, err := s.catRepo.FindByID(input.CategoryID); err != nil {
		return nil, errors.New("category not found")
	}

	now := time.Now()
	if input.ExpenseDate.IsZero() {
		input.ExpenseDate = now
	}

	seq, err := s.repo.NextSequence(input.ExpenseDate.Year(), int(input.ExpenseDate.Month()))
	if err != nil {
		return nil, err
	}
	expNumber := models.GenerateExpenseNumber(input.ExpenseDate, seq)

	pm := input.PaymentMethod
	if pm == "" {
		pm = "cash"
	}

	createdBy := &input.CreatedByID
	if input.CreatedByID == uuid.Nil {
		createdBy = nil
	}

	e := &models.Expense{
		ExpenseNumber:   expNumber,
		ExpenseDate:     input.ExpenseDate,
		CategoryID:      input.CategoryID,
		Amount:          input.Amount,
		PaymentMethod:   pm,
		Vendor:          input.Vendor,
		ReferenceNumber: input.ReferenceNumber,
		Description:     input.Description,
		AttachmentPath:  input.AttachmentPath,
		CreatedByID:     createdBy,
		UpdatedByID:     createdBy,
	}
	if err := s.repo.Create(e); err != nil {
		return nil, err
	}
	return s.repo.FindByID(e.ID)
}

func (s *expenseService) Update(id uuid.UUID, input UpdateExpenseInput) (*models.Expense, error) {
	e, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("expense not found")
	}
	if input.ExpenseDate != nil {
		e.ExpenseDate = *input.ExpenseDate
	}
	if input.CategoryID != nil {
		if _, err := s.catRepo.FindByID(*input.CategoryID); err != nil {
			return nil, errors.New("category not found")
		}
		e.CategoryID = *input.CategoryID
	}
	if input.Amount != nil {
		if *input.Amount <= 0 {
			return nil, errors.New("amount must be greater than zero")
		}
		e.Amount = *input.Amount
	}
	if input.PaymentMethod != "" {
		e.PaymentMethod = input.PaymentMethod
	}
	if input.Vendor != "" {
		e.Vendor = input.Vendor
	}
	if input.ReferenceNumber != "" {
		e.ReferenceNumber = input.ReferenceNumber
	}
	if input.Description != "" {
		e.Description = input.Description
	}
	if input.AttachmentPath != "" {
		e.AttachmentPath = input.AttachmentPath
	}
	updatedBy := input.UpdatedByID
	if updatedBy != uuid.Nil {
		e.UpdatedByID = &updatedBy
	}
	if err := s.repo.Update(e); err != nil {
		return nil, err
	}
	return s.repo.FindByID(e.ID)
}

func (s *expenseService) Delete(id uuid.UUID, deletedByID uuid.UUID) error {
	if _, err := s.repo.FindByID(id); err != nil {
		return errors.New("expense not found")
	}
	return s.repo.SoftDelete(id, deletedByID)
}

func (s *expenseService) Get(id uuid.UUID) (*models.Expense, error) {
	return s.repo.FindByID(id)
}

func (s *expenseService) List(f repositories.ExpenseFilter, page, pageSize int) ([]models.Expense, int64, error) {
	return s.repo.List(f, page, pageSize)
}

func (s *expenseService) GetSummary() (*models.ExpenseSummary, error) {
	now := time.Now()

	// Today
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)

	// This week (Mon–Sun)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := now.AddDate(0, 0, -(weekday - 1))
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, now.Location())

	// This month
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// This year
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())

	todayTotal, _ := s.repo.Summary(&todayStart, &todayEnd, nil)
	weekTotal, _ := s.repo.Summary(&weekStart, nil, nil)
	monthTotal, _ := s.repo.Summary(&monthStart, nil, nil)
	yearTotal, _ := s.repo.Summary(&yearStart, nil, nil)
	allTimeTotal, _ := s.repo.Summary(nil, nil, nil)
	catTotals, _ := s.repo.CategoryTotals(&monthStart, nil)

	return &models.ExpenseSummary{
		TodayTotal:     todayTotal,
		WeekTotal:      weekTotal,
		MonthTotal:     monthTotal,
		YearTotal:      yearTotal,
		AllTimeTotal:   allTimeTotal,
		CategoryTotals: catTotals,
	}, nil
}
