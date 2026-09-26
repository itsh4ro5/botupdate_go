package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itsh4ro5/botupdate/internal/events"
	"github.com/itsh4ro5/botupdate/internal/models"
)

// HandleStart processes the /start command
func (r *Router) HandleStart(ctx context.Context, msg *tgbotapi.Message) {
	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}

	userID := msg.From.ID
	isNewUser := false

	user, exists := state.Users[userID]
	if !exists {
		isNewUser = true
		user = &models.User{
			ID:              userID,
			Username:        msg.From.UserName,
			FirstName:       msg.From.FirstName,
			LastName:        msg.From.LastName,
			JoinedAt:        time.Now().Unix(),
			Demos:           make(map[string]interface{}),
			TnCAccepted:     false,
			UnlockedBatches: []string{},
			ReferralCount:   0,
			TotalInvited:    0,
		}
		state.Users[userID] = user
	}
	if user.UnlockedBatches == nil {
		user.UnlockedBatches = []string{}
	}

	// CHECK REFERRAL DEEP LINK PARAMETER
	args := msg.CommandArguments()
	var targetBatchToOpen string
	var referrerID int64

	if args != "" {
		param := strings.TrimSpace(args)

		if strings.HasPrefix(param, "batch_") {
			parts := strings.Split(param, "_")
			if len(parts) >= 3 {
				targetBatchToOpen = parts[1]
				if id, err := strconv.ParseInt(parts[2], 10, 64); err == nil {
					referrerID = id
				}
			} else if len(parts) == 2 {
				targetBatchToOpen = parts[1]
			}
		} else if id, err := strconv.ParseInt(param, 10, 64); err == nil {
			referrerID = id
		} else if strings.HasPrefix(param, "ref_") {
			parts := strings.Split(param, "_")
			if len(parts) >= 2 {
				if id, err := strconv.ParseInt(parts[len(parts)-1], 10, 64); err == nil {
					referrerID = id
				}
			}
		}

		if isNewUser && referrerID != 0 && referrerID != userID {
			user.PendingReferral = referrerID
		}
		if targetBatchToOpen != "" {
			user.PendingBatch = targetBatchToOpen
		} else {
			user.PendingBatch = "" // Clear stale state
		}
	} else {
		// No payload, clear stale pending batch state so normal start menu shows
		user.PendingBatch = ""
	}

	_ = r.store.SetUser(ctx, userID, user)

	if isNewUser {
		events.Publish(events.TypeUserJoined, events.SeverityInfo, map[string]interface{}{
			"user_id":  userID,
			"username": user.Username,
			"name":     user.FirstName,
		})
	}
	// Get or Create topic
	r.getOrCreateTopic(ctx, msg.From)

	// LOADING ANIMATION
	loadingMsg, _ := r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ *Loading, please wait...*"))
	time.Sleep(700 * time.Millisecond)

	isAdmin := r.auth.IsAdmin(ctx, userID)
	isOwner := r.auth.IsOwner(ctx, userID)

	if isOwner || isAdmin {
		r.showRoleSelector(msg.Chat.ID, loadingMsg.MessageID, isOwner)
	} else {
		// Membership check
		isMember := r.checkMembership(ctx, userID)
		if isMember {
			if !user.TnCAccepted {
				r.showTnCMenu(msg.Chat.ID, loadingMsg.MessageID)
			} else {
				if user.PendingReferral != 0 {
					// Handle referral completion
					r.processSuccessfulReferral(ctx, userID, user.PendingReferral)
					user.PendingReferral = 0
					_ = r.store.SetUser(ctx, userID, user)
				}
				r.showHomeMenu(msg.Chat.ID, loadingMsg.MessageID, user)
			}
		} else {
			if !state.NewUsersAllowed {
				r.bot.Send(tgbotapi.NewEditMessageText(msg.Chat.ID, loadingMsg.MessageID, "🚫 *Entry Closed!*"))
				return
			}
			r.showJoinChannelMenu(msg.Chat.ID, loadingMsg.MessageID)
		}
	}
}

// HandleID processes the /id command
func (r *Router) HandleID(ctx context.Context, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := fmt.Sprintf("ID: `%d`", chatID)
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	r.bot.Send(reply)
}

// HandlePing processes the /ping command
func (r *Router) HandlePing(ctx context.Context, msg *tgbotapi.Message) {
	t := time.Now()
	reply := tgbotapi.NewMessage(msg.Chat.ID, "🏓 **Pinging BotAPI...**")
	reply.ParseMode = "Markdown"
	sent, err := r.bot.Send(reply)
	if err == nil {
		duration := time.Since(t).Milliseconds()
		edit := tgbotapi.NewEditMessageText(msg.Chat.ID, sent.MessageID, fmt.Sprintf("🏓 **Pong!**\n⚡ **Speed:** `%dms`\n🛡️ **Protocol:** `Golang Telegram-Bot-API`", duration))
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)
	}
}

// HandleMyInfo processes the /myinfo command
func (r *Router) HandleMyInfo(ctx context.Context, msg *tgbotapi.Message) {
	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}
	user, ok := state.Users[msg.From.ID]
	if !ok {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Aapka data nahi mila. /start press karein."))
		return
	}

	vipStatus := "Nahi"
	if user.Tier == "vip" {
		vipStatus = "Haan 👑"
	}

	info := fmt.Sprintf("👤 **Aapki Information:**\n\n🆔 **ID:** `%d`\n👤 **Name:** %s\n🏅 **VIP Status:** %s\n👥 **Total Invites:** %d\n💰 **Coins:** %d",
		user.ID, user.FirstName, vipStatus, user.TotalInvited, user.ReferralCount)

	reply := tgbotapi.NewMessage(msg.Chat.ID, info)
	reply.ParseMode = "Markdown"
	r.bot.Send(reply)
}
