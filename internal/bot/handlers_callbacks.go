package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itsh4ro5/botupdate/internal/models"
)

func (r *Router) handleMyBatches(ctx context.Context, query *tgbotapi.CallbackQuery, page int) {
	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}

	user, ok := state.Users[query.From.ID]
	if !ok {
		return
	}

	uid := query.From.ID

	unlockedMap := make(map[int64]bool)
	for _, idStr := range user.UnlockedBatches {
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil && id != 0 {
			unlockedMap[id] = true
		}
	}

	type EligibleBatch struct {
		ID     int64
		Name   string
		Status string
	}
	var eligible []EligibleBatch

	for cid, cname := range state.AllChats {
		isUnlocked := unlockedMap[cid]
		memConfig := tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{ChatID: cid, UserID: uid},
		}
		m, err := r.bot.GetChatMember(memConfig)

		status := ""
		if err == nil && (m.Status == "member" || m.Status == "creator" || m.Status == "administrator" || m.Status == "restricted") {
			status = "Joined"
		}

		if status != "" {
			eligible = append(eligible, EligibleBatch{ID: cid, Name: cname, Status: status})
		} else if isUnlocked {
			eligible = append(eligible, EligibleBatch{ID: cid, Name: cname, Status: "Referral Unlocked"})
		}
	}

	perPage := 8
	startIdx := page * perPage
	endIdx := startIdx + perPage
	total := len(eligible)

	if startIdx >= total {
		startIdx = 0
		endIdx = perPage
	}
	if endIdx > total {
		endIdx = total
	}

	var pageBatches []EligibleBatch
	if total > 0 {
		pageBatches = eligible[startIdx:endIdx]
	}

	text := "Yahan wo sabhi batches hain jisme aap join hain ya unlocked hain.\nClick karke access karein:"
	if total == 0 {
		text = "Aap abhi kisi bhi batch me join nahi hain."
	}

	var keyboard [][]tgbotapi.InlineKeyboardButton
	for _, b := range pageBatches {
		cleanID := strings.ReplaceAll(fmt.Sprintf("%d", b.ID), "-100", "")
		url := fmt.Sprintf("https://t.me/c/%s/1", cleanID)

		btnText := "⚡ " + b.Name
		if b.Status != "Joined" {
			btnText = "✨ " + b.Name + " [SPECIAL UNLOCKED]"
		}

		keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(btnText, url),
		))
	}

	var navRow []tgbotapi.InlineKeyboardButton
	if page > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("🔙 Back", fmt.Sprintf("my_batches_%d", page-1)))
	}
	if endIdx < total {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Next ➡️", fmt.Sprintf("my_batches_%d", page+1)))
	}
	if len(navRow) > 0 {
		keyboard = append(keyboard, navRow)
	}

	keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Main Menu", "u_main"),
	))

	markup := tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	edit := tgbotapi.NewEditMessageTextAndMarkup(query.Message.Chat.ID, query.Message.MessageID, text, markup)
	r.bot.Send(edit)
}

// handleAllBatches shows category list
func (r *Router) handleAllBatches(ctx context.Context, query *tgbotapi.CallbackQuery) {
	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}

	text := "📂 *Select a Category*"
	var keyboard [][]tgbotapi.InlineKeyboardButton

	// Categories are stored in state.Categories
	for i, cat := range state.Categories {
		keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(cat, fmt.Sprintf("showcat_%d", i)),
		))
	}

	keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Main Menu", "u_main"),
	))

	markup := tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	edit := tgbotapi.NewEditMessageTextAndMarkup(query.Message.Chat.ID, query.Message.MessageID, text, markup)
	edit.ParseMode = "Markdown"
	r.bot.Send(edit)
}

// handleShowCat shows free/paid/special type selector for a category
func (r *Router) handleShowCat(ctx context.Context, query *tgbotapi.CallbackQuery, catIndex int) {
	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}

	if catIndex < 0 || catIndex >= len(state.Categories) {
		return
	}
	category := state.Categories[catIndex]

	text := fmt.Sprintf("📂 *Category:* `%s`\n\nSelect batch type:", category)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟢 Free Batches", fmt.Sprintf("listcat_%d_free_0", catIndex)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟡 Paid Batches", fmt.Sprintf("listcat_%d_paid_0", catIndex)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔴 Special Batches", fmt.Sprintf("listcat_%d_special_0", catIndex)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Categories", "all_batches_0"),
		),
	)

	edit := tgbotapi.NewEditMessageTextAndMarkup(query.Message.Chat.ID, query.Message.MessageID, text, keyboard)
	edit.ParseMode = "Markdown"
	r.bot.Send(edit)
}

