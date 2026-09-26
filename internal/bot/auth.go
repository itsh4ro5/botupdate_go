package bot

import (
	"context"

	"github.com/itsh4ro5/botupdate/internal/database"
)

// AuthService handles access control checks
type AuthService struct {
	Store   database.Store
	ownerID int64
}

func NewAuthService(store database.Store, ownerID int64) *AuthService {
	return &AuthService{
		Store:   store,
		ownerID: ownerID,
	}
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
	state, err := s.Store.Load(ctx)
	if err != nil {
		return false
	}
	_, ok := state.AdminIDs[userID]
	return ok
}

// IsBlocked checks if the user is blocked
func (s *AuthService) IsBlocked(ctx context.Context, userID int64) bool {
	state, err := s.Store.Load(ctx)
	if err != nil {
		return false
	}
	_, ok := state.BlockedUsers[userID]
	return ok
}
