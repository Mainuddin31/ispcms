package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"github.com/ispcms/backend/internal/repositories"
)

// ScheduleInput holds the fields required to create a new visit.
type ScheduleInput struct {
	InternetAccountID uuid.UUID
	BillID            uuid.UUID
	BillingMonth      int
	BillingYear       int
	AssignedStaffID   uuid.UUID
	ScheduledDate     string // "YYYY-MM-DD"
	ScheduledTime     string // "HH:MM"
	Notes             string
	CreatedBy         uuid.UUID
}

// RescheduleInput holds new scheduling data for an existing visit.
type RescheduleInput struct {
	ScheduledDate   string
	ScheduledTime   string
	AssignedStaffID *uuid.UUID // nil = keep current
	Notes           string
	RescheduledBy   uuid.UUID
}

type VisitingService interface {
	// PendingCustomers returns accounts with a pending bill for the given month/year.
	PendingCustomers(month, year int) ([]models.PendingVisitCustomer, error)
	// Schedule creates a new visit. Returns error if an active visit already exists for the bill.
	Schedule(input ScheduleInput) (*models.Visit, error)
	// Get returns a single visit with preloaded associations.
	Get(id uuid.UUID) (*models.Visit, error)
	// List returns visits with optional filtering.
	List(filter repositories.VisitFilter, page, pageSize int) ([]models.Visit, int64, error)
	// Update updates mutable fields (date, time, staff, notes) of a Scheduled visit.
	Update(id uuid.UUID, input RescheduleInput) (*models.Visit, error)
	// Complete marks a visit Completed. Checks live bill status — rejects if still unpaid.
	Complete(id uuid.UUID, completedBy uuid.UUID, billRepo repositories.BillRepository) (*models.Visit, error)
	// Reschedule closes the current visit as Rescheduled and creates the next one.
	Reschedule(id uuid.UUID, input RescheduleInput, billRepo repositories.BillRepository) (*models.Visit, error)
	// Cancel marks a visit Cancelled.
	Cancel(id uuid.UUID, cancelledBy uuid.UUID) (*models.Visit, error)
	// TodayVisits returns today's visits, scoped to staffID when non-nil.
	TodayVisits(staffID *uuid.UUID) ([]models.Visit, error)
	// TodayCount returns the count of today's scheduled visits.
	TodayCount(staffID *uuid.UUID) (int64, error)
	// VisitsByAccount returns the visiting history for an internet account.
	VisitsByAccount(internetAccountID uuid.UUID) ([]models.Visit, error)
}

type visitingService struct {
	repo        repositories.VisitRepository
	activitySvc ActivityService
}

func NewVisitingService(repo repositories.VisitRepository, activitySvc ActivityService) VisitingService {
	return &visitingService{repo: repo, activitySvc: activitySvc}
}

func (s *visitingService) PendingCustomers(month, year int) ([]models.PendingVisitCustomer, error) {
	return s.repo.PendingCustomers(month, year)
}

func (s *visitingService) Schedule(input ScheduleInput) (*models.Visit, error) {
	if input.ScheduledDate == "" || input.ScheduledTime == "" {
		return nil, errors.New("scheduled_date and scheduled_time are required")
	}
	if input.AssignedStaffID == uuid.Nil {
		return nil, errors.New("assigned_staff_id is required")
	}
	// Prevent duplicate active visits for the same bill.
	existing, err := s.repo.HasActiveVisitForBill(input.BillID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("this customer already has an active scheduled visit (id: %s)", existing.ID)
	}

	v := &models.Visit{
		InternetAccountID: input.InternetAccountID,
		BillID:            input.BillID,
		BillingMonth:      input.BillingMonth,
		BillingYear:       input.BillingYear,
		AssignedStaffID:   input.AssignedStaffID,
		ScheduledDate:     input.ScheduledDate,
		ScheduledTime:     input.ScheduledTime,
		Notes:             input.Notes,
		Status:            "Scheduled",
		CreatedBy:         input.CreatedBy,
	}
	if err := s.repo.Create(v); err != nil {
		return nil, err
	}

	// Reload with associations for activity log description.
	full, _ := s.repo.FindByID(v.ID)
	if full != nil {
		username := ""
		if full.InternetAccount != nil {
			username = full.InternetAccount.Username
		}
		staffName := ""
		if full.AssignedStaff != nil {
			staffName = full.AssignedStaff.FullName
		}
		s.activitySvc.Log(
			&input.CreatedBy,
			"visiting", "visit_scheduled",
			fmt.Sprintf("Visit scheduled for %s", username),
			fmt.Sprintf("Date: %s %s, Staff: %s", input.ScheduledDate, input.ScheduledTime, staffName),
			"visit", v.ID.String(),
		)
	}
	return full, nil
}

func (s *visitingService) Get(id uuid.UUID) (*models.Visit, error) {
	return s.repo.FindByID(id)
}

func (s *visitingService) List(filter repositories.VisitFilter, page, pageSize int) ([]models.Visit, int64, error) {
	return s.repo.List(filter, page, pageSize)
}

