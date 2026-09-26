package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itsh4ro5/botupdate/internal/models"
)

func (r *Router) getOrCreateTopic(ctx context.Context, user *tgbotapi.User) {
	_, err := r.support.EnsureTopic(ctx, user, false)
	if err != nil {
		log.Printf("Failed to ensure support topic for user %d: %v", user.ID, err)
	}
}

func (r *Router) checkMembership(ctx context.Context, userID int64) bool {
	result, err := r.membership.CheckMembership(ctx, userID)
	if err != nil {
		log.Printf("Failed to check membership for user %d: %v", userID, err)
		return false // Or true if fail-open? Python usually handles errors inside
	}
	return result
}

func (r *Router) showRoleSelector(chatID int64, messageID int, isOwner bool) {
	text := "⚙️ *Control Panel*"

	// Similar to python:
	// build_role_selector_kb
	var keyboard tgbotapi.InlineKeyboardMarkup
	if isOwner {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("👑 Owner Panel", "goto_owner_panel")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🛠️ Admin Panel", "goto_admin_panel")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("👤 User Panel", "goto_user_panel")),
		)
	} else {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🛠️ Admin Panel", "goto_admin_panel")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("👤 User Panel", "goto_user_panel")),
		)
	}

	edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, keyboard)
	edit.ParseMode = "Markdown"
	r.bot.Send(edit)
}

func (r *Router) showTnCMenu(chatID int64, messageID int) {
	text := "⚠️ *STRICT WARNING & TERMS OF SERVICE*\n\n" +
		"🇬🇧 *ENGLISH:*\n" +
		"If you leave the Main Channel or block this bot, you will be *INSTANTLY REMOVED* from ALL joined groups and channels.\n\n" +
		"🇮🇳 *HINDI:*\n" +
		"Agar aapne Main Channel ko chhoda (leave kiya) ya is bot ko block kiya, toh aapko sabhi groups aur channels se *TURANT NIKAL* diya jayega.\n\n" +
		"✅ _Click 'I Read & Accept' only if you agree to these terms._"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✅ I Read & Accept", "accept_tnc")),
	)

	edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, keyboard)
	edit.ParseMode = "Markdown"
	r.bot.Send(edit)
}

func (r *Router) showHomeMenu(chatID int64, messageID int, user *models.User) {
	var text string
	var keyboard tgbotapi.InlineKeyboardMarkup

	if user.PendingBatch != "" {
		bname := "Shared Batch"
		state, err := r.store.Load(context.Background())
		if err == nil {
			batchID, _ := strconv.ParseInt(user.PendingBatch, 10, 64)
			if name, ok := state.AllChats[batchID]; ok {
				bname = name
			}
		}
		text = fmt.Sprintf("🎉 *You were invited to a Batch!*\n\n📦 *Batch Name:* `%s`\n\nAapke dost ne aapko is batch me join karne ke liye invite kiya hai. Neeche diye gaye button par click karke details dekhein aur turant join karein!", bname)

		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🚀 Open Shared Batch", fmt.Sprintf("open_batch_%s", user.PendingBatch)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🏠 Go to Main Menu", "clear_pending_batch"),
			),
		)
	} else if user.Tier == "vip" {
		text = fmt.Sprintf("👑 *[Elite Referrer] %s*\n━━━━━━━━━━━━━━━━━━━━━━━━━\nWelcome back! You have successfully referred `%d` students so far.\n💰 *Wallet Balance:* `%d` Coins\n\n✨ *Your VIP dashboard is ready:*", user.FirstName, user.TotalInvited, user.ReferralCount)
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("👑 My Batches (Elite Access)", "my_batches_0")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🌟 All Batches", "all_batches_0")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL("📢 Batch Updates", "https://t.me/YourUpdateChannel")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("💎 VIP Course Materials", "vip_materials")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🎁 Claim Monthly Bonus", "vip_monthly_bonus")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🚀 Refer & Earn", "menu_refer")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🤖 Test Bot", "test_bot")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL("🎥 How to use the bot", "https://t.me/c/123/456")),
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("💎 My Info", "my_info")),
		)
	} else {
		text = "🌟 *Welcome to the Premium Hub!* 🌟\n\n*4️⃣ Browse batches*\n🇬🇧 📚 *My Batches* = batches you already have. 🌐 *All Batches* = List of all section courese.\n🇮🇳 📚 *My Batches* = jo aapke paas already hain. 🌐 *All Batches* = Sare courese ka section hai.\n\n*5️⃣ Test Series Website*\n🇬🇧 Tap *🤖 Test Bot* Here you can practice your question, daily new Current Affairs and also notes of all exam.\n🇮🇳 *🤖 Test Bot* par tap karke aap aapne exam ka practice kar sakte hai saath hi current affairs and notes bhi hai.\n\n*6️⃣ Earn coins to unlock free batches*\n🇬🇧 Tap *🎁 Refer & Earn*, share your personal link with friends. Every real join earns you a coin — coins unlock free and special batches.\n🇮🇳 *🎁 Refer & Earn* par tap karke apna personal link dosto ko bhejein. Har real join par coin milta hai — coins se free batches aur special batches unlock hoti hain.\n\n*7️⃣ Check your stats anytime*\n🇬🇧 Tap *ℹ️ My Info* to see your ID, total refers, and coin balance.\n🇮🇳 *ℹ️ My Info* par tap karke apni ID, total refers aur coin balance dekhein.\n\n"
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📚 My Batches", "my_batches_0"),
				tgbotapi.NewInlineKeyboardButtonData("🌐 All Batches", "all_batches_0"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔍 Search Batch", "search_batch_start"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("📢 Batch Updates", getEnvWithFallback("BATCH_UPDATE_CHANNEL_LINK", "https://t.me/YourUpdateChannel")),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🤖 Test Bot", "test_bot"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🎁 Refer & Earn", "menu_refer"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("ℹ️ My Info", "my_info"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonURL("🎥 How to use the bot", "https://t.me/c/123/456"),
			),
		)
	}

	edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, keyboard)
	edit.ParseMode = "Markdown"
	r.bot.Send(edit)
}

