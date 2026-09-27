package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itsh4ro5/botupdate/internal/events"
	"github.com/itsh4ro5/botupdate/internal/models"
)

func (s *BatchService) HandleCallback(ctx context.Context, query *tgbotapi.CallbackQuery) {
	data := query.Data
	if strings.HasPrefix(data, "get_f_") {
		s.handleGetFree(ctx, query)
	} else if strings.HasPrefix(data, "view_p_") {
		s.handleViewPaid(ctx, query)
	} else if strings.HasPrefix(data, "view_s_") {
		s.handleViewSpecial(ctx, query)
	} else if strings.HasPrefix(data, "join_f_") {
		s.handleJoinFree(ctx, query)
	} else if strings.HasPrefix(data, "req_access_") {
		s.handleReqAccess(ctx, query)
	} else if strings.HasPrefix(data, "unlock_s_") {
		s.handleUnlockSpecial(ctx, query)
	} else if data == "unlock_all_free" {
		s.handleUnlockAllFree(ctx, query)
	} else if strings.HasPrefix(data, "share_btn_") {
		s.handleShareBtn(ctx, query)
	}
}

func (s *BatchService) handleShareBtn(ctx context.Context, query *tgbotapi.CallbackQuery) {
	cidStr := strings.TrimPrefix(query.Data, "share_btn_")

	s.bot.Request(tgbotapi.NewCallback(query.ID, ""))
	botUsername := s.bot.Self.UserName
	refLink := fmt.Sprintf("https://t.me/%s?start=batch_%s", botUsername, cidStr)
	shareText := "Join this awesome batch on H4R!"

	tgShareUrl := fmt.Sprintf("https://t.me/share/url?url=%s&text=%s", refLink, shareText)
	waShareUrl := fmt.Sprintf("https://api.whatsapp.com/send?text=%s", refLink+" "+shareText)

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("✈️ Share on Telegram", tgShareUrl),
			tgbotapi.NewInlineKeyboardButtonURL("💬 Share on WhatsApp", waShareUrl),
		),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "u_main")),
	)

	edit := tgbotapi.NewEditMessageTextAndMarkup(
		query.Message.Chat.ID,
		query.Message.MessageID,
		"🔗 **Share Batch**\n\nClick below to invite your friends directly to this batch:",
		kb,
	)
	edit.ParseMode = "Markdown"
	s.bot.Send(edit)
}

