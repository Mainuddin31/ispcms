package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"github.com/ispcms/backend/internal/repositories"
	"github.com/ispcms/backend/internal/services"
)

type ProfileMappingHandler struct{ svc services.ProfileMappingService }

func NewProfileMappingHandler(svc services.ProfileMappingService) *ProfileMappingHandler {
	return &ProfileMappingHandler{svc: svc}
}

func (h *ProfileMappingHandler) List(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 50)
	filter := repositories.ProfileMappingFilter{
		PackageID: c.Query("package_id"),
		Search:    c.Query("search"),
	}
	mappings, total, err := h.svc.List(filter, page, pageSize)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	totalPages := (int(total) + pageSize - 1) / pageSize
	return c.JSON(fiber.Map{
		"success": true, "data": mappings, "total": total,
		"page": page, "page_size": pageSize, "total_pages": totalPages,
	})
}

func (h *ProfileMappingHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}
	m, err := h.svc.Get(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "mapping not found"})
	}
	return c.JSON(fiber.Map{"success": true, "data": m})
}

func (h *ProfileMappingHandler) Create(c *fiber.Ctx) error {
	var m models.ProfileMapping
	if err := c.BodyParser(&m); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid body"})
	}
	if m.MikrotikProfile == "" || m.PackageID == uuid.Nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "mikrotik_profile and package_id are required",
		})
	}
	if err := h.svc.Create(&m); err != nil {
		return c.Status(409).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "data": m})
}

func (h *ProfileMappingHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid body"})
	}
	m, err := h.svc.Update(id, body)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": m})
}

func (h *ProfileMappingHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}
	if err := h.svc.Delete(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "mapping deleted"})
}

func (h *ProfileMappingHandler) UnmappedProfiles(c *fiber.Ctx) error {
	profiles, err := h.svc.UnmappedProfiles()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": profiles, "total": len(profiles)})
}
