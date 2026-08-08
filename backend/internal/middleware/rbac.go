package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/repositories"
	jwtpkg "github.com/ispcms/backend/pkg/jwt"
)

// RequirePermission checks that the authenticated user has module.action permission.
func RequirePermission(roleRepo repositories.RoleRepository, module, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(ClaimsKey).(*jwtpkg.Claims)
		if !ok || claims == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "unauthenticated",
			})
		}

		// Super admin bypass
		for _, r := range claims.Roles {
			if r == "super_admin" {
				return c.Next()
			}
		}

		perms, err := roleRepo.GetUserPermissions(claims.UserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "failed to load permissions",
			})
		}

		for _, p := range perms {
			if p.Module == module && p.Action == action {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "permission denied",
		})
	}
}

// RequireRole allows only users that have at least one of the given role names.
func RequireRole(roleNames ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(roleNames))
	for _, r := range roleNames {
		allowed[r] = struct{}{}
	}
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals(ClaimsKey).(*jwtpkg.Claims)
		if !ok || claims == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "unauthenticated",
			})
		}
		for _, r := range claims.Roles {
			if _, ok := allowed[r]; ok {
				return c.Next()
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "insufficient role",
		})
	}
}

// GetCurrentUserID extracts the authenticated user's UUID from context.
func GetCurrentUserID(c *fiber.Ctx) (uuid.UUID, bool) {
	claims, ok := c.Locals(ClaimsKey).(*jwtpkg.Claims)
	if !ok || claims == nil {
		return uuid.Nil, false
	}
	return claims.UserID, true
}
