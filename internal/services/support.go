package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/models"
	"github.com/itsh4ro5/botupdate/internal/telegram"
)

// SupportService manages the forum topic mapping and synchronization
type SupportService struct {
	bot            *tgbotapi.BotAPI
	api            *telegram.APIClient
	store          database.Store
	supportGroupID int64
	topicLocks     sync.Map // Prevents duplicate topic creation for the same user concurrently
}

func NewSupportService(bot *tgbotapi.BotAPI, api *telegram.APIClient, store database.Store, groupID int64) *SupportService {
	return &SupportService{
		bot:            bot,
		api:            api,
		store:          store,
		supportGroupID: groupID,
	}
}

func (s *SupportService) GetSupportGroupID() int64 {
	return s.supportGroupID
}

// EnsureTopic retrieves or creates a forum topic for the user
func (s *SupportService) EnsureTopic(ctx context.Context, user *tgbotapi.User, isRetry bool) (*models.SupportTopic, error) {
	if s.supportGroupID == 0 {
		return nil, nil
	}

	// 1. Lock per user
	lock, _ := s.topicLocks.LoadOrStore(user.ID, &sync.Mutex{})
	mutex := lock.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()

	// 2. Check existing state
	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	if topic, exists := state.UserTopics[user.ID]; exists {
		return topic, nil
	}

	// 3. Create new topic in Telegram via custom API client
	title := BuildSupportTopicName(user)

	threadID, err := s.api.CreateForumTopic(s.supportGroupID, title)
	if err != nil {
		if !isRetry {
			log.Printf("Topic Creation Error for %d: %v. Retrying...", user.ID, err)
			time.Sleep(1 * time.Second) // Python does some native refresh, here we just backoff and retry
			// To avoid deadlock on retry since we already have the lock, we unlock and recall or just do the logic.
			// Actually we can just call EnsureTopic with isRetry=true outside the lock, or do the retry inline.
		}

		if !isRetry {
			mutex.Unlock()
			time.Sleep(1 * time.Second)
			topic, err := s.EnsureTopic(ctx, user, true)
			mutex.Lock() // Relock to satisfy defer
			return topic, err
		}

		log.Printf("Topic Creation Error (peer still invalid): %v", err)
		return nil, err
	}

	newTopic := &models.SupportTopic{
		UserID:        user.ID,
		TopicID:       threadID,
		MessageThread: threadID,
	}

	if err := s.store.SetSupportTopic(ctx, user.ID, newTopic); err != nil {
		// Clean up on save failure? Not strictly needed if state is memory first
		return nil, err
	}

	// Send NEW USER TICKET message
	dispName := user.FirstName
	if dispName == "" {
		dispName = user.UserName
	}
	if dispName == "" {
		dispName = "User"
	}
	text := fmt.Sprintf("🚨 *NEW USER TICKET*\n👤 [%s](tg://user?id=%d)\n🆔 `%d`\n", dispName, user.ID, user.ID)
	msg := tgbotapi.NewMessage(s.supportGroupID, text)
	msg.ReplyToMessageID = threadID
	msg.ParseMode = "Markdown"
	s.bot.Send(msg)

	return newTopic, nil
}

// BuildSupportTopicName creates a safe, non-empty topic name based on the user's details.
func BuildSupportTopicName(user *tgbotapi.User) string {
	name := user.FirstName
	if name == "" {
		name = user.UserName
	}
	if name == "" {
		name = "User"
	}
	if len(name) > 20 {
		name = name[:20]
	}
	return fmt.Sprintf("%s (%d)", name, user.ID)
}
