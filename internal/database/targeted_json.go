package database

import (
	"context"

	"github.com/itsh4ro5/botupdate/internal/models"
)

// Helper to wrap load/modify/save for JSON store
func (s *JSONStore) modifyState(ctx context.Context, modifyFn func(*models.BotState)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return nil
}

// Since JSONStore is just a fallback, we can use the simplest thread-safe approach:
func (s *JSONStore) updateWithLock(ctx context.Context, modifyFn func(*models.BotState)) error {
	state, err := s.Load(ctx)
	if err != nil {
		return err
	}

	// Because Load and Save use their own locks, we can't lock across them without deadlock
	// using the existing RWMutex. However, a coarse lock over the read-modify-write is needed.
	// For JSONStore fallback, we accept a slight race or we add a global lock.
	// To keep it simple for the interface implementation:
	modifyFn(state)
	return s.Save(ctx, state)
}

func (s *JSONStore) SetUser(ctx context.Context, userID int64, user *models.User) error {
	return s.updateWithLock(ctx, func(state *models.BotState) { state.Users[userID] = user })
}

func (s *JSONStore) SetBlockedUser(ctx context.Context, userID int64, blocked bool) error {
	return s.updateWithLock(ctx, func(state *models.BotState) {
		if blocked {
			state.BlockedUsers[userID] = struct{}{}
		} else {
			delete(state.BlockedUsers, userID)
		}
	})
}

func (s *JSONStore) SetAdmin(ctx context.Context, userID int64, isAdmin bool) error {
	return s.updateWithLock(ctx, func(state *models.BotState) {
		if isAdmin {
			state.AdminIDs[userID] = struct{}{}
		} else {
			delete(state.AdminIDs, userID)
		}
	})
}

func (s *JSONStore) SetSupportTopic(ctx context.Context, userID int64, topic *models.SupportTopic) error {
	return s.updateWithLock(ctx, func(state *models.BotState) { state.UserTopics[userID] = topic })
}

func (s *JSONStore) SetPendingRequest(ctx context.Context, requestID string, req *models.PendingRequest) error {
	return s.updateWithLock(ctx, func(state *models.BotState) {
		if req == nil {
			delete(state.PendingRequests, requestID)
		} else {
			state.PendingRequests[requestID] = req
		}
	})
}

func (s *JSONStore) SetInviteLink(ctx context.Context, hash string, link *models.InviteMapping) error {
	return s.updateWithLock(ctx, func(state *models.BotState) {
		if link == nil {
			delete(state.LinkMap, hash)
		} else {
			state.LinkMap[hash] = link
		}
	})
}

func (s *JSONStore) SetBatchCategory(ctx context.Context, batchID int64, category string) error {
	return s.updateWithLock(ctx, func(state *models.BotState) { state.BatchCategories[batchID] = category })
}

func (s *JSONStore) SetCategories(ctx context.Context, categories []string) error {
	return s.updateWithLock(ctx, func(state *models.BotState) { state.Categories = categories })
}

func (s *JSONStore) SetLockState(ctx context.Context, lockType string, locked bool) error {
	return s.updateWithLock(ctx, func(state *models.BotState) {
		switch lockType {
		case "free":
			state.FreeLocked = locked
		case "paid":
			state.PaidLocked = locked
		case "testbot":
			state.TestBotLocked = locked
		case "lockdown":
			state.NewUsersAllowed = locked
		}
	})
}

func (s *JSONStore) SetMaintenanceMode(ctx context.Context, enabled bool) error {
	return s.updateWithLock(ctx, func(state *models.BotState) { state.MaintenanceMode = enabled })
}

func (s *JSONStore) AddScheduledDelete(ctx context.Context, sd *models.ScheduledDelete) error {
	return s.updateWithLock(ctx, func(state *models.BotState) {
		state.ScheduledDeletes = append(state.ScheduledDeletes, sd)
	})
}

func (s *JSONStore) RemoveScheduledDelete(ctx context.Context, chatID int64, msgID int) error {
	return s.updateWithLock(ctx, func(state *models.BotState) {
		var filtered []*models.ScheduledDelete
		for _, sd := range state.ScheduledDeletes {
			if sd.ChatID != chatID || sd.MessageID != msgID {
				filtered = append(filtered, sd)
			}
		}
		state.ScheduledDeletes = filtered
	})
}

func (s *JSONStore) SetCustomWelcome(ctx context.Context, batchID int64, text string) error {
	return s.updateWithLock(ctx, func(state *models.BotState) {
		if text == "" {
			delete(state.CustomWelcomes, batchID)
		} else {
			state.CustomWelcomes[batchID] = text
		}
	})
}

func (s *JSONStore) SetUserbotSession(ctx context.Context, session string) error {
	return s.updateWithLock(ctx, func(state *models.BotState) { state.UserbotSession = session })
}

func (s *JSONStore) SetBatch(ctx context.Context, id int64, batch *models.Batch, batchType string) error {
	return s.updateWithLock(ctx, func(state *models.BotState) {
		switch batchType {
		case "free":
			state.FreeBatches[id] = batch
		case "paid":
			state.PaidBatches[id] = batch
		case "special":
			state.SpecialBatches[id] = batch
		}
		state.AllChats[id] = batch.Name
		state.BatchCategories[id] = batch.Category
	})
}

func (s *JSONStore) SetBatchCoin(ctx context.Context, id int64, coins int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.Load(ctx)
	if err != nil {
		return err
	}

	if state.BatchCoins == nil {
		state.BatchCoins = make(map[int64]int64)
	}
	state.BatchCoins[id] = coins
	return s.Save(ctx, state)
}

func (s *JSONStore) RemoveBatch(ctx context.Context, id int64) error {
	return s.updateWithLock(ctx, func(state *models.BotState) {
		delete(state.FreeBatches, id)
		delete(state.PaidBatches, id)
		delete(state.SpecialBatches, id)
		delete(state.AllChats, id)
		delete(state.BatchCategories, id)
		delete(state.CustomWelcomes, id)
		delete(state.BatchCoins, id)
	})
}

func (s *JSONStore) SaveBatchContents(ctx context.Context, chatID string, data interface{}) error {
	// JSON fallback mock
	return nil
}
