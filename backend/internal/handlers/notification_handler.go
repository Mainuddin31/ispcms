package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/services"
)

type NotificationHandler struct{ svc services.NotificationService }

func NewNotificationHandler(svc services.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (h *NotificationHandler) List(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 50)
	unreadOnly := c.QueryBool("unread_only", false)

	notifs, total, err := h.svc.List(unreadOnly, page, pageSize)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	totalPages := (int(total) + pageSize - 1) / pageSize
	return c.JSON(fiber.Map{
		"success": true, "data": notifs, "total": total,
		"page": page, "page_size": pageSize, "total_pages": totalPages,
	})
}

func (h *NotificationHandler) CountUnread(c *fiber.Ctx) error {
	count, err := h.svc.CountUnread()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"count": count}})
}

func (h *NotificationHandler) MarkRead(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "error": "invalid id"})
	}
	if err := h.svc.MarkRead(id); err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "error": "notification not found"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "marked as read"})
}

func (h *NotificationHandler) MarkAllRead(c *fiber.Ctx) error {
	if err := h.svc.MarkAllRead(); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "message": "all notifications marked as read"})
}
