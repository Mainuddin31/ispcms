package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ispcms/backend/internal/middleware"
	"github.com/ispcms/backend/internal/repositories"
	"github.com/ispcms/backend/internal/services"
	"github.com/ispcms/backend/pkg/utils"
)

type DashboardHandler struct {
	dashboardSvc services.DashboardService
	activitySvc  services.ActivityService
	roleRepo     repositories.RoleRepository
}

func NewDashboardHandler(dashboardSvc services.DashboardService, activitySvc services.ActivityService, roleRepo repositories.RoleRepository) *DashboardHandler {
	return &DashboardHandler{dashboardSvc: dashboardSvc, activitySvc: activitySvc, roleRepo: roleRepo}
}

func (h *DashboardHandler) Stats(c *fiber.Ctx) error {
	var prefixes []string
	prefixRestricted := false

	if userID, ok := middleware.GetCurrentUserID(c); ok {
		if p, isSuperAdmin, _ := h.roleRepo.GetUserAccountPrefixes(userID); !isSuperAdmin {
			prefixRestricted = true
			prefixes = p
		}
	}

	stats, err := h.dashboardSvc.GetStats(prefixes, prefixRestricted)
	if err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OK(c, stats)
}

func (h *DashboardHandler) Activities(c *fiber.Ctx) error {
	module := c.Query("module", "")
	period := c.Query("period", "30days")
	limit := c.QueryInt("limit", 30)

	activities, err := h.activitySvc.List(module, period, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": activities, "total": len(activities)})
}
