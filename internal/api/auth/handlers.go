package auth

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/events"
	"github.com/itsh4ro5/botupdate/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	store database.Store
}

func NewAuthHandler(store database.Store) *AuthHandler {
	return &AuthHandler{store: store}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	state, err := h.store.Load(context.Background())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	admin, exists := state.WebAdmins[req.Username]
	if !exists {
		// Generic message
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	err = bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	// Create session
	sessionID := uuid.New().String()
	session := &models.WebSession{
		SessionID: sessionID,
		AdminID:   admin.ID,
		Role:      admin.Role,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := h.store.SetWebSession(context.Background(), sessionID, session); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create session"})
	}

	// Set HttpOnly cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Expires:  session.ExpiresAt,
		HTTPOnly: true,
		Secure:   false, // Set to true in prod with HTTPS
		SameSite: "Lax",
	})

	events.GetBus().Publish(events.EventType("ADMIN_LOGIN"), "info", map[string]interface{}{
		"actor": admin.Username,
		"ip":    c.IP(),
	})

	return c.JSON(fiber.Map{
		"authenticated": true,
		"user": fiber.Map{
			"id":                   admin.ID,
			"username":             admin.Username,
			"role":                 admin.Role,
			"must_change_password": admin.MustChangePassword,
		},
	})
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if len(req.NewPassword) < 10 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password must be at least 10 characters long"})
	}

	sessionID := c.Cookies("session_id")
	if sessionID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Not authenticated"})
	}

	state, err := h.store.Load(context.Background())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	session, exists := state.WebSessions[sessionID]
	if !exists || time.Now().After(session.ExpiresAt) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Session expired or invalid"})
	}

	var currentAdmin *models.WebAdmin
	for _, admin := range state.WebAdmins {
		if admin.ID == session.AdminID {
			currentAdmin = admin
			break
		}
	}

	if currentAdmin == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not found"})
	}

	err = bcrypt.CompareHashAndPassword([]byte(currentAdmin.PasswordHash), []byte(req.CurrentPassword))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid current password"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash password"})
	}

	currentAdmin.PasswordHash = string(hash)
	currentAdmin.MustChangePassword = false

	if err := h.store.SetWebAdmin(context.Background(), currentAdmin.Username, currentAdmin); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save password"})
	}

	events.GetBus().Publish(events.EventType("ADMIN_PASSWORD_CHANGED"), "info", map[string]interface{}{
		"actor": currentAdmin.Username,
	})

	// Revoke all other sessions for this user
	for sid, sess := range state.WebSessions {
		if sess.AdminID == currentAdmin.ID && sid != sessionID {
			h.store.DeleteWebSession(context.Background(), sid)
		}
	}

	return c.JSON(fiber.Map{"success": true})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	sessionID := c.Cookies("session_id")
	if sessionID != "" {
		h.store.DeleteWebSession(context.Background(), sessionID)

		state, _ := h.store.Load(context.Background())
		var actor string
		if state != nil {
			if sess, exists := state.WebSessions[sessionID]; exists {
				for _, admin := range state.WebAdmins {
					if admin.ID == sess.AdminID {
						actor = admin.Username
						break
					}
				}
			}
		}

		events.GetBus().Publish(events.EventType("ADMIN_LOGOUT"), "info", map[string]interface{}{
			"actor": actor,
		})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
	})

	return c.JSON(fiber.Map{"success": true})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	sessionID := c.Cookies("session_id")
	if sessionID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Not authenticated"})
	}

	state, err := h.store.Load(context.Background())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	session, exists := state.WebSessions[sessionID]
	if !exists || time.Now().After(session.ExpiresAt) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Session expired or invalid"})
	}

	// Find the admin
	var currentAdmin *models.WebAdmin
	for _, admin := range state.WebAdmins {
		if admin.ID == session.AdminID {
			currentAdmin = admin
			break
		}
	}

	if currentAdmin == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(fiber.Map{
		"authenticated": true,
		"user": fiber.Map{
			"id":                   currentAdmin.ID,
			"username":             currentAdmin.Username,
			"role":                 currentAdmin.Role,
			"must_change_password": currentAdmin.MustChangePassword,
		},
	})
}
