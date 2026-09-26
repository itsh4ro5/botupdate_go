package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/itsh4ro5/botupdate/internal/api/middleware"
	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/services/dashboard"
)

func RegisterDashboardRoutes(router fiber.Router, store database.Store) {
	dashboardService := dashboard.NewService(store)

	// Protected dashboard routes
	dashboardGroup := router.Group("/dashboard", middleware.RequireAuth(store), middleware.RequireRole("OWNER", "ADMIN"))

	dashboardGroup.Get("/overview", func(c *fiber.Ctx) error {
		overview, err := dashboardService.GetOverview(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Database unavailable",
			})
		}

		// Fill missing system states from process context
		overview.System.Telegram = "online"
		overview.System.Scheduler = "online"
		overview.System.Userbot = "online"

		return c.JSON(overview)
	})
}
