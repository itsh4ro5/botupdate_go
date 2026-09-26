package mtproto

import (
	"context"
	"sync"
	"time"
)

type LoginState struct {
	OwnerID   int64
	Phone     string
	PhoneHash string
	ExpiresAt time.Time
}

type AuthManager struct {
	mu         sync.RWMutex
	loginState *LoginState
}

func NewAuthManager() *AuthManager {
	return &AuthManager{}
}

func (m *AuthManager) StartLogin(ownerID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loginState = &LoginState{
		OwnerID:   ownerID,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	return nil
}

func (m *AuthManager) CancelLogin() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loginState = nil
}

// SessionStorage implements gotd session.Storage interface
type SessionStorage struct {
	// ... hook into DB or BotState
}

func (s *SessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	return nil, nil // mock
}

func (s *SessionStorage) StoreSession(ctx context.Context, data []byte) error {
	return nil // mock
}
