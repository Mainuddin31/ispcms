package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/middleware"
	"github.com/ispcms/backend/internal/services"
	"github.com/ispcms/backend/pkg/utils"
)

type AuthHandler struct {
	authSvc services.AuthService
	userSvc services.UserService
}

func NewAuthHandler(authSvc services.AuthService, userSvc services.UserService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, userSvc: userSvc}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type resetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, "invalid request body")
	}
	if req.Username == "" || req.Password == "" {
		return utils.BadRequest(c, "username and password are required")
	}

	tokens, user, err := h.authSvc.Login(req.Username, req.Password)
	if err != nil {
		switch err {
		case services.ErrAccountLocked:
			return utils.Unauthorized(c, "account is temporarily locked")
		case services.ErrAccountDisabled:
			return utils.Unauthorized(c, "account is disabled")
		default:
			return utils.Unauthorized(c, "invalid username or password")
		}
	}

	return utils.OK(c, fiber.Map{
		"tokens": tokens,
		"user":   user,
	})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// JWT is stateless; client should discard the token.
	return utils.OKMessage(c, "logged out successfully", nil)
}

func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req refreshRequest
	if err := c.BodyParser(&req); err != nil || req.RefreshToken == "" {
		return utils.BadRequest(c, "refresh_token is required")
	}
	tokens, err := h.authSvc.RefreshToken(req.RefreshToken)
	if err != nil {
		return utils.Unauthorized(c, "invalid or expired refresh token")
	}
	return utils.OK(c, tokens)
}

func (h *AuthHandler) Profile(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return utils.Unauthorized(c, "unauthenticated")
	}
	user, err := h.userSvc.GetByID(userID)
	if err != nil {
		return utils.NotFound(c, "user not found")
	}
	return utils.OK(c, user)
}

func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		return utils.Unauthorized(c, "unauthenticated")
	}
	var req changePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, "invalid request body")
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		return utils.BadRequest(c, "old_password and new_password are required")
	}
	if len(req.NewPassword) < 8 {
		return utils.BadRequest(c, "new password must be at least 8 characters")
	}
	if err := h.authSvc.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		return utils.BadRequest(c, err.Error())
	}
	return utils.OKMessage(c, "password changed successfully", nil)
}

func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	targetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return utils.BadRequest(c, "invalid user id")
	}
	var req resetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.BadRequest(c, "invalid request body")
	}
	if len(req.NewPassword) < 8 {
		return utils.BadRequest(c, "new password must be at least 8 characters")
	}
	if err := h.authSvc.ResetPassword(targetID, req.NewPassword); err != nil {
		return utils.BadRequest(c, err.Error())
	}
	return utils.OKMessage(c, "password reset successfully", nil)
}
