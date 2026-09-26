package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/itsh4ro5/botupdate/internal/models"
)

// Store defines the interface for database operations
type Store interface {
	Load(ctx context.Context) (*models.BotState, error)
	Save(ctx context.Context, state *models.BotState) error

	// Targeted updates to prevent concurrent map overwrites
	SetUser(ctx context.Context, userID int64, user *models.User) error
	SetBlockedUser(ctx context.Context, userID int64, blocked bool) error
	SetAdmin(ctx context.Context, userID int64, isAdmin bool) error
	SetSupportTopic(ctx context.Context, userID int64, topic *models.SupportTopic) error
	SetPendingRequest(ctx context.Context, requestID string, req *models.PendingRequest) error
	SetInviteLink(ctx context.Context, hash string, link *models.InviteMapping) error
	SetBatchCategory(ctx context.Context, batchID int64, category string) error
	SetCategories(ctx context.Context, categories []string) error
	SetLockState(ctx context.Context, lockType string, locked bool) error
	SetMaintenanceMode(ctx context.Context, enabled bool) error
	AddScheduledDelete(ctx context.Context, sd *models.ScheduledDelete) error
	RemoveScheduledDelete(ctx context.Context, chatID int64, msgID int) error
	SetCustomWelcome(ctx context.Context, batchID int64, text string) error
	SetUserbotSession(ctx context.Context, session string) error
	SetBatch(ctx context.Context, id int64, batch *models.Batch, batchType string) error
	SetBatchCoin(ctx context.Context, id int64, coins int64) error
	RemoveBatch(ctx context.Context, id int64) error

	SetWebAdmin(ctx context.Context, username string, admin *models.WebAdmin) error
	SetWebSession(ctx context.Context, sessionID string, session *models.WebSession) error
	DeleteWebSession(ctx context.Context, sessionID string) error

	SetMessageMapping(ctx context.Context, key string, val string) error
	RemoveMessageMapping(ctx context.Context, key string) error

	// Dashboard Analytics (Read-only)
	GetDashboardOverview(ctx context.Context) (*models.DashboardOverview, error)
}

// JSONStore provides a local fallback persistence using a JSON file
type JSONStore struct {
	mu       sync.RWMutex
	filePath string
}

func NewJSONStore(filePath string) *JSONStore {
	return &JSONStore{
		filePath: filePath,
	}
}

func (s *JSONStore) Load(ctx context.Context) (*models.BotState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return s.emptyState(), nil
		}
		return nil, fmt.Errorf("failed to read data file: %w", err)
	}

	var state models.BotState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Initialize maps if they are nil
	if state.AdminIDs == nil {
		state.AdminIDs = make(map[int64]struct{})
	}
	if state.Users == nil {
		state.Users = make(map[int64]*models.User)
	}
	// ... (other map initializations)

	return &state, nil
}

func (s *JSONStore) Save(ctx context.Context, state *models.BotState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode state: %w", err)
	}

	// Write to temp file then rename for atomic save
	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, s.filePath)
}

func (s *JSONStore) emptyState() *models.BotState {
	return &models.BotState{
		AdminIDs:        make(map[int64]struct{}),
		FreeBatches:     make(map[int64]*models.Batch),
		PaidBatches:     make(map[int64]*models.Batch),
		SpecialBatches:  make(map[int64]*models.Batch),
		AllChats:        make(map[int64]string),
		Users:           make(map[int64]*models.User),
		BlockedUsers:    make(map[int64]struct{}),
		UserTopics:      make(map[int64]*models.SupportTopic),
		PendingRequests: make(map[string]*models.PendingRequest),
		LinkMap:         make(map[string]*models.InviteMapping),
		CustomWelcomes:  make(map[int64]string),
		BatchCategories: make(map[int64]string),
		Categories:      make([]string, 0),
		BatchCoins:      make(map[int64]int64),
		WebAdmins:       make(map[string]*models.WebAdmin),
		WebSessions:     make(map[string]*models.WebSession),
		MessageMap:      make(map[string]string),
	}
}

func (s *JSONStore) SetWebAdmin(ctx context.Context, username string, admin *models.WebAdmin) error {
	state, err := s.Load(ctx)
	if err != nil {
		return err
	}
	state.WebAdmins[username] = admin
	return s.Save(ctx, state)
}

func (s *JSONStore) SetWebSession(ctx context.Context, sessionID string, session *models.WebSession) error {
	state, err := s.Load(ctx)
	if err != nil {
		return err
	}
	state.WebSessions[sessionID] = session
	return s.Save(ctx, state)
}

func (s *JSONStore) SetMessageMapping(ctx context.Context, key string, val string) error {
	state, err := s.Load(ctx)
	if err != nil {
		return err
	}
	if state.MessageMap == nil {
		state.MessageMap = make(map[string]string)
	}
	state.MessageMap[key] = val
	return s.Save(ctx, state)
}

func (s *JSONStore) RemoveMessageMapping(ctx context.Context, key string) error {
	state, err := s.Load(ctx)
	if err != nil {
		return err
	}
	if state.MessageMap != nil {
		delete(state.MessageMap, key)
		return s.Save(ctx, state)
	}
	return nil
}

func (s *JSONStore) DeleteWebSession(ctx context.Context, sessionID string) error {
	state, err := s.Load(ctx)
	if err != nil {
		return err
	}
	delete(state.WebSessions, sessionID)
	return s.Save(ctx, state)
}

func (s *JSONStore) GetDashboardOverview(ctx context.Context) (*models.DashboardOverview, error) {
	overview := &models.DashboardOverview{
		GeneratedAt: time.Now().UTC(),
	}

	state, err := s.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load state for dashboard: %w", err)
	}

	overview.Users.Total = len(state.Users)
	overview.Users.Active = -1 // Activity timestamp not uniformly tracked

	overview.Batches.Total = len(state.FreeBatches) + len(state.PaidBatches) + len(state.SpecialBatches)
	overview.Requests.Pending = len(state.PendingRequests)
	overview.System.Database = "online"

	return overview, nil
}
