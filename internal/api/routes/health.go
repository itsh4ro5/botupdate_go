package routes

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/itsh4ro5/botupdate/internal/database"
)

var startTime = time.Now()

// RegisterHealthRoutes attaches health and status endpoints to the given router.
func RegisterHealthRoutes(router fiber.Router, store database.Store) {
	// Simple health check (public)
	router.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})

	// Readiness check
	router.Get("/ready", func(c *fiber.Ctx) error {
		_, err := store.Load(c.Context())
		if err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "error",
				"reason": "database unavailable",
			})
		}
		return c.JSON(fiber.Map{
			"status": "ready",
		})
	})
}