func (r *Router) showJoinChannelMenu(chatID int64, messageID int) {
	text := "📢 *Join Main Channel First / पहले Channel Join Karein*\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━\n\n" +
		"🇬🇧 *1️⃣* Tap *📢 Join Channel* below and join it.\n" +
		"🇮🇳 *1️⃣* Neeche *📢 Join Channel* par tap karke channel join karein.\n\n" +
		"🇬🇧 *2️⃣* Come back here and tap *✅ I've Joined* to unlock the bot.\n" +
		"🇮🇳 *2️⃣* Wapas yahan aakar *✅ I've Joined* par tap karein aur bot unlock karein.\n\n" +
		"💡 _This unlocks batches, free demos, and referral rewards._"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL("📢 Join Channel", getEnvWithFallback("MANDATORY_CHANNEL_LINK", "https://t.me/YourChannel"))),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✅ I've Joined", "verify")),
	)

	edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, keyboard)
	edit.ParseMode = "Markdown"
	r.bot.Send(edit)
}

// getEnvWithFallback is a small helper to grab an env var or return default
func getEnvWithFallback(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}

func (r *Router) handleMyInfo(ctx context.Context, query *tgbotapi.CallbackQuery) {
	state, err := r.store.Load(ctx)
	if err != nil {
		r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Error loading data"))
		return
	}

	user, ok := state.Users[query.From.ID]
	if !ok {
		r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "User not found"))
		return
	}

	isVIP := user.Tier == "vip"
	var txt string

	if isVIP {
		txt = fmt.Sprintf("👑 **[Elite Referrer] — MY INFO** 👑\n━━━━━━━━━━━━━━━━━━━━━━━━━\n🆔 **User ID:** `%d`\n🏷️ **Tag:** `👑 VIP Referrer`\n👥 **Total Refers:** `%d`\n💰 **Wallet:** `%d` Coins\n\n💎 *Enjoy zero cooldowns, exclusive materials, and monthly bonuses — thank you for being Elite.*", query.From.ID, user.TotalInvited, user.ReferralCount)
	} else {
		txt = fmt.Sprintf("👤 **MY INFO**\n🆔 **ID:** `%d`\n👥 **Total Refers:** `%d`\n🎁 **Available Coins:** `%d`\n\n💡 *In coins ka use karke aap koi bhi Special Batch ya saari Free Batches unlock kar sakte hain.*", query.From.ID, user.TotalInvited, user.ReferralCount)
	}

	r.bot.Request(tgbotapi.NewCallback(query.ID, ""))

	edit := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, txt)
	edit.ParseMode = "Markdown"

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "u_main")),
	)
	edit.ReplyMarkup = &kb
	r.bot.Send(edit)
}

