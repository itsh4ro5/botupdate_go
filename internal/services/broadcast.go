package services

import (
	"context"
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itsh4ro5/botupdate/internal/database"
)

// BroadcastService manages mass messaging
type BroadcastService struct {
	bot    *tgbotapi.BotAPI
	store  database.Store
	mu     sync.Mutex
	active bool
}

func NewBroadcastService(bot *tgbotapi.BotAPI, store database.Store) *BroadcastService {
	return &BroadcastService{
		bot:   bot,
		store: store,
	}
}

// RunBroadcast executes a broadcast to all users in a separate goroutine
func (s *BroadcastService) RunBroadcast(ctx context.Context, text string) error {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return nil // Already running
	}
	s.active = true
	s.mu.Unlock()

	state, err := s.store.Load(ctx)
	if err != nil {
		s.mu.Lock()
		s.active = false
		s.mu.Unlock()
		return err
	}

	go func() {
		defer func() {
			s.mu.Lock()
			s.active = false
			s.mu.Unlock()
		}()

		success := 0
		failed := 0

		for userID := range state.Users {
			select {
			case <-ctx.Done():
				log.Println("Broadcast cancelled.")
				return
			default:
				msg := tgbotapi.NewMessage(userID, text)
				_, err := s.bot.Send(msg)
				if err != nil {
					failed++
					// If forbidden, could mark user as blocked
				} else {
					success++
				}
				// Simple rate limiting
				time.Sleep(50 * time.Millisecond)
			}
		}

		log.Printf("Broadcast complete. Success: %d, Failed: %d", success, failed)
	}()

	return nil
}
