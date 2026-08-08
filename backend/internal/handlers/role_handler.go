package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/services"
	"github.com/ispcms/backend/pkg/utils"
)

type RoleHandler struct {
	roleSvc services.RoleService
}

func NewRoleHandler(roleSvc services.RoleService) *RoleHandler {
	return &RoleHandler{roleSvc: roleSvc}
}

func (h *RoleHandler) List(c *fiber.Ctx) error {
	roles, err := h.roleSvc.List()
	if err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OK(c, roles)
}

func (h *RoleHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid role id")
	}
	role, err := h.roleSvc.GetByID(id)
	if err != nil {
		return utils.NotFound(c, "role not found")
	}
	return utils.OK(c, role)
}

func (h *RoleHandler) Create(c *fiber.Ctx) error {
	var req services.CreateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, "invalid request body")
	}
	if req.Name == "" || req.DisplayName == "" {
		return utils.BadRequest(c, "name and display_name are required")
	}
	role, err := h.roleSvc.Create(req)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}
	return utils.Created(c, role)
}

func (h *RoleHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid role id")
	}
	var req services.UpdateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, "invalid request body")
	}
	role, err := h.roleSvc.Update(id, req)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}
	return utils.OK(c, role)
}

func (h *RoleHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid role id")
	}
	if err := h.roleSvc.Delete(id); err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OKMessage(c, "role deleted", nil)
}

func (h *RoleHandler) ListPermissions(c *fiber.Ctx) error {
	perms, err := h.roleSvc.ListPermissions()
	if err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OK(c, perms)
}

func (h *RoleHandler) AssignPermission(c *fiber.Ctx) error {
	roleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid role id")
	}
	var body struct {
		PermissionID uuid.UUID `json:"permission_id"`
	}
	if err := c.BodyParser(&body); err != nil || body.PermissionID == uuid.Nil {
		return utils.BadRequest(c, "permission_id is required")
	}
	if err := h.roleSvc.AssignPermission(roleID, body.PermissionID); err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OKMessage(c, "permission assigned", nil)
}

func (h *RoleHandler) RemovePermission(c *fiber.Ctx) error {
	roleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid role id")
	}
	permID, err := uuid.Parse(c.Params("permId"))
	if err != nil {
		return utils.BadRequest(c, "invalid permission id")
	}
	if err := h.roleSvc.RemovePermission(roleID, permID); err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OKMessage(c, "permission removed", nil)
}

func (h *RoleHandler) SetPermissions(c *fiber.Ctx) error {
	roleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid role id")
	}
	var body struct {
		PermissionIDs []uuid.UUID `json:"permission_ids"`
	}
	if err := c.BodyParser(&body); err != nil {
		return utils.BadRequest(c, "invalid request body")
	}
	if err := h.roleSvc.SetPermissions(roleID, body.PermissionIDs); err != nil {
		return utils.BadRequest(c, err.Error())
	}
	return utils.OKMessage(c, "permissions updated", nil)
}

// SetAccountPrefixes handles PUT /api/v1/roles/:id/account-prefixes
func (h *RoleHandler) SetAccountPrefixes(c *fiber.Ctx) error {
	roleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid role id")
	}
	var body struct {
		Prefixes []string `json:"prefixes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return utils.BadRequest(c, "invalid request body")
	}
	if body.Prefixes == nil {
		body.Prefixes = []string{}
	}
	if err := h.roleSvc.SetAccountPrefixes(roleID, body.Prefixes); err != nil {
		return utils.BadRequest(c, err.Error())
	}
	return utils.OKMessage(c, "account prefixes updated", nil)
}
