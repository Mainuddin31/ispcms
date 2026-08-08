package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/repositories"
	"github.com/ispcms/backend/internal/services"
)

type ExpenseHandler struct {
	svc         services.ExpenseService
	catSvc      services.ExpenseCategoryService
	activitySvc services.ActivityService
}

func NewExpenseHandler(svc services.ExpenseService, catSvc services.ExpenseCategoryService, activitySvc services.ActivityService) *ExpenseHandler {
	return &ExpenseHandler{svc: svc, catSvc: catSvc, activitySvc: activitySvc}
}

// ── Categories ────────────────────────────────────────────────────────────────

func (h *ExpenseHandler) ListCategories(c *fiber.Ctx) error {
	cats, err := h.catSvc.List(c.Query("status", "active"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": cats, "total": len(cats)})
}

func (h *ExpenseHandler) CreateCategory(c *fiber.Ctx) error {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid body"})
	}
	cat, err := h.catSvc.Create(body.Name, body.Description)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "data": cat})
}

func (h *ExpenseHandler) UpdateCategory(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid body"})
	}
	cat, err := h.catSvc.Update(id, body.Name, body.Description, body.Status)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": cat})
}

func (h *ExpenseHandler) DeleteCategory(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}
	if err := h.catSvc.Delete(id); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "category deleted"})
}

// ── Expenses ──────────────────────────────────────────────────────────────────

func (h *ExpenseHandler) List(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 25)

	f := repositories.ExpenseFilter{
		Search:        c.Query("search"),
		PaymentMethod: c.Query("payment_method"),
		SortBy:        c.Query("sort_by", "date"),
		SortDir:       c.Query("sort_dir", "desc"),
	}

	if id := c.Query("category_id"); id != "" {
		uid, err := uuid.Parse(id)
		if err == nil {
			f.CategoryID = &uid
		}
	}
	if id := c.Query("user_id"); id != "" {
		uid, err := uuid.Parse(id)
		if err == nil {
			f.UserID = &uid
		}
	}
	if s := c.Query("date_from"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			f.DateFrom = &t
		}
	}
	if s := c.Query("date_to"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			end := t.Add(24*time.Hour - time.Second)
			f.DateTo = &end
		}
	}
	if s := c.Query("amount_min"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
			f.AmountMin = &v
		}
	}
	if s := c.Query("amount_max"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
			f.AmountMax = &v
		}
	}

	expenses, total, err := h.svc.List(f, page, pageSize)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	totalPages := (int(total) + pageSize - 1) / pageSize
	return c.JSON(fiber.Map{
		"success": true, "data": expenses, "total": total,
		"page": page, "page_size": pageSize, "total_pages": totalPages,
	})
}

func (h *ExpenseHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}
	e, err := h.svc.Get(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "expense not found"})
	}
	return c.JSON(fiber.Map{"success": true, "data": e})
}

func (h *ExpenseHandler) Create(c *fiber.Ctx) error {
	var body struct {
		ExpenseDate     string  `json:"expense_date"`
		CategoryID      string  `json:"category_id"`
		Amount          float64 `json:"amount"`
		PaymentMethod   string  `json:"payment_method"`
		Vendor          string  `json:"vendor"`
		ReferenceNumber string  `json:"reference_number"`
		Description     string  `json:"description"`
		AttachmentPath  string  `json:"attachment_path"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid body"})
	}

	catID, err := uuid.Parse(body.CategoryID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid category_id"})
	}

	expDate := time.Now()
	if body.ExpenseDate != "" {
		if t, err := time.Parse("2006-01-02", body.ExpenseDate); err == nil {
			expDate = t
		}
	}

	userID, _ := c.Locals("userID").(uuid.UUID)

	expense, err := h.svc.Create(services.CreateExpenseInput{
		ExpenseDate:     expDate,
		CategoryID:      catID,
		Amount:          body.Amount,
		PaymentMethod:   body.PaymentMethod,
		Vendor:          body.Vendor,
		ReferenceNumber: body.ReferenceNumber,
		Description:     body.Description,
		AttachmentPath:  body.AttachmentPath,
		CreatedByID:     userID,
	})
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if h.activitySvc != nil {
		desc := fmt.Sprintf("Expense: %s | Amount: %.2f | Method: %s", expense.ExpenseNumber, expense.Amount, expense.PaymentMethod)
		h.activitySvc.Log(&userID, "expenses", "expense_created", "Expense Recorded", desc, "expense", expense.ID.String())
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "data": expense})
}

func (h *ExpenseHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}
	var body struct {
		ExpenseDate     string   `json:"expense_date"`
		CategoryID      string   `json:"category_id"`
		Amount          *float64 `json:"amount"`
		PaymentMethod   string   `json:"payment_method"`
		Vendor          string   `json:"vendor"`
		ReferenceNumber string   `json:"reference_number"`
		Description     string   `json:"description"`
		AttachmentPath  string   `json:"attachment_path"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid body"})
	}

	userID, _ := c.Locals("userID").(uuid.UUID)
	input := services.UpdateExpenseInput{
		Amount:          body.Amount,
		PaymentMethod:   body.PaymentMethod,
		Vendor:          body.Vendor,
		ReferenceNumber: body.ReferenceNumber,
		Description:     body.Description,
		AttachmentPath:  body.AttachmentPath,
		UpdatedByID:     userID,
	}
	if body.ExpenseDate != "" {
		if t, err := time.Parse("2006-01-02", body.ExpenseDate); err == nil {
			input.ExpenseDate = &t
		}
	}
	if body.CategoryID != "" {
		if uid, err := uuid.Parse(body.CategoryID); err == nil {
			input.CategoryID = &uid
		}
	}

	expense, err := h.svc.Update(id, input)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if h.activitySvc != nil {
		desc := fmt.Sprintf("Expense: %s | Amount: %.2f", expense.ExpenseNumber, expense.Amount)
		h.activitySvc.Log(&userID, "expenses", "expense_updated", "Expense Updated", desc, "expense", expense.ID.String())
	}
	return c.JSON(fiber.Map{"success": true, "data": expense})
}

func (h *ExpenseHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}
	userID, _ := c.Locals("userID").(uuid.UUID)
	if err := h.svc.Delete(id, userID); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	if h.activitySvc != nil {
		h.activitySvc.Log(&userID, "expenses", "expense_deleted", "Expense Deleted", "Expense ID: "+id.String(), "expense", id.String())
	}
	return c.JSON(fiber.Map{"success": true, "message": "expense deleted"})
}

func (h *ExpenseHandler) Summary(c *fiber.Ctx) error {
	summary, err := h.svc.GetSummary()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": summary})
}
