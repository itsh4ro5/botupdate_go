package bot

import (
	"context"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itsh4ro5/botupdate/internal/database"
)

type MembershipService struct {
	bot       *tgbotapi.BotAPI
	auth      *AuthService
	store     database.Store
	mandatory int64
	cache     map[int64]bool
	mu        sync.RWMutex
}

func NewMembershipService(bot *tgbotapi.BotAPI, auth *AuthService, store database.Store, mandatory int64) *MembershipService {
	return &MembershipService{
		bot:       bot,
		auth:      auth,
		store:     store,
		mandatory: mandatory,
		cache:     make(map[int64]bool),
	}
}

func (s *MembershipService) CheckMembership(ctx context.Context, userID int64) (bool, error) {
	if s.auth.IsAdmin(ctx, userID) {
		return true, nil
	}
	if s.mandatory == 0 {
		return true, nil
	}

	s.mu.RLock()
	if isMember, exists := s.cache[userID]; exists {
		s.mu.RUnlock()
		return isMember, nil
	}
	s.mu.RUnlock()

	// Call API if not in cache
	member, err := s.bot.GetChatMember(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: s.mandatory,
			UserID: userID,
		},
	})
	if err != nil {
		return false, err
	}

	result := false
	switch member.Status {
	case "member", "administrator", "creator", "restricted":
		result = true
	}

	s.mu.Lock()
	s.cache[userID] = result
	s.mu.Unlock()

	return result, nil
}

// UpdateMembership updates the cached membership status for a user based on ChatMember events.
// This allows the bot to react to users leaving the channel immediately without relying on API polling.
func (s *MembershipService) UpdateMembership(userID int64, isMember bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[userID] = isMember
}