// handleListCat shows the batches in a specific category and type
func (r *Router) handleListCat(ctx context.Context, query *tgbotapi.CallbackQuery, catIndex int, batchType string, page int) {
	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}

	if catIndex < 0 || catIndex >= len(state.Categories) {
		return
	}
	category := state.Categories[catIndex]

	var batches []*models.Batch

	switch batchType {
	case "free":
		for _, b := range state.FreeBatches {
			if b.Category == category {
				batches = append(batches, b)
			}
		}
	case "paid":
		for _, b := range state.PaidBatches {
			if b.Category == category {
				batches = append(batches, b)
			}
		}
	case "special":
		for _, b := range state.SpecialBatches {
			if b.Category == category {
				batches = append(batches, b)
			}
		}
	}

	perPage := 10
	startIdx := page * perPage
	endIdx := startIdx + perPage

	total := len(batches)
	if startIdx >= total {
		startIdx = 0
		endIdx = perPage
	}
	if endIdx > total {
		endIdx = total
	}

	var pageBatches []*models.Batch
	if total > 0 {
		pageBatches = batches[startIdx:endIdx]
	}

	text := fmt.Sprintf("📂 *%s* ➔ `%s` Batches", category, strings.ToUpper(batchType))
	if len(pageBatches) == 0 {
		text = fmt.Sprintf("📂 *%s* ➔ `%s` Batches\n\nNo batches available in this section.", category, strings.ToUpper(batchType))
	}

	var keyboard [][]tgbotapi.InlineKeyboardButton
	for _, b := range pageBatches {
		var prefix string
		switch b.Type {
		case "free":
			prefix = "get_f_"
		case "paid":
			prefix = "view_p_"
		case "special":
			prefix = "view_s_"
		}
		keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(b.Name, fmt.Sprintf("%s%d", prefix, b.ID)),
		))
	}

	var navRow []tgbotapi.InlineKeyboardButton
	if page > 0 {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ Previous", fmt.Sprintf("listcat_%d_%s_%d", catIndex, batchType, page-1)))
	}
	if endIdx < total {
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("Next ➡️", fmt.Sprintf("listcat_%d_%s_%d", catIndex, batchType, page+1)))
	}
	if len(navRow) > 0 {
		keyboard = append(keyboard, navRow)
	}

	keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Back", fmt.Sprintf("showcat_%d", catIndex)),
	))

	markup := tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	edit := tgbotapi.NewEditMessageTextAndMarkup(query.Message.Chat.ID, query.Message.MessageID, text, markup)
	edit.ParseMode = "Markdown"
	r.bot.Send(edit)
}

func (r *Router) handleMenuRefer(ctx context.Context, query *tgbotapi.CallbackQuery) {
	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}

	uid := query.From.ID
	user, ok := state.Users[uid]
	if !ok {
		return
	}

	botUsername := r.bot.Self.UserName
	refLink := fmt.Sprintf("https://t.me/%s?start=%d", botUsername, uid)
	shareText := "Join me on H4R — the smartest way to crack your exam!"

	tgShareUrl := fmt.Sprintf("https://t.me/share/url?url=%s&text=%s", refLink, shareText)
	waShareUrl := fmt.Sprintf("https://api.whatsapp.com/send?text=%s", refLink+" "+shareText)

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("✈️ Share on Telegram", tgShareUrl),
			tgbotapi.NewInlineKeyboardButtonURL("💬 Share on WhatsApp", waShareUrl),
		),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Main Menu", "u_main")),
	)

	vipStr := ""
	if user.Tier == "vip" {
		vipStr = "👑 Your Tag: **VIP Referrer**\n"
	}

	text := fmt.Sprintf("🎁 **Refer & Earn Program**\n━━━━━━━━━━━━━━━━━━━━━━━━━\nInvite your friends and earn 1 coin on every successful refer!\n\n🏆 **Milestone Bonus:** Every 5 successful refers = **+1 EXTRA Coin!**\n👑 **VIP Tag:** Cross 25 total refers to unlock the VIP Referrer tag!\n\n%s👥 Total Referred Users: `%d`\n💰 Total Earnings: `%d` Coins\n\n🔗 Your Referral Link:\n`%s`\n\nClick the buttons below to share directly with your friends! 👇", vipStr, user.TotalInvited, user.ReferralCount, refLink)

	edit := tgbotapi.NewEditMessageTextAndMarkup(query.Message.Chat.ID, query.Message.MessageID, text, kb)
	edit.ParseMode = "Markdown"
	r.bot.Send(edit)
}

