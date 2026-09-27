package bot

import (
	"context"
	"sync"

	"github.com/itsh4ro5/botupdate/internal/database"
)

// AuthService handles access control checks
type AuthService struct {
	Store   database.Store
	ownerID int64
	admins  map[int64]struct{}
	blocked map[int64]struct{}
	mu      sync.RWMutex
}

func NewAuthService(store database.Store, ownerID int64) *AuthService {
	a := &AuthService{
		Store:   store,
		ownerID: ownerID,
		admins:  make(map[int64]struct{}),
		blocked: make(map[int64]struct{}),
	}
	
	state, err := store.Load(context.Background())
	if err == nil {
		for id := range state.AdminIDs {
			a.admins[id] = struct{}{}
		}
		for id := range state.BlockedUsers {
			a.blocked[id] = struct{}{}
		}
	}
	
	return a
}

// IsOwner checks if the given userID is the owner
func (s *AuthService) IsOwner(ctx context.Context, userID int64) bool {
	return userID == s.ownerID
}

// IsAdmin checks if the given userID is an admin or the owner
func (s *AuthService) IsAdmin(ctx context.Context, userID int64) bool {
	if s.IsOwner(ctx, userID) {
		return true
	}
	s.mu.RLock()
	_, ok := s.admins[userID]
	s.mu.RUnlock()
	return ok
}

// IsBlocked checks if the user is blocked
func (s *AuthService) IsBlocked(ctx context.Context, userID int64) bool {
	s.mu.RLock()
	_, ok := s.blocked[userID]
	s.mu.RUnlock()
	return ok
}

// UpdateAdminStatus updates the local admin cache
func (s *AuthService) UpdateAdminStatus(userID int64, isAdmin bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if isAdmin {
		s.admins[userID] = struct{}{}
	} else {
		delete(s.admins, userID)
	}
}

// UpdateBlockedStatus updates the local blocked cache
func (s *AuthService) UpdateBlockedStatus(userID int64, isBlocked bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if isBlocked {
		s.blocked[userID] = struct{}{}
	} else {
		delete(s.blocked, userID)
	}
}
