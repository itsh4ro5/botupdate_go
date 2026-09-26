package routes

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/itsh4ro5/botupdate/internal/api/middleware"
	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/events"
	"github.com/itsh4ro5/botupdate/internal/services/users"
)

func RegisterUserRoutes(api fiber.Router, store database.Store, userService *users.Service) {
	usersGroup := api.Group("/users")

	// Protected routes
	usersGroup.Use(middleware.RequireAuth(store), middleware.RequireRole("OWNER", "ADMIN"))

	usersGroup.Get("/", func(c *fiber.Ctx) error {
		page, _ := strconv.Atoi(c.Query("page", "1"))
		if page < 1 {
			page = 1
		}

		pageSize, _ := strconv.Atoi(c.Query("pageSize", "25"))
		if pageSize < 1 || pageSize > 100 {
			pageSize = 25
		}

		search := c.Query("search", "")
		sortField := c.Query("sort", "joined_at_desc")

		var isBlocked *bool
		statusFilter := strings.ToLower(c.Query("status", ""))
		if statusFilter == "blocked" {
			val := true
			isBlocked = &val
		} else if statusFilter == "active" {
			val := false
			isBlocked = &val
		}

		res, err := userService.ListUsers(c.Context(), page, pageSize, search, sortField, isBlocked)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch users",
			})
		}

		return c.JSON(res)
	})

	usersGroup.Get("/:id", func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
		}

		user, err := userService.GetUser(c.Context(), id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch user",
			})
		}

		if user == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
		}

		return c.JSON(user)
	})

	managementService := users.NewManagementService(store, events.GetBus())

	usersGroup.Post("/:id/block", func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
		}
		adminID, ok := getStringLocalSafe(c, "admin_id")
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		user, err := managementService.BlockUser(c.Context(), id, adminID)
		if err != nil {
			if err.Error() == "user not found" {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to block user"})
		}
		return c.JSON(user)
	})

	usersGroup.Post("/:id/unblock", func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
		}
		adminID, ok := getStringLocalSafe(c, "admin_id")
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		user, err := managementService.UnblockUser(c.Context(), id, adminID)
		if err != nil {
			if err.Error() == "user not found" {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to unblock user"})
		}
		return c.JSON(user)
	})

	usersGroup.Post("/:id/tier", func(c *fiber.Ctx) error {
		idStr := c.Params("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
		}

		type TierRequest struct {
			Tier string `json:"tier"`
		}
		var req TierRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		adminID, ok := getStringLocalSafe(c, "admin_id")
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		user, err := managementService.ChangeTier(c.Context(), id, req.Tier, adminID)
		if err != nil {
			if err.Error() == "invalid tier" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid tier"})
			}
			if err.Error() == "user not found" {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to change tier"})
		}
		return c.JSON(user)
	})
}
