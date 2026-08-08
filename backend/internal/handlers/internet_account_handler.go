package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/repositories"
	"github.com/ispcms/backend/internal/services"
)

type InternetAccountHandler struct {
	repo    repositories.InternetAccountRepository
	syncSvc services.SyncService
}

func NewInternetAccountHandler(
	repo repositories.InternetAccountRepository,
	syncSvc services.SyncService,
) *InternetAccountHandler {
	return &InternetAccountHandler{repo: repo, syncSvc: syncSvc}
}

// List handles GET /api/v1/internet-accounts
func (h *InternetAccountHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	filter := repositories.InternetAccountFilter{
		Search:  c.Query("search"),
		Profile: c.Query("profile"),
	}

	if rid := c.Query("router_id"); rid != "" {
		parsed, err := uuid.Parse(rid)
		if err == nil {
			filter.RouterID = &parsed
		}
	}

	if v := c.Query("is_online"); v != "" {
		b := v == "true" || v == "1"
		filter.IsOnline = &b
	}

	if v := c.Query("disabled"); v != "" {
		b := v == "true" || v == "1"
		filter.Disabled = &b
	}

	if v := c.Query("archived"); v != "" {
		b := v == "true" || v == "1"
		filter.Archived = &b
	}

	accounts, total, err := h.repo.List(filter, page, pageSize)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	return c.JSON(fiber.Map{
		"success":     true,
		"data":        accounts,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}

// Get handles GET /api/v1/internet-accounts/:id
func (h *InternetAccountHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	account, err := h.repo.GetByID(id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "account not found")
	}
	return c.JSON(fiber.Map{"success": true, "data": account})
}

// Stats handles GET /api/v1/internet-accounts/stats
func (h *InternetAccountHandler) Stats(c *fiber.Ctx) error {
	total, enabled, disabled, online, offline, archived, err := h.repo.CountStats()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"total":    total,
			"enabled":  enabled,
			"disabled": disabled,
			"online":   online,
			"offline":  offline,
			"archived": archived,
		},
	})
}

// Profiles handles GET /api/v1/internet-accounts/profiles
func (h *InternetAccountHandler) Profiles(c *fiber.Ctx) error {
	profiles, err := h.repo.ListProfiles()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"success": true, "data": profiles})
}

// SyncAll handles POST /api/v1/internet-accounts/sync-all
func (h *InternetAccountHandler) SyncAll(c *fiber.Ctx) error {
	summary, _ := h.syncSvc.SyncAllRouters()
	return c.JSON(fiber.Map{"success": true, "data": summary})
}
