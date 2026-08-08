package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"github.com/ispcms/backend/internal/repositories"
	"github.com/ispcms/backend/internal/services"
)

type PackageHandler struct{ svc services.PackageService }

func NewPackageHandler(svc services.PackageService) *PackageHandler {
	return &PackageHandler{svc: svc}
}

func (h *PackageHandler) List(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 50)
	filter := repositories.PackageFilter{
		Status: c.Query("status"),
		Search: c.Query("search"),
	}
	packages, total, err := h.svc.List(filter, page, pageSize)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	totalPages := (int(total) + pageSize - 1) / pageSize
	return c.JSON(fiber.Map{
		"success": true, "data": packages, "total": total,
		"page": page, "page_size": pageSize, "total_pages": totalPages,
	})
}

func (h *PackageHandler) ListActive(c *fiber.Ctx) error {
	packages, err := h.svc.ListActive()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": packages})
}

func (h *PackageHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}
	pkg, err := h.svc.Get(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "package not found"})
	}
	return c.JSON(fiber.Map{"success": true, "data": pkg})
}

func (h *PackageHandler) Create(c *fiber.Ctx) error {
	var pkg models.Package
	if err := c.BodyParser(&pkg); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid body"})
	}
	if pkg.PackageName == "" || pkg.DisplayName == "" || pkg.MonthlyPrice <= 0 {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"error":   "package_name, display_name and monthly_price are required",
		})
	}
	if pkg.Status == "" {
		pkg.Status = "active"
	}
	if err := h.svc.Create(&pkg); err != nil {
		return c.Status(409).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"success": true, "data": pkg})
}

func (h *PackageHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}
	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid body"})
	}
	pkg, err := h.svc.Update(id, body)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": pkg})
}

func (h *PackageHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}
	if err := h.svc.Delete(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "package deleted"})
}
