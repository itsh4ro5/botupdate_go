package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itsh4ro5/botupdate/internal/events"
	"github.com/itsh4ro5/botupdate/internal/models"
)

type MsgKey struct {
	ChatID int64
	MsgID  int
}

func (r *Router) setMapping(ctx context.Context, k1, k2 MsgKey) {
	key1 := fmt.Sprintf("%d_%d", k1.ChatID, k1.MsgID)
	key2 := fmt.Sprintf("%d_%d", k2.ChatID, k2.MsgID)

	r.msgMapMutex.Lock()
	if r.msgMapCache == nil {
		r.msgMapCache = make(map[string]string)
	}
	r.msgMapCache[key1] = key2
	r.msgMapCache[key2] = key1
	r.msgMapMutex.Unlock()

	// Asynchronously save to DB
	go func() {
		_ = r.store.SetMessageMapping(context.Background(), key1, key2)
		_ = r.store.SetMessageMapping(context.Background(), key2, key1)
	}()
}

func (r *Router) getMapping(ctx context.Context, k MsgKey) (MsgKey, bool) {
	key := fmt.Sprintf("%d_%d", k.ChatID, k.MsgID)
	
	r.msgMapMutex.RLock()
	val, ok := r.msgMapCache[key]
	r.msgMapMutex.RUnlock()

	if !ok {
		return MsgKey{}, false
	}

	var chatID int64
	var msgID int
	fmt.Sscanf(val, "%d_%d", &chatID, &msgID)
	return MsgKey{ChatID: chatID, MsgID: msgID}, true
}

func (r *Router) delMapping(ctx context.Context, k MsgKey) {
	key1 := fmt.Sprintf("%d_%d", k.ChatID, k.MsgID)
	
	r.msgMapMutex.Lock()
	val, ok := r.msgMapCache[key1]
	if ok {
		delete(r.msgMapCache, key1)
		delete(r.msgMapCache, val)
	}
	r.msgMapMutex.Unlock()

	if ok {
		go func() {
			_ = r.store.RemoveMessageMapping(context.Background(), key1)
			_ = r.store.RemoveMessageMapping(context.Background(), val)
		}()
	}
}

func (r *Router) handlePrivateMessage(ctx context.Context, msg *tgbotapi.Message) {
	topic, err := r.support.EnsureTopic(ctx, msg.From, false)
	if err != nil || topic == nil {
		log.Printf("Could not ensure topic for user %d: %v", msg.From.ID, err)
		return
	}

	copyMsg := tgbotapi.NewCopyMessage(r.support.GetSupportGroupID(), msg.Chat.ID, msg.MessageID)
	copyMsg.ReplyToMessageID = topic.MessageThread

	if msg.ReplyToMessage != nil {
		userKey := MsgKey{ChatID: msg.Chat.ID, MsgID: msg.ReplyToMessage.MessageID}
		if suppKey, ok := r.getMapping(ctx, userKey); ok {
			copyMsg.ReplyToMessageID = suppKey.MsgID
		}
	}

	sentMsg, err := r.bot.CopyMessage(copyMsg)
	if err == nil {
		userKey := MsgKey{ChatID: msg.Chat.ID, MsgID: msg.MessageID}
		suppKey := MsgKey{ChatID: r.support.GetSupportGroupID(), MsgID: sentMsg.MessageID}

		r.setMapping(ctx, userKey, suppKey)
		log.Printf("MAPPING CREATED: suppKey={%d, %d} targetKey={%d, %d}", suppKey.ChatID, suppKey.MsgID, userKey.ChatID, userKey.MsgID)

		supportMsg := &models.SupportMessage{
			ID:             int64(msg.MessageID),
			ConversationID: msg.From.ID,
			SenderType:     "incoming",
			SenderName:     msg.From.FirstName,
			Text:           msg.Text,
			Timestamp:      time.Now(),
			TelegramMsgID:  msg.MessageID,
		}

		events.Publish(events.TypeSupportMessage, events.SeverityInfo, map[string]interface{}{
			"message": supportMsg,
		})
		_ = r.store.AddSupportMessage(ctx, msg.From.ID, supportMsg)
	} else {
		log.Printf("Failed to copy message to support topic: %v", err)
	}
}