func (r *Router) processSuccessfulReferral(ctx context.Context, userID int64, referrerID int64) {
	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}

	referrer, ok := state.Users[referrerID]
	if !ok {
		return
	}

	referrer.ReferralCount++
	referrer.TotalInvited++

	milestoneHit := referrer.TotalInvited%5 == 0
	if milestoneHit {
		referrer.ReferralCount++
	}

	totalNow := referrer.TotalInvited
	newlyVIP := totalNow >= 25 && referrer.Tier != "vip"
	if newlyVIP {
		referrer.Tier = "vip"
	}

	if referee, ok := state.Users[userID]; ok {
		referee.ReferredBy = referrerID
		_ = r.store.SetUser(ctx, userID, referee)
	}

	_ = r.store.SetUser(ctx, referrerID, referrer)

	// Send support topic notification
	if r.support.GetSupportGroupID() != 0 {
		topic, _ := r.support.EnsureTopic(ctx, &tgbotapi.User{ID: referrerID}, false)
		if topic != nil {
			msg := tgbotapi.NewMessage(r.support.GetSupportGroupID(), fmt.Sprintf("🎉 *%d refers successful by the user*", totalNow))
			msg.ReplyToMessageID = topic.MessageThread
			msg.ParseMode = "Markdown"
			r.bot.Send(msg)
		}
	}

	// Welcome bonus for referrer
	if !referrer.WelcomeBonusClaimed && referrer.ReferredBy != 0 {
		referrer.ReferralCount++
		referrer.WelcomeBonusClaimed = true
		_ = r.store.SetUser(ctx, referrerID, referrer)

		msg := tgbotapi.NewMessage(referrerID, "🔓 *WELCOME BONUS UNLOCKED!*\n\nAapne apna pehla successful refer kar diya, isliye aapka pending *1 Coin Welcome Bonus* bhi ab wallet me add ho gaya hai! 🎉")
		msg.ParseMode = "Markdown"
		r.bot.Send(msg)
	}

	// General Referrer notification
	msg := tgbotapi.NewMessage(referrerID, fmt.Sprintf("🎉 *REFERRAL SUCCESSFUL!*\n\nAapke bheje gaye link se ek naye user ne saare steps complete kar liye hain! Aapko *1 Coin* mil gaya hai. 🎁\n💰 Total Balance: `%d` Coins", referrer.ReferralCount))
	msg.ParseMode = "Markdown"
	r.bot.Send(msg)

	if milestoneHit {
		mMsg := tgbotapi.NewMessage(referrerID, fmt.Sprintf("🏆 *MILESTONE BONUS!*\n\nAapne *%d successful refers* poore kar liye hain — har 5 refers par 1 EXTRA Coin milta hai. Aapke wallet me *+1 Bonus Coin* add ho gaya hai! 🎁", totalNow))
		mMsg.ParseMode = "Markdown"
		r.bot.Send(mMsg)
	}

	if newlyVIP {
		vMsg := tgbotapi.NewMessage(referrerID, "👑 *VIP REFERRER UNLOCKED!*\n\nAapne 25+ dosto ko refer kar diya hai! Aapko ab bot ke andar *👑 VIP Referrer* tag mil gaya hai — yeh 'My Info' aur 'Refer & Earn' section me dikhega.")
		vMsg.ParseMode = "Markdown"
		r.bot.Send(vMsg)
	}
}

func (r *Router) handleAcceptTnC(ctx context.Context, query *tgbotapi.CallbackQuery) {
	userID := query.From.ID
	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}

	user, ok := state.Users[userID]
	if !ok {
		return
	}

	user.TnCAccepted = true
	_ = r.store.SetUser(ctx, userID, user)

	// Callback answer
	callback := tgbotapi.NewCallback(query.ID, "T&C Accepted!")
	r.bot.Request(callback)

	// Show home menu or refer processing
	if user.PendingReferral != 0 {
		r.processSuccessfulReferral(ctx, userID, user.PendingReferral)
		user.PendingReferral = 0
		_ = r.store.SetUser(ctx, userID, user)
	}

	r.showHomeMenu(query.Message.Chat.ID, query.Message.MessageID, user)
}

func (r *Router) handleVerify(ctx context.Context, query *tgbotapi.CallbackQuery) {
	userID := query.From.ID
	isMember := r.checkMembership(ctx, userID)
	if !isMember {
		callback := tgbotapi.NewCallbackWithAlert(query.ID, "❌ You haven't joined the main channel yet! Please join and try again.")
		r.bot.Request(callback)
		return
	}

	callback := tgbotapi.NewCallback(query.ID, "✅ Membership Verified!")
	r.bot.Request(callback)

	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}

	user, ok := state.Users[userID]
	if !ok {
		return
	}

	if !user.TnCAccepted {
		r.showTnCMenu(query.Message.Chat.ID, query.Message.MessageID)
	} else {
		if user.PendingReferral != 0 {
			r.processSuccessfulReferral(ctx, userID, user.PendingReferral)
			user.PendingReferral = 0
			_ = r.store.Save(ctx, state)
		}
		r.showHomeMenu(query.Message.Chat.ID, query.Message.MessageID, user)
	}
}
