package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/middleware"
	"github.com/ispcms/backend/internal/services"
	"github.com/ispcms/backend/pkg/utils"
	"strconv"
)

type UserHandler struct {
	userSvc services.UserService
}

func NewUserHandler(userSvc services.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

func (h *UserHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	search := c.Query("search")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	users, total, err := h.userSvc.List(page, pageSize, search)
	if err != nil {
		return utils.InternalError(c, err)
	}
	return utils.Paginated(c, users, total, page, pageSize)
}

func (h *UserHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid user id")
	}
	user, err := h.userSvc.GetByID(id)
	if err != nil {
		return utils.NotFound(c, "user not found")
	}
	return utils.OK(c, user)
}

func (h *UserHandler) Create(c *fiber.Ctx) error {
	var req services.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, "invalid request body")
	}
	if req.FullName == "" || req.Username == "" || req.Email == "" || req.Password == "" {
		return utils.BadRequest(c, "full_name, username, email, and password are required")
	}

	creatorID, _ := middleware.GetCurrentUserID(c)
	user, err := h.userSvc.Create(req, creatorID)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}
	return utils.Created(c, user)
}

func (h *UserHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid user id")
	}
	var req services.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, "invalid request body")
	}
	user, err := h.userSvc.Update(id, req)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}
	return utils.OK(c, user)
}

func (h *UserHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid user id")
	}
	if err := h.userSvc.Delete(id); err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OKMessage(c, "user deleted", nil)
}

func (h *UserHandler) AssignRole(c *fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid user id")
	}
	var body struct {
		RoleID uuid.UUID `json:"role_id"`
	}
	if err := c.BodyParser(&body); err != nil || body.RoleID == uuid.Nil {
		return utils.BadRequest(c, "role_id is required")
	}
	if err := h.userSvc.AssignRole(userID, body.RoleID); err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OKMessage(c, "role assigned", nil)
}

func (h *UserHandler) RemoveRole(c *fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid user id")
	}
	roleID, err := uuid.Parse(c.Params("roleId"))
	if err != nil {
		return utils.BadRequest(c, "invalid role id")
	}
	if err := h.userSvc.RemoveRole(userID, roleID); err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OKMessage(c, "role removed", nil)
}

func (h *UserHandler) SetStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid user id")
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return utils.BadRequest(c, "invalid request body")
	}
	if err := h.userSvc.SetStatus(id, body.Status); err != nil {
		return utils.BadRequest(c, err.Error())
	}
	return utils.OKMessage(c, "status updated", nil)
}
