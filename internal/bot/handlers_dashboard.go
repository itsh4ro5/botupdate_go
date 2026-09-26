package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// showOwnerDashboard renders the main owner dashboard
func (r *Router) showOwnerDashboard(chatID int64, msgID int) {
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

	text := "👑 **SYSTEM MASTER TERMINAL**\n\nSelect a module below:"

	if msgID == 0 {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ReplyMarkup = kb
		msg.ParseMode = "Markdown"
		r.bot.Send(msg)
	} else {
		edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, msgID, text, kb)
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)
	}
}

// showAdminPanel renders the admin dashboard
func (r *Router) showAdminPanel(chatID int64, msgID int) {
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

	text := "🛡 **ADMIN DASHBOARD**\n\nSelect a module below:"

	if msgID == 0 {
		msg := tgbotapi.NewMessage(chatID, text)
		msg.ReplyMarkup = kb
		msg.ParseMode = "Markdown"
		r.bot.Send(msg)
	} else {
		edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, msgID, text, kb)
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)
	}
}

// HandleAdmin processes the /admin command
func (r *Router) HandleAdmin(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ You are not authorized to use this command."))
		return
	}
	if r.auth.IsOwner(ctx, msg.From.ID) {
		r.showOwnerDashboard(msg.Chat.ID, 0)
	} else {
		r.showAdminPanel(msg.Chat.ID, 0)
	}
}

