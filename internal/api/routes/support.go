package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/itsh4ro5/botupdate/internal/api/middleware"
	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/services/support"
	"strconv"
)

func RegisterSupportRoutes(router fiber.Router, store database.Store, supportService *support.SupportService) {
	group := router.Group("/support")

	// Protected routes
	group.Use(middleware.RequireAuth(store))

	group.Get("/conversations", func(c *fiber.Ctx) error {
		search := c.Query("search", "")

		list, err := supportService.ListConversations(c.Context(), search)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list conversations"})
		}

		return c.JSON(list)
	})

	group.Get("/conversations/:id", func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		userID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
		}

		detail, err := supportService.GetConversation(c.Context(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get conversation"})
		}
		if detail == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Conversation not found"})
		}

		return c.JSON(detail)
	})

	group.Get("/conversations/:id/messages", func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		userID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
		}

		msgs, err := supportService.GetMessages(c.Context(), userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get messages"})
		}

		return c.JSON(msgs)
	})

	group.Post("/conversations/:id/reply", middleware.RequireRole("OWNER", "ADMIN"), func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		userID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
		}

		var req struct {
			Text string `json:"text"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		adminID, ok := getStringLocalSafe(c, "admin_id")
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		msg, err := supportService.Reply(c.Context(), userID, req.Text, adminID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"success": true, "message_id": msg.ID})
	})

	group.Post("/messages/:id/delete", middleware.RequireRole("OWNER", "ADMIN"), func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		msgID, err := strconv.Atoi(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid message ID"})
		}

		err = supportService.DeleteMessage(c.Context(), msgID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		return c.JSON(fiber.Map{"success": true})
	})
}