func (r *Router) handleTestBot(ctx context.Context, query *tgbotapi.CallbackQuery) {
	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}

	if state.TestBotLocked {
		r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Sorry, but at this moment the test bot is locked. When it will unlock I will inform you."))
		return
	}

	r.bot.Request(tgbotapi.NewCallback(query.ID, ""))

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL("🤖 Test Series Bot", "https://t.me/your_test_bot")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Main Menu", "u_main")),
	)

	edit := tgbotapi.NewEditMessageTextAndMarkup(
		query.Message.Chat.ID,
		query.Message.MessageID,
		"  **Test Series Hub**\n\nTap the button below to start practicing with our Test Bot!",
		kb,
	)
	edit.ParseMode = "Markdown"
	r.bot.Send(edit)
}

func (r *Router) handleRoleSelector(ctx context.Context, query *tgbotapi.CallbackQuery) {
	isOwner := r.auth.IsOwner(ctx, query.From.ID)
	r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
	r.showRoleSelector(query.Message.Chat.ID, query.Message.MessageID, isOwner)
}

func (r *Router) handleGotoOwnerPanel(ctx context.Context, query *tgbotapi.CallbackQuery) {
	if !r.auth.IsOwner(ctx, query.From.ID) {
		r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "  Access Denied! Owner Only."))
		return
	}
	r.bot.Request(tgbotapi.NewCallback(query.ID, ""))

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔒 Security", "dash_locks"),
			tgbotapi.NewInlineKeyboardButtonData("💾 Database", "dash_db"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📦 Batches", "dash_batches"),
			tgbotapi.NewInlineKeyboardButtonData("🧑\u200d💼 Staff", "dash_staff"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📢 Comms", "dash_comms"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Analytics", "dash_stats"),
		),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🤖 Userbot Login & Stats", "userbot_details")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔄 Switch Panel", "role_selector")),
	)

	edit := tgbotapi.NewEditMessageTextAndMarkup(query.Message.Chat.ID, query.Message.MessageID, "👑 **SYSTEM MASTER TERMINAL**\n\nSelect a module below:", kb)
	edit.ParseMode = "Markdown"
	r.bot.Send(edit)
}

func (r *Router) handleGotoAdminPanel(ctx context.Context, query *tgbotapi.CallbackQuery) {
	if !r.auth.IsAdmin(ctx, query.From.ID) {
		r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "  Access Denied! Admins Only."))
		return
	}
	r.bot.Request(tgbotapi.NewCallback(query.ID, ""))

	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👥 Users", "adash_users"),
			tgbotapi.NewInlineKeyboardButtonData("✅ Approvals", "adash_approvals"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📦 Batches", "adash_batches"),
			tgbotapi.NewInlineKeyboardButtonData("📢 Comms", "adash_comms"),
		),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔄 Switch Panel", "role_selector")),
	)

	edit := tgbotapi.NewEditMessageTextAndMarkup(query.Message.Chat.ID, query.Message.MessageID, "🛡 **ADMIN DASHBOARD**\n\nSelect a module below:", kb)
	edit.ParseMode = "Markdown"
	r.bot.Send(edit)
}

func (r *Router) handleGotoUserPanel(ctx context.Context, query *tgbotapi.CallbackQuery) {
	r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}
	user, ok := state.Users[query.From.ID]
	if !ok {
		return
	}
	r.showHomeMenu(query.Message.Chat.ID, query.Message.MessageID, user)
}
