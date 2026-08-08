package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/repositories"
	"github.com/ispcms/backend/pkg/utils"
)

type PPPoEHandler struct {
	pppoeRepo repositories.PPPoERepository
}

func NewPPPoEHandler(pppoeRepo repositories.PPPoERepository) *PPPoEHandler {
	return &PPPoEHandler{pppoeRepo: pppoeRepo}
}

func (h *PPPoEHandler) ListSecrets(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	search := c.Query("search")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	var routerID *uuid.UUID
	if rid := c.Query("router_id"); rid != "" {
		id, err := uuid.Parse(rid)
		if err == nil {
			routerID = &id
		}
	}

	var disabled *bool
	switch c.Query("disabled") {
	case "true":
		t := true
		disabled = &t
	case "false":
		f := false
		disabled = &f
	}

	secrets, total, err := h.pppoeRepo.FindSecrets(routerID, page, pageSize, search, disabled)
	if err != nil {
		return utils.InternalError(c, err)
	}
	return utils.Paginated(c, secrets, total, page, pageSize)
}

func (h *PPPoEHandler) GetSecret(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid secret id")
	}
	secret, err := h.pppoeRepo.FindSecretByID(id)
	if err != nil {
		return utils.NotFound(c, "secret not found")
	}
	return utils.OK(c, secret)
}

func (h *PPPoEHandler) ListSessions(c *fiber.Ctx) error {
	var routerID *uuid.UUID
	if rid := c.Query("router_id"); rid != "" {
		id, err := uuid.Parse(rid)
		if err == nil {
			routerID = &id
		}
	}
	sessions, err := h.pppoeRepo.FindSessions(routerID)
	if err != nil {
		return utils.InternalError(c, err)
	}
	return utils.OK(c, sessions)
}