func (r *Router) handleSupportReply(ctx context.Context, msg *tgbotapi.Message) {
	var topicID int
	if msg.ReplyToMessage != nil {
		topicID = msg.ReplyToMessage.MessageID
	}

	var targetUserID int64

	if topicID != 0 {
		targetUserID = r.support.GetUserIDByTopicID(topicID)
	}

	var userMsgID int
	if msg.ReplyToMessage != nil {
		replyToSupportMsgID := msg.ReplyToMessage.MessageID
		suppKey := MsgKey{ChatID: msg.Chat.ID, MsgID: replyToSupportMsgID}
		if mappedKey, ok := r.getMapping(ctx, suppKey); ok {
			targetUserID = mappedKey.ChatID
			userMsgID = mappedKey.MsgID
		}
	}

	if targetUserID == 0 {
		return
	}

	copyMsg := tgbotapi.NewCopyMessage(targetUserID, msg.Chat.ID, msg.MessageID)
	if userMsgID != 0 {
		copyMsg.ReplyToMessageID = userMsgID
	}

	sentMsg, err := r.bot.CopyMessage(copyMsg)
	if err != nil && copyMsg.ReplyToMessageID != 0 {
		copyMsg.ReplyToMessageID = 0
		sentMsg, err = r.bot.CopyMessage(copyMsg)
	}

	if err == nil {
		suppKey := MsgKey{ChatID: msg.Chat.ID, MsgID: msg.MessageID}
		userKey := MsgKey{ChatID: targetUserID, MsgID: sentMsg.MessageID}

		r.setMapping(ctx, suppKey, userKey)

		events.Publish(events.TypeSupportMessage, events.SeverityInfo, map[string]interface{}{
			"user_id":   targetUserID,
			"direction": "outgoing",
		})
	} else {
		log.Printf("Failed to forward admin reply to user: %v", err)
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Failed to deliver message to user: %v", err)))
	}
}

func (r *Router) HandleDelMessage(ctx context.Context, msg *tgbotapi.Message) {
	log.Printf("==== PHASE 1: /del TRACE ====")
	log.Printf("/del received = true")
	log.Printf("chat ID = %d", msg.Chat.ID)
	log.Printf("sender/admin ID = %d", msg.From.ID)
	log.Printf("message ID = %d", msg.MessageID)
	log.Printf("reply-to message = %v", msg.ReplyToMessage != nil)
	if msg.ReplyToMessage != nil {
		log.Printf("reply-to message ID = %d", msg.ReplyToMessage.MessageID)
		log.Printf("reply-to chat ID = %d", msg.ReplyToMessage.Chat.ID)
	}
	log.Printf("command text = %s", msg.Text)
	log.Printf("HandleDelMessage entered = true")

	isAdmin := r.isAdminMessage(ctx, msg)
	log.Printf("==== PHASE 7: CHECK ADMIN PERMISSION ====")
	log.Printf("telegram sender ID = %d", msg.From.ID)
	log.Printf("AuthService result = %v", isAdmin)

	if !isAdmin || msg.ReplyToMessage == nil {
		log.Printf("Aborting: isAdmin=%v, hasReply=%v", isAdmin, msg.ReplyToMessage != nil)
		return
	}

	suppKey := MsgKey{ChatID: msg.Chat.ID, MsgID: msg.ReplyToMessage.MessageID}
	log.Printf("==== PHASE 3: TRACE /del LOOKUP ====")
	log.Printf("DEL RECEIVED")
	log.Printf("support chat ID = %d", msg.Chat.ID)
	log.Printf("reply message ID = %d", msg.ReplyToMessage.MessageID)
	log.Printf("calculated suppKey = {%d, %d}", suppKey.ChatID, suppKey.MsgID)

	targetKey, exists := r.getMapping(ctx, suppKey)

	log.Printf("mapping found = %v", exists)
	if !exists {
		log.Printf("==== PHASE 4: IF MAPPING IS NOT FOUND ====")
		log.Printf("ALL CURRENT KEYS IN MAP: (omitted for persistent map)")
	}

	if exists {
		log.Printf("==== PHASE 5: IF MAPPING IS FOUND ====")
		log.Printf("target chat ID = %d", targetKey.ChatID)
		log.Printf("target message ID = %d", targetKey.MsgID)

		_, err := r.bot.Request(tgbotapi.NewDeleteMessage(targetKey.ChatID, targetKey.MsgID))
		log.Printf("delete success = %v", err == nil)
		if err != nil {
			log.Printf("Telegram error description: %v", err)
			sentMsg, _ := r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Delete failed: %v", err)))

			if sentMsg.MessageID != 0 {
				_ = r.store.AddScheduledDelete(ctx, &models.ScheduledDelete{
					ChatID:    sentMsg.Chat.ID,
					MessageID: sentMsg.MessageID,
					DeleteAt:  time.Now().Add(20 * time.Minute),
				})
			}
			log.Printf("USER DELETE = FAIL. Stopping here.")
			return
		}
		log.Printf("USER DELETE = SUCCESS")

		log.Printf("==== PHASE 6: VERIFY SUPPORT-SIDE DELETE ====")
		_, errSupp := r.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.ReplyToMessage.MessageID))
		log.Printf("SUPPORT DELETE = %v", errSupp == nil)

		_, errCmd := r.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
		log.Printf("COMMAND DELETE = %v", errCmd == nil)

		r.delMapping(ctx, suppKey)
		log.Printf("MAP CLEANUP = SUCCESS")
	} else {
		log.Printf("Not deleting user message since mapping is lost. Attempting support delete only.")
		r.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.ReplyToMessage.MessageID))
		r.bot.Request(tgbotapi.NewDeleteMessage(msg.Chat.ID, msg.MessageID))
	}
}

