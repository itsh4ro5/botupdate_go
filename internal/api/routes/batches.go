package routes

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/itsh4ro5/botupdate/internal/api/middleware"
	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/events"
	"github.com/itsh4ro5/botupdate/internal/services/batches"
)

func RegisterBatchRoutes(router fiber.Router, store database.Store) {
	batchesGroup := router.Group("/batches")

	// Protected routes
	batchesGroup.Use(middleware.RequireAuth(store))

	batchService := batches.NewBatchService(store, events.GetBus())
	managementService := batches.NewManagementService(store, events.GetBus())

	batchesGroup.Get("/", func(c *fiber.Ctx) error {
		search := c.Query("search", "")
		category := c.Query("category", "")
		typeFilter := c.Query("type", "")

		list, err := batchService.ListBatches(c.Context(), search, category, typeFilter)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list batches"})
		}

		return c.JSON(fiber.Map{
			"batches": list,
			"total":   len(list),
		})
	})

	batchesGroup.Get("/:id", func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid batch ID"})
		}

		detail, err := batchService.GetBatch(c.Context(), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get batch"})
		}
		if detail == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Batch not found"})
		}

		return c.JSON(detail)
	})

	// Mutation routes
	batchesGroup.Post("/:id/access/grant", middleware.RequireRole("OWNER", "ADMIN"), func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		batchID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid batch ID"})
		}

		type AccessRequest struct {
			UserID int64 `json:"user_id"`
		}
		var req AccessRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		adminID, ok := getStringLocalSafe(c, "admin_id")
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		err = managementService.GrantAccess(c.Context(), batchID, req.UserID, adminID)
		if err != nil {
			if err.Error() == "batch not found" || err.Error() == "user not found" {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
			}
			if err.Error() == "user already has access" {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to grant access"})
		}

		return c.JSON(fiber.Map{"success": true})
	})

	batchesGroup.Post("/:id/access/revoke", middleware.RequireRole("OWNER", "ADMIN"), func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		batchID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid batch ID"})
		}

		type AccessRequest struct {
			UserID int64 `json:"user_id"`
		}
		var req AccessRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		adminID, ok := getStringLocalSafe(c, "admin_id")
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		err = managementService.RevokeAccess(c.Context(), batchID, req.UserID, adminID)
		if err != nil {
			if err.Error() == "batch not found" || err.Error() == "user not found" {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
			}
			if err.Error() == "user does not have access" {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to revoke access"})
		}

		return c.JSON(fiber.Map{"success": true})
	})
}
