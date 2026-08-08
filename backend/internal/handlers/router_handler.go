package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/services"
	"github.com/ispcms/backend/pkg/utils"
)

type RouterHandler struct {
	routerSvc services.RouterService
	syncSvc   services.SyncService
}

func NewRouterHandler(routerSvc services.RouterService, syncSvc services.SyncService) *RouterHandler {
	return &RouterHandler{routerSvc: routerSvc, syncSvc: syncSvc}
}

func (h *RouterHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	search := c.Query("search")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	routers, total, err := h.routerSvc.List(page, pageSize, search)
	if err != nil {
		return utils.InternalError(c, err)
	}
	return utils.Paginated(c, routers, total, page, pageSize)
}

func (h *RouterHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid router id")
	}
	router, err := h.routerSvc.GetByID(id)
	if err != nil {
		return utils.NotFound(c, "router not found")
	}
	return utils.OK(c, router)
}

func (h *RouterHandler) Create(c *fiber.Ctx) error {
	var req services.CreateRouterRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, "invalid request body")
	}
	if req.Name == "" || req.IPAddress == "" || req.Username == "" || req.Password == "" {
		return utils.BadRequest(c, "name, ip_address, username, and password are required")
	}
	router, err := h.routerSvc.Create(req)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}
	return utils.Created(c, router)
}

func (h *RouterHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid router id")
	}
	var req services.UpdateRouterRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, "invalid request body")
	}
	router, err := h.routerSvc.Update(id, req)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}
	return utils.OK(c, router)
}

func (h *RouterHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid router id")
	}
	if err := h.routerSvc.Delete(id); err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OKMessage(c, "router deleted", nil)
}

func (h *RouterHandler) TestConnection(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid router id")
	}
	if err := h.routerSvc.TestConnection(id); err != nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success":   false,
			"connected": false,
			"error":     err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"success":   true,
		"connected": true,
		"message":   "connection successful",
	})
}

func (h *RouterHandler) TestConnectionRaw(c *fiber.Ctx) error {
	var body struct {
		IPAddress string `json:"ip_address"`
		APIPort   int    `json:"api_port"`
		Username  string `json:"username"`
		Password  string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return utils.BadRequest(c, "invalid request body")
	}
	port := body.APIPort
	if port == 0 {
		port = 8728
	}
	if err := h.routerSvc.TestConnectionRaw(body.IPAddress, port, body.Username, body.Password); err != nil {
		return c.JSON(fiber.Map{"success": false, "connected": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "connected": true, "message": "connection successful"})
}

func (h *RouterHandler) Sync(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid router id")
	}
	log, err := h.syncSvc.SyncRouter(id)
	if err != nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
			"log":     log,
		})
	}
	return utils.OK(c, fiber.Map{"message": "sync completed", "log": log})
}

func (h *RouterHandler) GetSyncLogs(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid router id")
	}
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	logs, err := h.syncSvc.GetSyncLogs(&id, limit)
	if err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OK(c, logs)
}
