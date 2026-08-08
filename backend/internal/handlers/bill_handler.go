package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/repositories"
	"github.com/ispcms/backend/internal/services"
)

type BillHandler struct{ svc services.BillingService }

func NewBillHandler(svc services.BillingService) *BillHandler {
	return &BillHandler{svc: svc}
}

func (h *BillHandler) List(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 50)

	filter := repositories.BillFilter{
		Status:  c.Query("status"),
		Month:   c.QueryInt("month", 0),
		Year:    c.QueryInt("year", 0),
		Search:  c.Query("search"),
	}
	if id := c.Query("internet_account_id"); id != "" {
		uid, err := uuid.Parse(id)
		if err == nil {
			filter.InternetAccountID = &uid
		}
	}
	if id := c.Query("package_id"); id != "" {
		uid, err := uuid.Parse(id)
		if err == nil {
			filter.PackageID = &uid
		}
	}

	bills, total, err := h.svc.ListBills(filter, page, pageSize)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	totalPages := (int(total) + pageSize - 1) / pageSize
	return c.JSON(fiber.Map{
		"success": true, "data": bills, "total": total,
		"page": page, "page_size": pageSize, "total_pages": totalPages,
	})
}

func (h *BillHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}
	bill, err := h.svc.GetBill(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "bill not found"})
	}
	return c.JSON(fiber.Map{"success": true, "data": bill})
}

func (h *BillHandler) GenerateBills(c *fiber.Ctx) error {
	var body struct {
		Month int `json:"month"`
		Year  int `json:"year"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid body"})
	}
	now := time.Now()
	if body.Month == 0 {
		body.Month = int(now.Month())
	}
	if body.Year == 0 {
		body.Year = now.Year()
	}
	if body.Month < 1 || body.Month > 12 {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "month must be 1-12"})
	}

	// Get generating user ID from JWT locals (set by auth middleware)
	var generatedBy *uuid.UUID
	if userID, ok := c.Locals("userID").(uuid.UUID); ok {
		generatedBy = &userID
	}

	genLog, err := h.svc.GenerateBills(services.GenerateBillsRequest{
		Month:       body.Month,
		Year:        body.Year,
		GeneratedBy: generatedBy,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": genLog})
}

func (h *BillHandler) UpdateStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}
	var body struct {
		Status        string   `json:"status"`
		PaidAmount    *float64 `json:"paid_amount"`
		Notes         string   `json:"notes"`
		PaymentMethod string   `json:"payment_method"`
		ReceiptNumber string   `json:"receipt_number"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid body"})
	}
	// Capture who received the payment.
	var receivedByID *uuid.UUID
	if userID, ok := c.Locals("userID").(uuid.UUID); ok {
		receivedByID = &userID
	}
	bill, err := h.svc.UpdateBillStatus(id, body.Status, body.PaidAmount, body.Notes, body.PaymentMethod, body.ReceiptNumber, receivedByID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": bill})
}

func (h *BillHandler) BillingHistory(c *fiber.Ctx) error {
	accountID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid account id"})
	}
	entries, err := h.svc.GetBillingHistory(accountID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": entries, "total": len(entries)})
}

func (h *BillHandler) PaymentHistory(c *fiber.Ctx) error {
	accountID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid account id"})
	}
	records, err := h.svc.GetPaymentHistory(accountID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": records, "total": len(records)})
}

func (h *BillHandler) ListGenerationLogs(c *fiber.Ctx) error {
	month := c.QueryInt("month", 0)
	year := c.QueryInt("year", 0)
	limit := c.QueryInt("limit", 20)

	logs, err := h.svc.ListGenerationLogs(month, year, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": logs, "total": len(logs)})
}

func (h *BillHandler) BillingStatus(c *fiber.Ctx) error {
	now := time.Now()
	month := c.QueryInt("month", int(now.Month()))
	year := c.QueryInt("year", now.Year())

	generated, pending, err := h.svc.CheckMonthlyBillingStatus(month, year)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"success": true, "data": fiber.Map{
			"month": month, "year": year,
			"bills_generated": generated, "bills_pending": pending,
		},
	})
}

// ── Subscriptions ─────────────────────────────────────────────────────────────

func (h *BillHandler) ListSubscriptions(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 50)
	activeOnly := c.QueryBool("active_only", false)

	var accountID *uuid.UUID
	if id := c.Query("internet_account_id"); id != "" {
		uid, err := uuid.Parse(id)
		if err == nil {
			accountID = &uid
		}
	}
	var packageID *uuid.UUID
	if id := c.Query("package_id"); id != "" {
		uid, err := uuid.Parse(id)
		if err == nil {
			packageID = &uid
		}
	}

	subs, total, err := h.svc.ListSubscriptions(page, pageSize, accountID, packageID, activeOnly)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	totalPages := (int(total) + pageSize - 1) / pageSize
	return c.JSON(fiber.Map{
		"success": true, "data": subs, "total": total,
		"page": page, "page_size": pageSize, "total_pages": totalPages,
	})
}

func (h *BillHandler) AssignSubscription(c *fiber.Ctx) error {
	var body struct {
		InternetAccountID string `json:"internet_account_id"`
		PackageID         string `json:"package_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid body"})
	}
	accountID, err := uuid.Parse(body.InternetAccountID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid internet_account_id"})
	}
	packageID, err := uuid.Parse(body.PackageID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid package_id"})
	}
	sub, err := h.svc.AssignSubscription(accountID, packageID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "data": sub})
}

func (h *BillHandler) AutoAssignSubscriptions(c *fiber.Ctx) error {
	result, err := h.svc.AutoAssignFromProfiles()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": result})
}

func (h *BillHandler) GetActiveSubscription(c *fiber.Ctx) error {
	accountID, err := uuid.Parse(c.Params("accountId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid account id"})
	}
	sub, err := h.svc.GetActiveSubscription(accountID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "no active subscription"})
	}
	return c.JSON(fiber.Map{"success": true, "data": sub})
}
