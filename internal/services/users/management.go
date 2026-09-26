package users

import (
	"context"
	"fmt"

	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/events"
)

type ManagementService struct {
	store    database.Store
	eventBus *events.Bus
}

func NewManagementService(store database.Store, eventBus *events.Bus) *ManagementService {
	return &ManagementService{
		store:    store,
		eventBus: eventBus,
	}
}

func (s *ManagementService) BlockUser(ctx context.Context, userID int64, adminID string) (*UserProfile, error) {
	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	user, exists := state.Users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	// Telegram source of truth is state.BlockedUsers, modified via SetBlockedUser
	if err := s.store.SetBlockedUser(ctx, userID, true); err != nil {
		return nil, err
	}

	// Update the user struct as well for consistency
	user.IsBlocked = true
	if err := s.store.SetUser(ctx, userID, user); err != nil {
		return nil, err
	}

	if s.eventBus != nil {
		s.eventBus.Publish(events.TypeUserUpdated, events.SeverityInfo, map[string]interface{}{
			"user_id":  userID,
			"username": user.Username,
			"action":   "BLOCKED",
			"admin_id": adminID,
		})
	}

	// Use existing GetUser to return full DTO
	service := NewService(s.store)
	return service.GetUser(ctx, userID)
}

func (s *ManagementService) UnblockUser(ctx context.Context, userID int64, adminID string) (*UserProfile, error) {
	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	user, exists := state.Users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	// Telegram source of truth
	if err := s.store.SetBlockedUser(ctx, userID, false); err != nil {
		return nil, err
	}

	// Update struct
	user.IsBlocked = false
	if err := s.store.SetUser(ctx, userID, user); err != nil {
		return nil, err
	}

	if s.eventBus != nil {
		s.eventBus.Publish(events.TypeUserUpdated, events.SeverityInfo, map[string]interface{}{
			"user_id":  userID,
			"username": user.Username,
			"action":   "UNBLOCKED",
			"admin_id": adminID,
		})
	}

	service := NewService(s.store)
	return service.GetUser(ctx, userID)
}

func (s *ManagementService) ChangeTier(ctx context.Context, userID int64, newTier string, adminID string) (*UserProfile, error) {
	// Only valid tiers are empty (standard) and "vip"
	if newTier != "" && newTier != "vip" {
		return nil, fmt.Errorf("invalid tier")
	}

	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	user, exists := state.Users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	user.Tier = newTier
	if err := s.store.SetUser(ctx, userID, user); err != nil {
		return nil, err
	}

	if s.eventBus != nil {
		s.eventBus.Publish(events.TypeUserUpdated, events.SeverityInfo, map[string]interface{}{
			"user_id":  userID,
			"username": user.Username,
			"action":   "TIER_CHANGED",
			"tier":     newTier,
			"admin_id": adminID,
		})
	}

	service := NewService(s.store)
	return service.GetUser(ctx, userID)
}
