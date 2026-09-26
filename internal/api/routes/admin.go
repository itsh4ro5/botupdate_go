package routes

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/itsh4ro5/botupdate/internal/api/middleware"
	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/services/admin"
	"github.com/itsh4ro5/botupdate/internal/services/audit"
)

type createAdminReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type changeRoleReq struct {
	Role string `json:"role"`
}

func RegisterAdminRoutes(router fiber.Router, store database.Store, adminSvc *admin.AdminService, auditSvc *audit.AuditService) {
	group := router.Group("/admins")

	group.Use(middleware.RequireAuth(store))

	// Get all admins - OWNER only
	group.Get("/", middleware.RequireRole("OWNER"), func(c *fiber.Ctx) error {
		res, err := adminSvc.ListAdmins(c.Context())
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(res)
	})

	// Create admin - OWNER only
	group.Post("/", middleware.RequireRole("OWNER"), func(c *fiber.Ctx) error {
		actor, ok := getStringLocalSafe(c, "username")
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
		}
		var req createAdminReq
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
		}

		if err := adminSvc.CreateAdmin(c.Context(), actor, req.Username, req.Password, req.Role); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		auditSvc.Log(actor, "OWNER", "ADMIN_CREATED", "ADMIN", req.Username, true, map[string]interface{}{"role": req.Role})
		return c.JSON(fiber.Map{"success": true})
	})

	// Change role - OWNER only
	group.Put("/:username/role", middleware.RequireRole("OWNER"), func(c *fiber.Ctx) error {
		actor, ok := getStringLocalSafe(c, "username")
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
		}
		target := c.Params("username")
		var req changeRoleReq
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
		}

		if err := adminSvc.ChangeAdminRole(c.Context(), actor, target, req.Role); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		auditSvc.Log(actor, "OWNER", "ADMIN_ROLE_CHANGED", "ADMIN", target, true, map[string]interface{}{"role": req.Role})
		return c.JSON(fiber.Map{"success": true})
	})

	// List Sessions - OWNER or Self
	group.Get("/:username/sessions", func(c *fiber.Ctx) error {
		actor, ok := getStringLocalSafe(c, "username")
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
		}
		role, ok := getStringLocalSafe(c, "role")
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
		}
		target := c.Params("username")

		if role != "OWNER" && actor != target {
			return c.Status(403).JSON(fiber.Map{"error": "Forbidden"})
		}

		sessionID, _ := getStringLocalSafe(c, "session_id")
		res, err := adminSvc.ListSessions(c.Context(), target, sessionID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(res)
	})

	// Revoke Session
	group.Post("/:username/sessions/:id/revoke", func(c *fiber.Ctx) error {
		actor, ok := getStringLocalSafe(c, "username")
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
		}
		role, ok := getStringLocalSafe(c, "role")
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
		}
		target := c.Params("username")

		if role != "OWNER" && actor != target {
			return c.Status(403).JSON(fiber.Map{"error": "Forbidden"})
		}

		id := c.Params("id")
		if err := adminSvc.RevokeSession(c.Context(), actor, id); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		auditSvc.Log(actor, role, "SESSION_REVOKED", "SESSION", id, true, nil)
		return c.JSON(fiber.Map{"success": true})
	})

	// Revoke All Sessions
	group.Post("/:username/sessions/revoke-all", func(c *fiber.Ctx) error {
		actor, ok := getStringLocalSafe(c, "username")
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
		}
		role, ok := getStringLocalSafe(c, "role")
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
		}
		target := c.Params("username")

		if role != "OWNER" && actor != target {
			return c.Status(403).JSON(fiber.Map{"error": "Forbidden"})
		}

		if err := adminSvc.RevokeAllSessions(c.Context(), actor, target); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		auditSvc.Log(actor, role, "SESSION_REVOKED_ALL", "USER", target, true, nil)
		return c.JSON(fiber.Map{"success": true})
	})

	// Force Password Change
	group.Post("/:username/force-password-change", middleware.RequireRole("OWNER"), func(c *fiber.Ctx) error {
		actor, ok := getStringLocalSafe(c, "username")
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
		}
		role, ok := getStringLocalSafe(c, "role")
		if !ok {
			return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
		}
		target := c.Params("username")

		if err := adminSvc.ForcePasswordChange(c.Context(), actor, target); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		auditSvc.Log(actor, role, "ADMIN_FORCE_PASSWORD_CHANGE", "USER", target, true, nil)
		return c.JSON(fiber.Map{"success": true})
	})

	// Audit Logs - OWNER/ADMIN
	auditGroup := router.Group("/audit")
	auditGroup.Use(middleware.RequireAuth(store))
	auditGroup.Use(middleware.RequireRole("OWNER", "ADMIN"))

	auditGroup.Get("/", func(c *fiber.Ctx) error {
		page, _ := strconv.Atoi(c.Query("page", "1"))
		pageSize, _ := strconv.Atoi(c.Query("pageSize", "25"))
		if pageSize > 100 {
			pageSize = 100
		}
		search := c.Query("search", "")
		action := c.Query("action", "")

		res := auditSvc.GetLogs(page, pageSize, search, action)
		return c.JSON(res)
	})
}
