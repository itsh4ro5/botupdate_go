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

	topicsCache map[int64]*models.SupportTopic
	topicsMutex sync.RWMutex
}

func NewSupportService(bot *tgbotapi.BotAPI, api *telegram.APIClient, store database.Store, groupID int64) *SupportService {
	s := &SupportService{
		bot:            bot,
		api:            api,
		store:          store,
		supportGroupID: groupID,
		topicsCache:    make(map[int64]*models.SupportTopic),
	}

	state, err := store.Load(context.Background())
	if err == nil && state.UserTopics != nil {
		for k, v := range state.UserTopics {
			s.topicsCache[k] = v
		}
	}

	return s
}

func (s *SupportService) GetSupportGroupID() int64 {
	return s.supportGroupID
}

// GetUserIDByTopicID searches the cache to reverse-lookup a user by their topic ID
func (s *SupportService) GetUserIDByTopicID(topicID int) int64 {
	s.topicsMutex.RLock()
	defer s.topicsMutex.RUnlock()

	for uid, topic := range s.topicsCache {
		if topic.MessageThread == topicID || topic.TopicID == topicID {
			return uid
		}
	}
	return 0
}

// EnsureTopic retrieves or creates a forum topic for the user
func (s *SupportService) EnsureTopic(ctx context.Context, user *tgbotapi.User, isRetry bool) (*models.SupportTopic, error) {
	if s.supportGroupID == 0 {
		return nil, nil
	}

	// 1. Check existing state (fast path)
	s.topicsMutex.RLock()
	if topic, exists := s.topicsCache[user.ID]; exists {
		s.topicsMutex.RUnlock()
		return topic, nil
	}
	s.topicsMutex.RUnlock()

	// 2. Lock per user
	lock, _ := s.topicLocks.LoadOrStore(user.ID, &sync.Mutex{})
	mutex := lock.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()

	// 3. Double-check inside lock
	s.topicsMutex.RLock()
	if topic, exists := s.topicsCache[user.ID]; exists {
		s.topicsMutex.RUnlock()
		return topic, nil
	}
	s.topicsMutex.RUnlock()

	// 4. Create new topic in Telegram via custom API client
	title := BuildSupportTopicName(user)

	threadID, err := s.api.CreateForumTopic(s.supportGroupID, title)
	if err != nil {
		if !isRetry {
			log.Printf("Topic Creation Error for %d: %v. Retrying...", user.ID, err)
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

	s.topicsMutex.Lock()
	s.topicsCache[user.ID] = newTopic
	s.topicsMutex.Unlock()

	go func() {
		_ = s.store.SetSupportTopic(context.Background(), user.ID, newTopic)
	}()

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
