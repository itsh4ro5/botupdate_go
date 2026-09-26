package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/itsh4ro5/botupdate/internal/api/middleware"
	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/services/analytics"
)

func RegisterAnalyticsRoutes(router fiber.Router, store database.Store, analyticsService *analytics.AnalyticsService) {
	group := router.Group("/analytics")

	// Protected routes
	group.Use(middleware.RequireAuth(store))
	group.Use(middleware.RequireRole("OWNER", "ADMIN"))

	group.Get("/overview", func(c *fiber.Ctx) error {
		data, err := analyticsService.GetOverview(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get overview analytics"})
		}
		return c.JSON(data)
	})

	group.Get("/users", func(c *fiber.Ctx) error {
		data, err := analyticsService.GetUserAnalytics(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get user analytics"})
		}
		return c.JSON(data)
	})

	group.Get("/batches", func(c *fiber.Ctx) error {
		data, err := analyticsService.GetBatchAnalytics(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get batch analytics"})
		}
		return c.JSON(data)
	})

	group.Get("/requests", func(c *fiber.Ctx) error {
		data, err := analyticsService.GetRequestAnalytics(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get request analytics"})
		}
		return c.JSON(data)
	})

	group.Get("/support", func(c *fiber.Ctx) error {
		data, err := analyticsService.GetSupportAnalytics(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get support analytics"})
		}
		return c.JSON(data)
	})

	group.Get("/system", func(c *fiber.Ctx) error {
		data := analyticsService.GetSystemStatus()
		return c.JSON(data)
	})

	// Stub activity route since we don't have true activity logs
	group.Get("/activity", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"available": false, "historical_info": "Historical activity tracking is not currently available."})
	})
}