func (r *Router) handleDashboardCallback(ctx context.Context, query *tgbotapi.CallbackQuery) {
	uid := query.From.ID
	data := query.Data
	chatID := query.Message.Chat.ID
	msgID := query.Message.MessageID

	// Critical Fix: Map the correct user ID since query.Message.From points to the bot itself
	query.Message.From = query.From

	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}

	if data == "dash_home" {
		r.wizardMutex.Lock()
		delete(r.adminWizard, uid)
		r.wizardMutex.Unlock()
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		if r.auth.IsOwner(ctx, uid) {
			r.showOwnerDashboard(chatID, msgID)
		} else {
			r.showAdminPanel(chatID, msgID)
		}
		return
	}

	if data == "dash_locks" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		kb := [][]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("🔐 System Lockdown: %s", getToggleText(!state.NewUsersAllowed)), "toggle_lockdown"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("🆓 Free Batches: %s", getLockText(state.FreeLocked)), "toggle_free"),
				tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("💰 Paid Batches: %s", getLockText(state.PaidLocked)), "toggle_paid"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("🤖 Test Bot: %s", getLockText(state.TestBotLocked)), "toggle_testbot"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("🛠️ Maintenance Mode: %s", getToggleText(state.MaintenanceMode)), "toggle_maintenance"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Terminal", "dash_home"),
			),
		}
		edit := tgbotapi.NewEditMessageText(chatID, msgID, "🔒 **Security & Access Control**")
		markup := tgbotapi.NewInlineKeyboardMarkup(kb...)
		edit.ReplyMarkup = &markup
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)
		return
	}

	if strings.HasPrefix(data, "toggle_") {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		key := strings.ToUpper(strings.TrimPrefix(data, "toggle_"))
		if key == "LOCKDOWN" {
			r.store.SetLockState(ctx, "lockdown", !state.NewUsersAllowed)
		} else if key == "MAINTENANCE" {
			r.store.SetMaintenanceMode(ctx, !state.MaintenanceMode)
		} else if key == "FREE" {
			r.store.SetLockState(ctx, "free", !state.FreeLocked)
		} else if key == "PAID" {
			r.store.SetLockState(ctx, "paid", !state.PaidLocked)
		} else if key == "TESTBOT" {
			r.store.SetLockState(ctx, "testbot", !state.TestBotLocked)
		}

		r.handleDashboardCallback(ctx, &tgbotapi.CallbackQuery{
			ID: query.ID, From: query.From, Data: "dash_locks", Message: query.Message,
		})
		return
	}

	if data == "dash_db" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		kb := [][]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📥 Download Backup", "act_backup"),
				tgbotapi.NewInlineKeyboardButtonData("🔄 Run Sync", "act_sync"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("👥 Download All Users List", "act_allusers"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🗄️ Store Batch Data (Scan)", "input_storebatch"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔍 Specific User Data", "input_userlookup"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🚫 Ban User", "input_ban"),
				tgbotapi.NewInlineKeyboardButtonData("✅ Unban User", "input_unban"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🎁 Gift Coin", "input_giftcoin"),
				tgbotapi.NewInlineKeyboardButtonData("☢️ Hard Delete User", "input_deluser"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "dash_home"),
			),
		}
		edit := tgbotapi.NewEditMessageText(chatID, msgID, "🛡️ **Database Tools**")
		markup := tgbotapi.NewInlineKeyboardMarkup(kb...)
		edit.ReplyMarkup = &markup
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)
		return
	}

	if data == "dash_batches" || data == "adash_batches" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		kb := [][]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("➕ Add Batch", "act_addbatch"),
				tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete Batch", "input_delbatch"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📁➕ Add Category", "input_addcat"),
				tgbotapi.NewInlineKeyboardButtonData("📁🗑️ Delete Category", "act_delcat"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🏷️ Set Batch Category", "input_setcat"),
				tgbotapi.NewInlineKeyboardButtonData("🧹 Empty Batch", "input_emptybatch"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📢 Post Batch Update", "input_updatepost"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🚀 Super Forwarder", "input_superfwd"),
				tgbotapi.NewInlineKeyboardButtonData("🛡️ Clean Unverified", "input_cleanbatch"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📝 Adv Caption Changer", "input_advcap"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📊 Batch Stats", "act_batchstats"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "dash_home"),
			),
		}
		edit := tgbotapi.NewEditMessageText(chatID, msgID, "📦 **Batches Management**")
		markup := tgbotapi.NewInlineKeyboardMarkup(kb...)
		edit.ReplyMarkup = &markup
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)
		return
	}

	if data == "dash_staff" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		kb := [][]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("➕ Add Admin", "input_addadmin"),
				tgbotapi.NewInlineKeyboardButtonData("➖ Remove Admin", "input_deladmin"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📋 Admin List", "act_adminlist"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "dash_home"),
			),
		}
		edit := tgbotapi.NewEditMessageText(chatID, msgID, "🧑\u200d💼 **Staff Management**")
		markup := tgbotapi.NewInlineKeyboardMarkup(kb...)
		edit.ReplyMarkup = &markup
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)
		return
	}

	if data == "dash_comms" || data == "adash_comms" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		kb := [][]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📢 Broadcast", "act_broadcast"),
				tgbotapi.NewInlineKeyboardButtonData("📝 Post Message", "act_post"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🤖 Set Test Bot", "input_settestbot"),
				tgbotapi.NewInlineKeyboardButtonData("👋 Set Welcome", "input_setwelcome"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "dash_home"),
			),
		}
		edit := tgbotapi.NewEditMessageText(chatID, msgID, "📢 **Communications**")
		markup := tgbotapi.NewInlineKeyboardMarkup(kb...)
		edit.ReplyMarkup = &markup
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)
		return
	}
	if data == "adash_users" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		kb := [][]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔍 Find User", "input_find"),
				tgbotapi.NewInlineKeyboardButtonData("📊 All Users", "act_allusers"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Ban User", "input_ban"),
				tgbotapi.NewInlineKeyboardButtonData("Unban User", "input_unban"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Reset User", "input_resetuser"),
				tgbotapi.NewInlineKeyboardButtonData("Kick User", "input_kick"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "dash_home"),
			),
		}
		edit := tgbotapi.NewEditMessageText(chatID, msgID, "👥 **User Management**")
		markup := tgbotapi.NewInlineKeyboardMarkup(kb...)
		edit.ReplyMarkup = &markup
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)
		return
	}

	if data == "adash_approvals" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		kb := [][]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ Approve Perm", "input_perm"),
				tgbotapi.NewInlineKeyboardButtonData("⏳ Demo Access", "input_demo"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⏰ Extend Access", "input_extend"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "dash_home"),
			),
		}
		edit := tgbotapi.NewEditMessageText(chatID, msgID, "✅ **Approvals & Access**")
		markup := tgbotapi.NewInlineKeyboardMarkup(kb...)
		edit.ReplyMarkup = &markup
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)
		return
	}

	if data == "userbot_details" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		if !r.auth.IsOwner(ctx, uid) {
			r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Access Denied! Owner only."))
			return
		}

		session := state.UserbotSession
		phone := state.UserbotPhone
		if phone == "" {
			phone = "Not Found"
		}

		var kb [][]tgbotapi.InlineKeyboardButton
		var text string

		if session != "" {
			text = fmt.Sprintf("  **USERBOT CONTROL PANEL**\n\n  **Status:** Active & Ready\n  **Logged in Number:** `%s`\n\n*Userbot is fully linked and ready to execute.*", phone)
			kb = [][]tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🚪 Logout (Delete Session)", "userbot_logout")),
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "dash_home")),
			}
		} else {
			text = "  **USERBOT CONTROL PANEL**\n\n  **Status:** NOT LOGGED IN\n\n*Koi active session nahi hai. Userbot features won't work. Kripya login karein.*"
			kb = [][]tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔑 Login Now", "input_userbotphone")),
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "dash_home")),
			}
		}

		edit := tgbotapi.NewEditMessageText(chatID, msgID, text)
		markup := tgbotapi.NewInlineKeyboardMarkup(kb...)
		edit.ReplyMarkup = &markup
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)
		return
	}

	if data == "userbot_logout" {
		if !r.auth.IsOwner(ctx, uid) {
			return
		}
		state.UserbotSession = ""
		r.store.Save(ctx, state)
		os.Remove("temp_owner.session")
		r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Session Deleted Successfully!"))

		edit := tgbotapi.NewEditMessageText(chatID, msgID, "  **Userbot is now LOGGED OUT.**")
		markup := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Back", "dash_home")),
		)
		edit.ReplyMarkup = &markup
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)
		return
	}

	if strings.HasPrefix(data, "input_") {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		cmdName := strings.TrimPrefix(data, "input_")

		if cmdName == "addcat" {
			log.Printf("ADD_CATEGORY_START\nuser=%d\nstate=%s", uid, "call_cmd_addcat")
		}

		r.wizardMutex.Lock()
		r.adminWizard[uid] = &WizardState{Step: "call_cmd_" + cmdName}
		r.wizardMutex.Unlock()

		prompts := map[string]string{
			"addadmin":     "Send User ID to make Admin:",
			"deladmin":     "Send User ID to remove from Admin:",
			"ban":          "Send User ID to Ban:",
			"unban":        "Send User ID to Unban:",
			"kick":         "Send User ID and Batch ID\nFormat: `uid bid`",
			"find":         "Send Username to find:",
			"resetuser":    "Send User ID to reset:",
			"demo":         "Send Link and Time:\nFormat: `link 10h`",
			"perm":         "Send Link to approve:",
			"extend":       "Send User ID, Batch ID, Hours:\nFormat: `uid bid 24`",
			"gendemo":      "Send User ID and Batch ID to generate a 3-hour Demo link:\nFormat: `uid bid`",
			"settestbot":   "Send new Test Bot link:",
			"setwelcome":   "Send Batch ID and Welcome Msg:\nFormat: `bid message`",
			"delbatch":     "Send Type and ID:\nFormat: `free 123` or `paid 123` or `special 123`",
			"addcat":       "Send Name for new Category:",
			"setcat":       "Send Batch ID(s) (comma ya space lagakar):\nFormat: `-100x, -100y`",
			"emptybatch":   "  **DHYAN DEIN!**\nSend Batch ID jisko poora khali (empty) karna hai:\nFormat: `-100123456789`",
			"advcap":       "📝 **Advanced Caption Changer (Step 1/5)**\n\nUs **Channel ID** ko bhejein jiske captions edit karne hain (e.g. `-10012345678`):",
			"cleanbatch":   "🛡️ **Clean Unverified Users (Anti-Leech)**\n\nUs **Batch/Channel ID** ko bhejein jise clean karna hai (e.g. `-100123456789`).",
			"storebatch":   "🗄️ **Store Batch Data**\n\nJis channel ka purana data (Videos/PDFs) Firebase me index karna hai, uska Chat ID bhejein:\nFormat: `-100123456789`",
			"updatepost":   "📢 **Post Batch Update**\n\nJis purane batch ka update post channel me bhejna hai, uska **Batch ID** bhejein (e.g. `-100123456789`):",
			"superfwd":     "🚀 **Super Forwarder (Step 1/7)**\n\nUs **Source Channel ID** ko bhejein jahan se files (content) uthani hain (e.g. `-10012345678`):",
			"userbotphone": "  **Apna Phone Number bhejein**\nCountry code ke sath (Jaise: `+919876543210`):",
			"userbototp":   "  **OTP Bhejein**\n  *OTP spaces me bhejein!* Jaise: `1 2 3 4 5`:",
			"userbotpass":  "  **2FA Password bhejein:**",
			"userlookup":   "🔍 **Specific User Data**\n\nUser ka User ID bhejein:",
			"deluser":      "☢️ **HARD DELETE USER**\n\nJis user ka data database se **poori tarah mitana hai**, uska **User ID** bhejein.\n*(Warning: Ye action undo nahi hoga)*:",
			"giftcoin":     "🎁 **Gift Coin**\n\nUser ID aur Amount bhejein:\nFormat: `uid amount` (e.g. `12345 5`):",
		}

		promptMsg, ok := prompts[cmdName]
		if !ok {
			promptMsg = "Please send input for " + cmdName
		}

		msg := tgbotapi.NewMessage(chatID, promptMsg)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true}
		r.bot.Send(msg)
		return
	}

	if data == "act_adminlist" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		r.HandleAdminList(ctx, query)
		return
	}
	if data == "act_backup" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		r.HandleBackup(ctx, query.Message)
		return
	}
	if data == "act_sync" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		r.HandleSync(ctx, query.Message)
		return
	}
	if data == "act_allusers" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		r.HandleAllUsers(ctx, query.Message)
		return
	}
	if data == "act_batchstats" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		r.HandleBatchStats(ctx, query.Message)
		return
	}
	if data == "act_addbatch" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		r.HandleAddBatch(ctx, query.Message)
		return
	}
	if data == "act_delcat" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		r.HandleDelCat(ctx, query.Message)
		return
	}
	if data == "act_broadcast" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		r.HandleBroadcast(ctx, query.Message)
		return
	}
	if data == "act_post" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		r.HandlePost(ctx, query.Message)
		return
	}
}

func getToggleText(isOn bool) string {
	if isOn {
		return "🔴 ON"
	}
	return "🟢 OFF"
}

func getLockText(isLocked bool) string {
	if isLocked {
		return "🔒 LOCKED"
	}
	return "🔓 OPEN"
}
