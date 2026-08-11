package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/middleware"
	"github.com/ispcms/backend/internal/repositories"
	"github.com/ispcms/backend/internal/services"
)

type VisitHandler struct {
	svc      services.VisitingService
	billRepo repositories.BillRepository
	roleRepo repositories.RoleRepository
}

func NewVisitHandler(svc services.VisitingService, billRepo repositories.BillRepository, roleRepo repositories.RoleRepository) *VisitHandler {
	return &VisitHandler{svc: svc, billRepo: billRepo, roleRepo: roleRepo}
}

// GET /api/v1/visits/pending-customers
// Returns internet accounts whose current-month bill is still pending/due.
func (h *VisitHandler) PendingCustomers(c *fiber.Ctx) error {
	now := time.Now()
	month := c.QueryInt("month", int(now.Month()))
	year := c.QueryInt("year", now.Year())
	customers, err := h.svc.PendingCustomers(month, year)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": customers, "month": month, "year": year})
}

// GET /api/v1/visits/today
// Returns today's scheduled visits. Staff see only their own; admins see all.
func (h *VisitHandler) Today(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "unauthenticated"})
	}

	// Admins (super_admin / admin) see all — regular staff see their own.
	_, isSuperAdmin, _ := h.roleRepo.GetUserAccountPrefixes(userID)
	var staffScope *uuid.UUID
	if !isSuperAdmin {
		staffScope = &userID
	}

	visits, err := h.svc.TodayVisits(staffScope)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": visits, "count": len(visits)})
}

// GET /api/v1/visits
func (h *VisitHandler) List(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 50)
	filter := repositories.VisitFilter{
		Status:   c.Query("status"),
		DateFrom: c.Query("date_from"),
		DateTo:   c.Query("date_to"),
		Search:   c.Query("search"),
	}
	if sid := c.Query("assigned_staff_id"); sid != "" {
		if uid, err := uuid.Parse(sid); err == nil {
			filter.AssignedStaffID = &uid
		}
	}

	// Shorthand date presets
	now := time.Now()
	switch c.Query("date_preset") {
	case "today":
		today := now.Format("2006-01-02")
		filter.DateFrom = today
		filter.DateTo = today
	case "tomorrow":
		tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
		filter.DateFrom = tomorrow
		filter.DateTo = tomorrow
	case "this_week":
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		monday := now.AddDate(0, 0, -(weekday - 1))
		sunday := monday.AddDate(0, 0, 6)
		filter.DateFrom = monday.Format("2006-01-02")
		filter.DateTo = sunday.Format("2006-01-02")
	}

	visits, total, err := h.svc.List(filter, page, pageSize)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	totalPages := (int(total) + pageSize - 1) / pageSize
	return c.JSON(fiber.Map{
		"success": true, "data": visits, "total": total,
		"page": page, "page_size": pageSize, "total_pages": totalPages,
	})
}

// GET /api/v1/visits/:id
func (h *VisitHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}
	v, err := h.svc.Get(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "visit not found"})
	}
	return c.JSON(fiber.Map{"success": true, "data": v})
}

// POST /api/v1/visits
func (h *VisitHandler) Create(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "unauthenticated"})
	}

	var body struct {
		InternetAccountID string `json:"internet_account_id"`
		BillID            string `json:"bill_id"`
		BillingMonth      int    `json:"billing_month"`
		BillingYear       int    `json:"billing_year"`
		AssignedStaffID   string `json:"assigned_staff_id"`
		ScheduledDate     string `json:"scheduled_date"`
		ScheduledTime     string `json:"scheduled_time"`
		Notes             string `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}

	iaID, err := uuid.Parse(body.InternetAccountID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid internet_account_id"})
	}
	billID, err := uuid.Parse(body.BillID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid bill_id"})
	}
	staffID, err := uuid.Parse(body.AssignedStaffID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid assigned_staff_id"})
	}

	visit, err := h.svc.Schedule(services.ScheduleInput{
		InternetAccountID: iaID,
		BillID:            billID,
		BillingMonth:      body.BillingMonth,
		BillingYear:       body.BillingYear,
		AssignedStaffID:   staffID,
		ScheduledDate:     body.ScheduledDate,
		ScheduledTime:     body.ScheduledTime,
		Notes:             body.Notes,
		CreatedBy:         userID,
	})
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "data": visit})
}

// PUT /api/v1/visits/:id
func (h *VisitHandler) Update(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "unauthenticated"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}

	var body struct {
		ScheduledDate   string `json:"scheduled_date"`
		ScheduledTime   string `json:"scheduled_time"`
		AssignedStaffID string `json:"assigned_staff_id"`
		Notes           string `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}

	input := services.RescheduleInput{
		ScheduledDate: body.ScheduledDate,
		ScheduledTime: body.ScheduledTime,
		Notes:         body.Notes,
		RescheduledBy: userID,
	}
	if body.AssignedStaffID != "" {
		if uid, err := uuid.Parse(body.AssignedStaffID); err == nil {
			input.AssignedStaffID = &uid
		}
	}

	visit, err := h.svc.Update(id, input)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": visit})
}

// POST /api/v1/visits/:id/complete
func (h *VisitHandler) Complete(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "unauthenticated"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}

	visit, err := h.svc.Complete(id, userID, h.billRepo)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": visit})
}

// POST /api/v1/visits/:id/reschedule
func (h *VisitHandler) Reschedule(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "unauthenticated"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}

	var body struct {
		ScheduledDate   string `json:"scheduled_date"`
		ScheduledTime   string `json:"scheduled_time"`
		AssignedStaffID string `json:"assigned_staff_id"`
		Notes           string `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid request body"})
	}
	if body.ScheduledDate == "" || body.ScheduledTime == "" {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "scheduled_date and scheduled_time are required for reschedule"})
	}

	input := services.RescheduleInput{
		ScheduledDate: body.ScheduledDate,
		ScheduledTime: body.ScheduledTime,
		Notes:         body.Notes,
		RescheduledBy: userID,
	}
	if body.AssignedStaffID != "" {
		if uid, err := uuid.Parse(body.AssignedStaffID); err == nil {
			input.AssignedStaffID = &uid
		}
	}

	visit, err := h.svc.Reschedule(id, input, h.billRepo)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "data": visit})
}

// POST /api/v1/visits/:id/cancel
func (h *VisitHandler) Cancel(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"success": false, "error": "unauthenticated"})
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}

	visit, err := h.svc.Cancel(id, userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": visit})
}

// GET /api/v1/internet-accounts/:id/visits (customer visiting history)
func (h *VisitHandler) ByAccount(c *fiber.Ctx) error {
	iaID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid account id"})
	}
	visits, err := h.svc.VisitsByAccount(iaID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": visits})
}
