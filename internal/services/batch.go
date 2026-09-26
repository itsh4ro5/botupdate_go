package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/models"
	"github.com/itsh4ro5/botupdate/internal/telegram"
)

// BatchService manages free, paid, and special batches
type BatchService struct {
	store          database.Store
	bot            *tgbotapi.BotAPI
	api            *telegram.APIClient
	SupportService *SupportService
	mu             sync.RWMutex
}

func NewBatchService(store database.Store, bot *tgbotapi.BotAPI, api *telegram.APIClient, supportService *SupportService) *BatchService {
	return &BatchService{
		store:          store,
		bot:            bot,
		api:            api,
		SupportService: supportService,
	}
}

// AddBatch creates a new batch in the specified category
func (s *BatchService) AddBatch(ctx context.Context, batchType string, id int64, name, category string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load(ctx)
	if err != nil {
		return err
	}

	batch := &models.Batch{
		ID:       id,
		Name:     name,
		Category: category,
		Type:     batchType,
	}

	switch batchType {
	case "free":
		state.FreeBatches[id] = batch
	case "paid":
		state.PaidBatches[id] = batch
	case "special":
		state.SpecialBatches[id] = batch
	default:
		return fmt.Errorf("invalid batch type: %s", batchType)
	}

	state.AllChats[id] = name
	state.BatchCategories[id] = category

	// Ensure category exists in global list
	found := false
	for _, c := range state.Categories {
		if c == category {
			found = true
			break
		}
	}
	if !found {
		state.Categories = append(state.Categories, category)
		s.store.SetCategories(ctx, state.Categories)
	}

	return s.store.SetBatch(ctx, id, batch, batchType)
}

// RemoveBatch deletes a batch from the system
func (s *BatchService) RemoveBatch(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.store.RemoveBatch(ctx, id)
}

// SetDemoAccess grants a user 3-hour demo access to all paid batches
func (s *BatchService) SetDemoAccess(ctx context.Context, userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.Load(ctx)
	if err != nil {
		return err
	}

	user, ok := state.Users[userID]
	if !ok {
		return fmt.Errorf("user %d not found", userID)
	}

	if user.Demos == nil {
		user.Demos = make(map[string]interface{})
	}

	expiry := time.Now().Add(3 * time.Hour).Unix()
	for id := range state.PaidBatches {
		user.Demos[fmt.Sprintf("%d", id)] = models.SetDemo(expiry)
	}

	return s.store.SetUser(ctx, userID, user)
}

// IsDemoActive checks if the user's trial period is still active for a specific batch
func (s *BatchService) IsDemoActive(ctx context.Context, userID int64, batchID int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, err := s.store.Load(ctx)
	if err != nil {
		return false
	}

	user, ok := state.Users[userID]
	if !ok || user.Demos == nil {
		return false
	}

	val, exists := user.Demos[fmt.Sprintf("%d", batchID)]
	if !exists {
		return false
	}
	exp := models.GetDemoExpiry(val)

	return time.Now().Unix() < exp
}
