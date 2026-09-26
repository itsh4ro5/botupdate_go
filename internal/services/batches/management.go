package batches

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/events"
	"github.com/itsh4ro5/botupdate/internal/models"
)

// ManagementService handles mutating access to batches
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

func (s *ManagementService) GrantAccess(ctx context.Context, batchID int64, userID int64, adminID string) error {
	state, err := s.store.Load(ctx)
	if err != nil {
		return err
	}

	bType := getBatchType(state, batchID)
	if bType == "" {
		return errors.New("batch not found")
	}

	user, ok := state.Users[userID]
	if !ok {
		return errors.New("user not found")
	}

	hasAccess := checkAccess(user, batchID, bType)
	if hasAccess {
		return errors.New("user already has access")
	}

	// Modify access
	switch bType {
	case "free":
		user.FreeBatchesJoined = append(user.FreeBatchesJoined, batchID)
	case "paid":
		user.JoinedBatches = append(user.JoinedBatches, batchID)
	case "special":
		idStr := strconv.FormatInt(batchID, 10)
		user.UnlockedBatches = append(user.UnlockedBatches, idStr)
	}

	// Targeted save of the single user
	err = s.store.SetUser(ctx, userID, user)
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	if s.eventBus != nil {
		s.eventBus.Publish(events.TypeAccessGranted, events.SeverityInfo, map[string]interface{}{
			"user_id":  userID,
			"username": user.Username,
			"batch_id": batchID,
			"admin_id": adminID,
			"action":   "ACCESS_GRANTED",
		})
	}

	return nil
}

func (s *ManagementService) RevokeAccess(ctx context.Context, batchID int64, userID int64, adminID string) error {
	state, err := s.store.Load(ctx)
	if err != nil {
		return err
	}

	bType := getBatchType(state, batchID)
	if bType == "" {
		return errors.New("batch not found")
	}

	user, ok := state.Users[userID]
	if !ok {
		return errors.New("user not found")
	}

	hasAccess := checkAccess(user, batchID, bType)
	if !hasAccess {
		return errors.New("user does not have access")
	}

	// Remove access
	switch bType {
	case "free":
		user.FreeBatchesJoined = removeInt64(user.FreeBatchesJoined, batchID)
	case "paid":
		user.JoinedBatches = removeInt64(user.JoinedBatches, batchID)
	case "special":
		idStr := strconv.FormatInt(batchID, 10)
		user.UnlockedBatches = removeString(user.UnlockedBatches, idStr)
	}

	// Targeted save of the single user
	err = s.store.SetUser(ctx, userID, user)
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	if s.eventBus != nil {
		s.eventBus.Publish(events.TypeAccessRevoked, events.SeverityWarning, map[string]interface{}{
			"user_id":  userID,
			"username": user.Username,
			"batch_id": batchID,
			"admin_id": adminID,
			"action":   "ACCESS_REVOKED",
		})
	}

	return nil
}

func getBatchType(state *models.BotState, batchID int64) string {
	if _, ok := state.FreeBatches[batchID]; ok {
		return "free"
	}
	if _, ok := state.PaidBatches[batchID]; ok {
		return "paid"
	}
	if _, ok := state.SpecialBatches[batchID]; ok {
		return "special"
	}
	return ""
}

func checkAccess(user *models.User, batchID int64, bType string) bool {
	if bType == "free" {
		for _, b := range user.FreeBatchesJoined {
			if b == batchID {
				return true
			}
		}
	} else if bType == "paid" {
		for _, b := range user.JoinedBatches {
			if b == batchID {
				return true
			}
		}
	} else if bType == "special" {
		idStr := strconv.FormatInt(batchID, 10)
		for _, b := range user.UnlockedBatches {
			if b == idStr {
				return true
			}
		}
	}
	return false
}

func removeInt64(slice []int64, val int64) []int64 {
	var out []int64
	for _, v := range slice {
		if v != val {
			out = append(out, v)
		}
	}
	return out
}

func removeString(slice []string, val string) []string {
	var out []string
	for _, v := range slice {
		if v != val {
			out = append(out, v)
		}
	}
	return out
}