func (s *BatchService) handleReqAccess(ctx context.Context, query *tgbotapi.CallbackQuery) {
	uid := query.From.ID
	cidStr := strings.TrimPrefix(query.Data, "req_access_")
	cid, err := strconv.ParseInt(cidStr, 10, 64)
	if err != nil {
		return
	}

	state, err := s.store.Load(ctx)
	if err != nil {
		return
	}

	if state.PaidLocked {
		s.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Sorry, but at this moment the paid batch is locked. When it will unlock I will inform you."))
		return
	}

	// Membership check would go here but we assume true for now, can implement check via MembershipService or just GetChatMember.

	// Check if already in channel
	memConfig := tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{ChatID: cid, UserID: uid},
	}
	m, err := s.bot.GetChatMember(memConfig)
	if err == nil && (m.Status == "member" || m.Status == "creator" || m.Status == "administrator") {
		s.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "  Already joined!"))
		return
	}

	hasActive := false
	for _, req := range state.PendingRequests {
		if req.UserID == uid && req.BatchID == cid {
			hasActive = true
			break
		}
	}
	if hasActive {
		s.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "  You already have an active request/link for this batch!"))
		return
	}

	s.bot.Request(tgbotapi.NewCallback(query.ID, "  Generating Link..."))

	expireDate := int(time.Now().Add(60 * time.Second).Unix())
	inviteConfig := tgbotapi.CreateChatInviteLinkConfig{
		ChatConfig:         tgbotapi.ChatConfig{ChatID: cid},
		Name:               fmt.Sprintf("Req-%d", uid),
		ExpireDate:         expireDate,
		CreatesJoinRequest: true,
	}

	resp, err := s.bot.Request(inviteConfig)
	if err != nil {
		s.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "  Failed to generate link."))
		return
	}

	var invite tgbotapi.ChatInviteLink
	json.Unmarshal(resp.Result, &invite)
	hash := invite.InviteLink

	inviteMapping := &models.InviteMapping{
		Hash:    hash,
		UserID:  uid,
		BatchID: cid,
		OneTime: true,
	}
	_ = s.store.SetInviteLink(ctx, hash, inviteMapping)

	reqID := fmt.Sprintf("%d_%d", uid, cid)
	pendingReq := &models.PendingRequest{
		UserID:    uid,
		BatchID:   cid,
		Requested: time.Now(),
	}
	_ = s.store.SetPendingRequest(ctx, reqID, pendingReq)

	bname := fmt.Sprintf("Batch %d", cid)
	if name, exists := state.AllChats[cid]; exists {
		bname = name
	}

	if s.SupportService != nil && s.SupportService.GetSupportGroupID() != 0 {
		topic, _ := s.SupportService.EnsureTopic(ctx, query.From, false)
		topicID := 0
		if topic != nil {
			topicID = topic.TopicID
		}

		notificationText := fmt.Sprintf("📩 <b>NEW REQUEST</b>\n👤 User: <a href=\"tg://user?id=%d\">%s</a>\n📦 Batch: <b>%s</b>\n🔗 Link: %s\n\n⚡ <b>Action:</b>\n/demo %s\n/per %s", uid, query.From.FirstName, bname, hash, hash, hash)

		msg := tgbotapi.NewMessage(s.SupportService.GetSupportGroupID(), notificationText)
		msg.ParseMode = "HTML"
		if topicID != 0 {
			msg.ReplyToMessageID = topicID
		}
		sentMsg, err := s.bot.Send(msg)
		if err == nil {
			supportMsg := &models.SupportMessage{
				ID:             int64(sentMsg.MessageID),
				ConversationID: uid,
				SenderType:     "incoming",
				SenderName:     "System (Join Request)",
				Text:           notificationText,
				Timestamp:      time.Now(),
				TelegramMsgID:  sentMsg.MessageID,
			}
			events.Publish(events.TypeSupportMessage, events.SeverityInfo, map[string]interface{}{
				"message": supportMsg,
			})
			_ = s.store.AddSupportMessage(ctx, uid, supportMsg)
		}
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL("🚀 Join Batch", hash)),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔗 Share Batch with Friends", fmt.Sprintf("share_btn_%d", cid))),
	)

	sentMsg := tgbotapi.NewMessage(query.Message.Chat.ID, fmt.Sprintf("  <b>Link Generated!</b>\n\n<b>%s</b>\n\n  <i>Request sent to admins. Please wait for approval.</i>\n  <i>(Link expires in 1 min)</i>", bname))
	sentMsg.ReplyMarkup = kb
	sentMsg.ParseMode = "HTML"
	s.bot.Send(sentMsg)
}

