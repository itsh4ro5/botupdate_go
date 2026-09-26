package middleware

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/itsh4ro5/botupdate/internal/database"
)

func RequireAuth(store database.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sessionID := c.Cookies("session_id")
		if sessionID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		state, err := store.Load(context.Background())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
		}

		session, exists := state.WebSessions[sessionID]
		if !exists || time.Now().After(session.ExpiresAt) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		// Attach role to locals for further checks
		c.Locals("role", session.Role)
		c.Locals("admin_id", session.AdminID)
		c.Locals("username", session.AdminID)
		c.Locals("session_id", sessionID)

		return c.Next()
	}
}

func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
		}

		// Owner always has access
		if role == "OWNER" {
			return c.Next()
		}

		for _, allowed := range allowedRoles {
			if role == allowed {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden"})
	}
}