func (s *visitingService) Update(id uuid.UUID, input RescheduleInput) (*models.Visit, error) {
	v, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("visit not found")
	}
	if v.Status == "Completed" || v.Status == "Cancelled" {
		return nil, fmt.Errorf("cannot update a %s visit", v.Status)
	}
	if input.ScheduledDate != "" {
		v.ScheduledDate = input.ScheduledDate
	}
	if input.ScheduledTime != "" {
		v.ScheduledTime = input.ScheduledTime
	}
	if input.AssignedStaffID != nil {
		v.AssignedStaffID = *input.AssignedStaffID
	}
	if input.Notes != "" {
		v.Notes = input.Notes
	}
	v.UpdatedBy = &input.RescheduledBy
	if err := s.repo.Update(v); err != nil {
		return nil, err
	}
	return s.repo.FindByID(id)
}

func (s *visitingService) Complete(id uuid.UUID, completedBy uuid.UUID, billRepo repositories.BillRepository) (*models.Visit, error) {
	v, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("visit not found")
	}
	if v.Status != "Scheduled" && v.Status != "Rescheduled" {
		return nil, fmt.Errorf("cannot complete a visit with status %s", v.Status)
	}

	// Always query live bill status — never trust cached state.
	bill, err := billRepo.FindByID(v.BillID)
	if err != nil {
		return nil, errors.New("associated bill not found")
	}
	if bill.Status != "paid" {
		return nil, errors.New("customer bill is still unpaid — please reschedule the visit")
	}

	now := time.Now()
	v.Status = "Completed"
	v.CompletedBy = &completedBy
	v.CompletedAt = &now
	v.UpdatedBy = &completedBy
	if err := s.repo.Update(v); err != nil {
		return nil, err
	}

	full, _ := s.repo.FindByID(id)
	username := ""
	if full != nil && full.InternetAccount != nil {
		username = full.InternetAccount.Username
	}
	s.activitySvc.Log(
		&completedBy,
		"visiting", "visit_completed",
		fmt.Sprintf("Visit completed for %s", username),
		fmt.Sprintf("Bill: %d/%d — Payment Status: Paid", v.BillingMonth, v.BillingYear),
		"visit", id.String(),
	)
	return full, nil
}

func (s *visitingService) Reschedule(id uuid.UUID, input RescheduleInput, billRepo repositories.BillRepository) (*models.Visit, error) {
	v, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("visit not found")
	}
	if v.Status != "Scheduled" && v.Status != "Rescheduled" {
		return nil, fmt.Errorf("cannot reschedule a visit with status %s", v.Status)
	}

	// Verify bill is still unpaid (optional check — reschedule is also allowed when bill is unpaid).
	// We do NOT block reschedule even if bill is paid; the staff may call reschedule before
	// clicking Complete. The Complete endpoint enforces the paid check.

	now := time.Now()
	oldDate := v.ScheduledDate
	oldTime := v.ScheduledTime

	v.Status = "Rescheduled"
	v.RescheduledBy = &input.RescheduledBy
	v.RescheduledAt = &now
	v.UpdatedBy = &input.RescheduledBy
	if err := s.repo.Update(v); err != nil {
		return nil, err
	}

	// Create the next visit.
	newStaffID := v.AssignedStaffID
	if input.AssignedStaffID != nil {
		newStaffID = *input.AssignedStaffID
	}
	next := &models.Visit{
		InternetAccountID: v.InternetAccountID,
		BillID:            v.BillID,
		BillingMonth:      v.BillingMonth,
		BillingYear:       v.BillingYear,
		AssignedStaffID:   newStaffID,
		ScheduledDate:     input.ScheduledDate,
		ScheduledTime:     input.ScheduledTime,
		Notes:             input.Notes,
		Status:            "Scheduled",
		CreatedBy:         input.RescheduledBy,
	}
	if err := s.repo.Create(next); err != nil {
		return nil, err
	}

	username := ""
	if v.InternetAccount != nil {
		username = v.InternetAccount.Username
	}
	s.activitySvc.Log(
		&input.RescheduledBy,
		"visiting", "visit_rescheduled",
		fmt.Sprintf("Visit rescheduled for %s", username),
		fmt.Sprintf("Old: %s %s → New: %s %s", oldDate, oldTime, input.ScheduledDate, input.ScheduledTime),
		"visit", id.String(),
	)
	return s.repo.FindByID(next.ID)
}

func (s *visitingService) Cancel(id uuid.UUID, cancelledBy uuid.UUID) (*models.Visit, error) {
	v, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("visit not found")
	}
	if v.Status == "Completed" || v.Status == "Cancelled" {
		return nil, fmt.Errorf("visit is already %s", v.Status)
	}
	v.Status = "Cancelled"
	v.UpdatedBy = &cancelledBy
	if err := s.repo.Update(v); err != nil {
		return nil, err
	}

	username := ""
	if v.InternetAccount != nil {
		username = v.InternetAccount.Username
	}
	s.activitySvc.Log(
		&cancelledBy,
		"visiting", "visit_cancelled",
		fmt.Sprintf("Visit cancelled for %s", username),
		fmt.Sprintf("Was scheduled for %s %s", v.ScheduledDate, v.ScheduledTime),
		"visit", id.String(),
	)
	return s.repo.FindByID(id)
}

func (s *visitingService) TodayVisits(staffID *uuid.UUID) ([]models.Visit, error) {
	return s.repo.TodayVisits(staffID)
}

func (s *visitingService) TodayCount(staffID *uuid.UUID) (int64, error) {
	return s.repo.TodayCount(staffID)
}

func (s *visitingService) VisitsByAccount(internetAccountID uuid.UUID) ([]models.Visit, error) {
	return s.repo.VisitsByAccount(internetAccountID)
}