func (s *BatchService) handleUnlockSpecial(ctx context.Context, query *tgbotapi.CallbackQuery) {
	uid := query.From.ID
	cidStr := strings.TrimPrefix(query.Data, "unlock_s_")
	cid, err := strconv.ParseInt(cidStr, 10, 64)
	if err != nil {
		return
	}

	state, err := s.store.Load(ctx)
	if err != nil {
		return
	}

	cost := int64(1)
	if c, ok := state.BatchCoins[cid]; ok {
		cost = c
	}

	user, ok := state.Users[uid]
	if !ok {
		return
	}

	if int64(user.ReferralCount) >= cost {
		user.ReferralCount -= int(cost)
		user.UnlockedBatches = append(user.UnlockedBatches, cidStr)
		s.store.SetUser(ctx, uid, user)

		events.Publish(events.TypeAccessGranted, events.SeverityInfo, map[string]interface{}{
			"user_id":  uid,
			"batch_id": cid,
			"method":   "referral_coins",
		})

		s.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "🎉 Batch Successfully Unlocked!"))

		cleanID := strings.ReplaceAll(cidStr, "-100", "")
		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL("🔗 Join Channel", fmt.Sprintf("https://t.me/c/%s/1", cleanID))),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Main Menu", "u_main")),
		)

		edit := tgbotapi.NewEditMessageTextAndMarkup(
			query.Message.Chat.ID,
			query.Message.MessageID,
			"🎉 **Batch Unlocked!**\n\nAb aap directly is batch me join kar sakte hain.",
			kb,
		)
		edit.ParseMode = "Markdown"
		s.bot.Send(edit)
	} else {
		s.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "❌ Aapke paas enough coins nahi hain."))
	}
}

func (s *BatchService) handleGetFree(ctx context.Context, query *tgbotapi.CallbackQuery) {
	cidStr := strings.TrimPrefix(query.Data, "get_f_")
	cid, err := strconv.ParseInt(cidStr, 10, 64)
	if err != nil {
		return
	}
	s.bot.Request(tgbotapi.NewCallback(query.ID, ""))

	bname := fmt.Sprintf("Batch %d", cid)
	state, err := s.store.Load(ctx)
	if err == nil {
		if name, exists := state.AllChats[cid]; exists {
			bname = name
		}
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔗 Join Channel", fmt.Sprintf("join_f_%d", cid))),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📤 Share Channel", fmt.Sprintf("share_btn_%d", cid))),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "u_main")),
	)

	edit := tgbotapi.NewEditMessageTextAndMarkup(
		query.Message.Chat.ID,
		query.Message.MessageID,
		fmt.Sprintf("📦 **%s**\n\nSelect an action below:", bname),
		kb,
	)
	edit.ParseMode = "Markdown"
	s.bot.Send(edit)
}