func (r *Router) handleEditedMessage(ctx context.Context, msg *tgbotapi.Message) {
	key := MsgKey{ChatID: msg.Chat.ID, MsgID: msg.MessageID}
	targetKey, exists := r.getMapping(ctx, key)

	if !exists {
		return
	}

	if msg.Text != "" {
		editMsg := tgbotapi.NewEditMessageText(targetKey.ChatID, targetKey.MsgID, msg.Text)
		editMsg.Entities = msg.Entities
		r.bot.Send(editMsg)
	} else if msg.Caption != "" {
		editCap := tgbotapi.NewEditMessageCaption(targetKey.ChatID, targetKey.MsgID, msg.Caption)
		editCap.CaptionEntities = msg.CaptionEntities
		r.bot.Send(editCap)
	}
}

// HandleReaction implements the telegram.ReactionHandler interface for bidirectional reaction sync.
func (r *Router) HandleReaction(ctx context.Context, reaction *models.MessageReactionUpdated) {
	// Loop prevention: ignore reactions set by the bot itself
	if reaction.User != nil && reaction.User.ID == r.bot.Self.ID {
		return
	}
	if reaction.ActorChat != nil && reaction.ActorChat.ID == r.bot.Self.ID {
		return
	}

	key := MsgKey{ChatID: reaction.Chat.ID, MsgID: reaction.MessageID}

	targetKey, exists := r.getMapping(ctx, key)

	if !exists {
		// Not tracked, ignore
		return
	}

	// Prepare the configuration to set the reaction on the opposite side
	params := make(tgbotapi.Params)
	params.AddNonZero64("chat_id", targetKey.ChatID)
	params.AddNonZero("message_id", targetKey.MsgID)

	if len(reaction.NewReaction) > 0 {
		b, err := json.Marshal(reaction.NewReaction)
		if err == nil {
			params["reaction"] = string(b)
		}
	} else {
		// Empty array to remove all reactions
		params["reaction"] = "[]"
	}

	// Make the API request directly using MakeRequest
	_, err := r.bot.MakeRequest("setMessageReaction", params)
	if err != nil {
		log.Printf("Failed to sync reaction for message {ChatID: %d, MsgID: %d}: %v", targetKey.ChatID, targetKey.MsgID, err)
	}
}
