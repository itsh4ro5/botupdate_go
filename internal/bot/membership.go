package bot

import (
	"context"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itsh4ro5/botupdate/internal/database"
)

type MembershipService struct {
	bot       *tgbotapi.BotAPI
	auth      *AuthService
	store     database.Store
	mandatory int64
	cache     map[string]int64
	cacheTTL  time.Duration
}

func NewMembershipService(bot *tgbotapi.BotAPI, auth *AuthService, store database.Store, mandatory int64) *MembershipService {
	return &MembershipService{
		bot:       bot,
		auth:      auth,
		store:     store,
		mandatory: mandatory,
		cache:     make(map[string]int64),
		cacheTTL:  30 * time.Second,
	}
}

func (s *MembershipService) CheckMembership(ctx context.Context, userID int64) (bool, error) {
	if s.auth.IsAdmin(ctx, userID) {
		return true, nil
	}
	if s.mandatory == 0 {
		return true, nil
	}

	key := fmt.Sprintf("%d_%d", s.mandatory, userID)
	if exp, ok := s.cache[key]; ok && time.Now().Unix() < exp {
		// Valid cache, return true (we only cache successful memberships, or we should cache the result boolean?)
		// The python code caches the actual result for 30s.
		// Wait, the python code sets: `_MEMBERSHIP_CACHE[key] = (result, now + ttl)`
	}

	// For simplicity in this mock, we will implement full cache later, for now call API
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

	if result {
		s.cache[key] = time.Now().Add(s.cacheTTL).Unix()
	}

	return result, nil
}