func (s *BatchService) handleJoinFree(ctx context.Context, query *tgbotapi.CallbackQuery) {
	uid := query.From.ID
	cidStr := strings.TrimPrefix(query.Data, "join_f_")
	cid, err := strconv.ParseInt(cidStr, 10, 64)
	if err != nil {
		s.bot.Request(tgbotapi.NewCallback(query.ID, "Invalid Batch ID"))
		return
	}

	state, err := s.store.Load(ctx)
	if err != nil {
		s.bot.Request(tgbotapi.NewCallback(query.ID, "Database error"))
		return
	}

	user, ok := state.Users[uid]
	if !ok {
		s.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "❌ User not found. Send /start first."))
		return
	}

	// FREE_LIMIT = 3
	const FreeLimit = 3
	alreadyUsedSlot := false
	for _, joinedCID := range user.FreeBatchesJoined {
		if joinedCID == cid {
			alreadyUsedSlot = true
			break
		}
	}

	isVip := user.Tier == "vip"
	needsUnlock := !isVip && !user.FreeUnlocked && !alreadyUsedSlot && len(user.FreeBatchesJoined) >= FreeLimit

	if needsUnlock {
		s.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		if user.ReferralCount >= 1 {
			kb := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔓 Unlock All Free Batches (Cost: 1 Coin)", "unlock_all_free")),
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "u_main")),
			)
			edit := tgbotapi.NewEditMessageTextAndMarkup(query.Message.Chat.ID, query.Message.MessageID,
				fmt.Sprintf("🔒 **Free Batches Locked**\n\nAapne apni **%d free** batches use kar li hain.\nAapke paas **%d Coins** hain. **1 Coin** use karke saari (baaki) Free Batches hamesha ke liye unlock karein — dobara coin nahi lagega.", FreeLimit, user.ReferralCount),
				kb)
			edit.ParseMode = "Markdown"
			s.bot.Send(edit)
		} else {
			kb := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🎁 Get Refer Link", "menu_refer")),
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "u_main")),
			)
			edit := tgbotapi.NewEditMessageTextAndMarkup(query.Message.Chat.ID, query.Message.MessageID,
				fmt.Sprintf("🔒 **Free Batches Locked**\n\nAapne apni **%d free** batches use kar li hain. Aage ke liye **1 Coin** chahiye.\nApne dosto ko refer karke coins earn karein!", FreeLimit),
				kb)
			edit.ParseMode = "Markdown"
			s.bot.Send(edit)
		}
		return
	}

	// Check if already in channel
	memConfig := tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: cid,
			UserID: uid,
		},
	}
	m, err := s.bot.GetChatMember(memConfig)
	if err == nil && (m.Status == "member" || m.Status == "creator" || m.Status == "administrator") {
		s.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "  Already Joined!"))
		return
	}

	// Cooldown and Active request checks
	// Active request checking (same logic as python)
	hasActive := false
	for _, req := range state.PendingRequests {
		if req.UserID == uid && req.BatchID == cid {
			hasActive = true
			break
		}
	}
	if hasActive {
		s.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "  You already have an active request/link for this batch!"))
		return
	}

	// Create Invite Link
	expireDate := int(time.Now().Add(60 * time.Second).Unix())
	inviteConfig := tgbotapi.CreateChatInviteLinkConfig{
		ChatConfig:         tgbotapi.ChatConfig{ChatID: cid},
		Name:               fmt.Sprintf("Free-%d", uid),
		ExpireDate:         expireDate,
		CreatesJoinRequest: true,
	}

	resp, err := s.bot.Request(inviteConfig)
	if err != nil {
		s.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "  Failed to generate link. Try again later."))
		return
	}

	// Register link request
	// (Add to pending requests or link map in DB)
	var invite tgbotapi.ChatInviteLink
	// Parse response
	json.Unmarshal(resp.Result, &invite)

	// Create link map
	hash := invite.InviteLink

	bname := fmt.Sprintf("Batch %d", cid)
	if name, exists := state.AllChats[cid]; exists {
		bname = name
	}

	if !user.FreeUnlocked && !alreadyUsedSlot {
		user.FreeBatchesJoined = append(user.FreeBatchesJoined, cid)
		s.store.SetUser(ctx, uid, user)
	}

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL("🚀 Join Channel", hash)),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📤 Share Channel", fmt.Sprintf("share_btn_%d", cid))),
	)

	s.bot.Request(tgbotapi.NewCallback(query.ID, ""))

	sentMsg := tgbotapi.NewMessage(query.Message.Chat.ID, fmt.Sprintf("  <b>Link Generated!</b>\n\n<b>%s</b>\n\n  <i>Request auto-approved.</i>\n  <i>(Expires in 1 min)</i>", bname))
	sentMsg.ReplyMarkup = kb
	sentMsg.ParseMode = "HTML"
	s.bot.Send(sentMsg)
}

func (s *BatchService) handleViewPaid(ctx context.Context, query *tgbotapi.CallbackQuery) {
	cidStr := strings.TrimPrefix(query.Data, "view_p_")
	cid, err := strconv.ParseInt(cidStr, 10, 64)
	if err != nil {
		return
	}

	state, err := s.store.Load(ctx)
	if err != nil {
		return
	}

	if state.PaidLocked {
		s.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Sorry, but at this moment the paid batch is locked. When it will unlock I will inform you."))
		return
	}

	s.bot.Request(tgbotapi.NewCallback(query.ID, ""))

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔗 Join Channel", fmt.Sprintf("req_access_%d", cid))),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📤 Share Channel", fmt.Sprintf("share_btn_%d", cid))),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "u_main")),
	)

	edit := tgbotapi.NewEditMessageTextAndMarkup(
		query.Message.Chat.ID,
		query.Message.MessageID,
		"  **Premium Access:**\nClick below.",
		kb,
	)
	edit.ParseMode = "Markdown"
	s.bot.Send(edit)
}

