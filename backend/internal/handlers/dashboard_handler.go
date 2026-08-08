package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ispcms/backend/internal/services"
	"github.com/ispcms/backend/pkg/utils"
)

type DashboardHandler struct {
	dashboardSvc services.DashboardService
	activitySvc  services.ActivityService
}

func NewDashboardHandler(dashboardSvc services.DashboardService, activitySvc services.ActivityService) *DashboardHandler {
	return &DashboardHandler{dashboardSvc: dashboardSvc, activitySvc: activitySvc}
}

func (h *DashboardHandler) Stats(c *fiber.Ctx) error {
	stats, err := h.dashboardSvc.GetStats()
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
