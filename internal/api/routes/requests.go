package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/itsh4ro5/botupdate/internal/api/middleware"
	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/services/requests"
)

func RegisterRequestRoutes(router fiber.Router, store database.Store, requestService *requests.RequestService) {
	reqGroup := router.Group("/requests")

	// Protected routes
	reqGroup.Use(middleware.RequireAuth(store))

	reqGroup.Get("/", func(c *fiber.Ctx) error {
		page := c.QueryInt("page", 1)
		pageSize := c.QueryInt("pageSize", 25)
		search := c.Query("search", "")
		batchIDStr := c.Query("batch", "")

		list, err := requestService.ListRequests(c.Context(), page, pageSize, search, batchIDStr)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list requests"})
		}

		return c.JSON(list)
	})

	reqGroup.Get("/:id", func(c *fiber.Ctx) error {
		idStr := c.Params("id")

		detail, err := requestService.GetRequest(c.Context(), idStr)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get request"})
		}
		if detail == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Request not found"})
		}

		return c.JSON(detail)
	})

	// Mutation routes
	reqGroup.Post("/:id/approve", middleware.RequireRole("OWNER", "ADMIN"), func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		adminID, ok := getStringLocalSafe(c, "admin_id")
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		err := requestService.ApproveRequest(c.Context(), idStr, adminID)
		if err != nil {
			if err.Error() == "request not found" {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"success": true})
	})

	reqGroup.Post("/:id/reject", middleware.RequireRole("OWNER", "ADMIN"), func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		adminID, ok := getStringLocalSafe(c, "admin_id")
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		err := requestService.RejectRequest(c.Context(), idStr, adminID)
		if err != nil {
			if err.Error() == "request not found" {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"success": true})
	})
}