func (s *BatchService) handleViewSpecial(ctx context.Context, query *tgbotapi.CallbackQuery) {
	uid := query.From.ID
	cidStr := strings.TrimPrefix(query.Data, "view_s_")
	cid, err := strconv.ParseInt(cidStr, 10, 64)
	if err != nil {
		return
	}

	s.bot.Request(tgbotapi.NewCallback(query.ID, ""))

	state, err := s.store.Load(ctx)
	if err != nil {
		return
	}

	bname := fmt.Sprintf("Special Batch %d", cid)
	if name, exists := state.AllChats[cid]; exists {
		bname = name
	}

	cost := int64(1)
	if c, ok := state.BatchCoins[cid]; ok {
		cost = c
	}

	user, ok := state.Users[uid]
	if !ok {
		return
	}

	isUnlocked := false
	for _, ub := range user.UnlockedBatches {
		if ub == cidStr {
			isUnlocked = true
			break
		}
	}

	pts := int64(user.ReferralCount)
	coinWord := "Coin"
	if cost != 1 {
		coinWord = "Coins"
	}

	var statusStr, desc string
	kb := tgbotapi.NewInlineKeyboardMarkup()

	if isUnlocked {
		cleanID := strings.ReplaceAll(cidStr, "-100", "")
		kb.InlineKeyboard = append(kb.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL("🔗 Join Channel", fmt.Sprintf("https://t.me/c/%s/1", cleanID))))
		statusStr = "🎉 Unlocked!"
		desc = "Aapne is batch ko successfully unlock kar liya hai."
	} else {
		if pts >= cost {
			kb.InlineKeyboard = append(kb.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🔓 Unlock Batch (Cost: %d %s)", cost, coinWord), fmt.Sprintf("unlock_s_%d", cid))))
			statusStr = "🔒 Locked"
			desc = fmt.Sprintf("Aapke paas **%d Coins** hain. Aap %d %s use karke is batch ko unlock kar sakte hain.", pts, cost, coinWord)
		} else {
			kb.InlineKeyboard = append(kb.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🎁 Get Refer Link", "menu_refer")))
			statusStr = "🔒 Locked (Not Enough Coins)"
			desc = fmt.Sprintf("Aapke paas enough coins nahi hain. Is batch ko unlock karne ke liye aapko **%d %s** chahiye.\nApne dosto ko refer karke coins earn karein!", cost, coinWord)
		}
	}

	kb.InlineKeyboard = append(kb.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📤 Share Channel", fmt.Sprintf("share_btn_%d", cid))))
	kb.InlineKeyboard = append(kb.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "u_main")))

	edit := tgbotapi.NewEditMessageTextAndMarkup(
		query.Message.Chat.ID,
		query.Message.MessageID,
		fmt.Sprintf("✨ **SPECIAL BATCH:** `%s`\n\n**Status:** `%s`\n\n%s", bname, statusStr, desc),
		kb,
	)
	edit.ParseMode = "Markdown"
	s.bot.Send(edit)
}

func (s *BatchService) handleUnlockAllFree(ctx context.Context, query *tgbotapi.CallbackQuery) {
	uid := query.From.ID
	state, err := s.store.Load(ctx)
	if err != nil {
		return
	}

	user, ok := state.Users[uid]
	if !ok {
		return
	}

	if user.ReferralCount >= 1 {
		user.ReferralCount -= 1
		user.FreeUnlocked = true
		s.store.SetUser(ctx, uid, user)

		s.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "✅ Saari Free Batches hamesha ke liye unlock ho gayi! Ab aap koi bhi free batch bina coins ke join kar sakte hain."))

		// Edit message back to main
		edit := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, "✅ Free Batches Unlocked! Navigate from Main Menu.")
		s.bot.Send(edit)
	} else {
		s.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "❌ Aapke paas enough coins nahi hain."))
	}
}
