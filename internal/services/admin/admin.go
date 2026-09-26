package admin

import (
	"context"
	"errors"
	"time"

	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/events"
	"github.com/itsh4ro5/botupdate/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type AdminService struct {
	store    database.Store
	eventBus *events.Bus
}

func NewAdminService(store database.Store, eventBus *events.Bus) *AdminService {
	return &AdminService{
		store:    store,
		eventBus: eventBus,
	}
}

type AdminResponse struct {
	Username           string `json:"username"`
	Role               string `json:"role"`
	CreatedAt          string `json:"created_at"`
	MustChangePassword bool   `json:"must_change_password"`
	ActiveSessions     int    `json:"active_sessions"`
}

type SessionResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	IsCurrent bool   `json:"is_current"`
}

func (s *AdminService) ListAdmins(ctx context.Context) ([]AdminResponse, error) {
	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	sessionCounts := make(map[string]int)
	for _, sess := range state.WebSessions {
		if sess.ExpiresAt.After(time.Now()) {
			sessionCounts[sess.AdminID]++
		}
	}

	var res []AdminResponse
	for _, a := range state.WebAdmins {
		res = append(res, AdminResponse{
			Username:           a.Username,
			Role:               a.Role,
			CreatedAt:          a.CreatedAt.Format(time.RFC3339),
			MustChangePassword: a.MustChangePassword,
			ActiveSessions:     sessionCounts[a.ID],
		})
	}
	return res, nil
}

func (s *AdminService) CreateAdmin(ctx context.Context, creator string, username, password, role string) error {
	if len(password) < 10 {
		return errors.New("password must be at least 10 characters")
	}

	if role != "ADMIN" && role != "SUPPORT" {
		return errors.New("invalid role assignment")
	}

	state, err := s.store.Load(ctx)
	if err != nil {
		return err
	}

	if _, exists := state.WebAdmins[username]; exists {
		return errors.New("admin username already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin := &models.WebAdmin{
		ID:                 username, // Using username as ID to simplify
		Username:           username,
		PasswordHash:       string(hash),
		Role:               role,
		CreatedAt:          time.Now(),
		MustChangePassword: true,
	}

	err = s.store.SetWebAdmin(ctx, username, admin)
	if err == nil {
		s.eventBus.Publish("ADMIN_CREATED", "info", map[string]interface{}{
			"actor":  creator,
			"target": username,
			"role":   role,
		})
	}
	return err
}

func (s *AdminService) ChangeAdminRole(ctx context.Context, actor, targetUsername, newRole string) error {
	if newRole != "ADMIN" && newRole != "SUPPORT" && newRole != "OWNER" {
		return errors.New("invalid role")
	}

	state, err := s.store.Load(ctx)
	if err != nil {
		return err
	}

	target, exists := state.WebAdmins[targetUsername]
	if !exists {
		return errors.New("admin not found")
	}

	if target.Role == "OWNER" && newRole != "OWNER" {
		// Prevent demoting the last OWNER
		ownerCount := 0
		for _, a := range state.WebAdmins {
			if a.Role == "OWNER" {
				ownerCount++
			}
		}
		if ownerCount <= 1 {
			return errors.New("cannot demote the last OWNER")
		}
	}

	target.Role = newRole
	err = s.store.SetWebAdmin(ctx, targetUsername, target)
	if err == nil {
		s.eventBus.Publish("ADMIN_ROLE_CHANGED", "info", map[string]interface{}{
			"actor":    actor,
			"target":   targetUsername,
			"new_role": newRole,
		})
	}
	return err
}

func (s *AdminService) ListSessions(ctx context.Context, targetUsername, currentSessionID string) ([]SessionResponse, error) {
	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	// Find the ID of the target
	target, exists := state.WebAdmins[targetUsername]
	if !exists {
		return nil, errors.New("admin not found")
	}

	var res []SessionResponse
	for id, sess := range state.WebSessions {
		if sess.AdminID == target.ID && sess.ExpiresAt.After(time.Now()) {
			res = append(res, SessionResponse{
				ID:        id,        // Passing ID back allows revocation targeting
				CreatedAt: "Unknown", // WebSession doesn't store this
				ExpiresAt: sess.ExpiresAt.Format(time.RFC3339),
				IPAddress: "Unknown",
				UserAgent: "Unknown",
				IsCurrent: id == currentSessionID,
			})
		}
	}
	return res, nil
}

func (s *AdminService) RevokeSession(ctx context.Context, actor, sessionID string) error {
	err := s.store.DeleteWebSession(ctx, sessionID)
	if err == nil {
		s.eventBus.Publish("SESSION_REVOKED", "info", map[string]interface{}{
			"actor":      actor,
			"session_id": sessionID,
		})
	}
	return err
}

func (s *AdminService) RevokeAllSessions(ctx context.Context, actor, targetUsername string) error {
	state, err := s.store.Load(ctx)
	if err != nil {
		return err
	}

	target, exists := state.WebAdmins[targetUsername]
	if !exists {
		return errors.New("admin not found")
	}

	for id, sess := range state.WebSessions {
		if sess.AdminID == target.ID {
			_ = s.store.DeleteWebSession(ctx, id)
		}
	}

	s.eventBus.Publish("SESSION_REVOKED_ALL", "info", map[string]interface{}{
		"actor":           actor,
		"target_username": targetUsername,
		"action":          "REVOKE_ALL",
	})

	return nil
}

func (s *AdminService) ForcePasswordChange(ctx context.Context, actor, targetUsername string) error {
	state, err := s.store.Load(ctx)
	if err != nil {
		return err
	}

	target, exists := state.WebAdmins[targetUsername]
	if !exists {
		return errors.New("admin not found")
	}

	target.MustChangePassword = true
	err = s.store.SetWebAdmin(ctx, targetUsername, target)
	if err == nil {
		// Optional: revoke their sessions so they are forced to log in again immediately
		_ = s.RevokeAllSessions(ctx, actor, targetUsername)

		s.eventBus.Publish("ADMIN_FORCE_PASSWORD_CHANGE", "info", map[string]interface{}{
			"actor":  actor,
			"target": targetUsername,
		})
	}
	return err
}
