package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/models"
	"github.com/itsh4ro5/botupdate/internal/mtproto"
	"github.com/itsh4ro5/botupdate/internal/services"
	"github.com/itsh4ro5/botupdate/internal/telegram"
)

// Router handles incoming Telegram updates and dispatches them
type Router struct {
	bot                  *tgbotapi.BotAPI
	api                  *telegram.APIClient
	store                database.Store
	auth                 *AuthService
	support              *services.SupportService
	membership           *MembershipService
	mtproto              *mtproto.Service
	batch                *services.BatchService
	adminWizard          map[int64]*WizardState
	wizardMutex          sync.RWMutex
	scheduler            *services.Scheduler
	batchUpdateChannelID int64
}

type WizardState struct {
	Step      string
	Category  string
	Type      string
	BatchID   int64
	BatchName string
	Target    int64
	Source    int64
	Dest      string
	StartID   int
	EndID     int
	TopicKW   string
	RemoveKW  string
	ChannelID int64
	GroupID   int64
	MessageID int
}

func NewRouter(bot *tgbotapi.BotAPI, api *telegram.APIClient, store database.Store, ownerID int64, supportGroupID int64, mandatoryChannelID int64, mtprotoService *mtproto.Service, scheduler *services.Scheduler, batchUpdateChannelID int64) *Router {
	authService := NewAuthService(store, ownerID)
	supportService := services.NewSupportService(bot, api, store, supportGroupID)
	return &Router{
		bot:                  bot,
		api:                  api,
		store:                store,
		auth:                 authService,
		support:              supportService,
		membership:           NewMembershipService(bot, authService, store, mandatoryChannelID),
		mtproto:              mtprotoService,
		batch:                services.NewBatchService(store, bot, api, supportService),
		adminWizard:          make(map[int64]*WizardState),
		scheduler:            scheduler,
		batchUpdateChannelID: batchUpdateChannelID,
	}
}

func (r *Router) HandleUpdate(ctx context.Context, update tgbotapi.Update) {
	// Blocked User Middleware
	var fromID int64
	if update.Message != nil && update.Message.From != nil {
		fromID = update.Message.From.ID
	} else if update.CallbackQuery != nil && update.CallbackQuery.From != nil {
		fromID = update.CallbackQuery.From.ID
	}

	if fromID != 0 {
		state, err := r.store.Load(ctx)
		if err == nil {
			if _, blocked := state.BlockedUsers[fromID]; blocked {
				// Blocked user! Silently drop the update.
				if update.CallbackQuery != nil {
					// Answer callback so it doesn't hang forever
					r.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "❌ You are blocked from using this bot."))
				}
				return
			}
		}
	}

	if update.Message != nil {
		r.handleMessage(ctx, update.Message)
	} else if update.EditedMessage != nil {
		r.handleEditedMessage(ctx, update.EditedMessage)
	} else if update.CallbackQuery != nil {
		r.handleCallback(ctx, update.CallbackQuery)
	} else if update.ChatJoinRequest != nil {
		r.handleChatJoinRequest(ctx, update.ChatJoinRequest)
	} else if update.ChatMember != nil {
		r.handleChatMember(ctx, update.ChatMember)
	} else if update.MyChatMember != nil {
		r.handleMyChatMember(ctx, update.MyChatMember)
	}
}

func (r *Router) handleChatJoinRequest(ctx context.Context, req *tgbotapi.ChatJoinRequest) {
	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}

	if _, blocked := state.BlockedUsers[req.From.ID]; blocked {
		_ = r.api.DeclineChatJoinRequest(req.Chat.ID, req.From.ID)
		return
	}

	// Check if this is a Free Batch
	if _, ok := state.FreeBatches[req.Chat.ID]; ok {
		// Verify mandatory membership first
		isMember, _ := r.membership.CheckMembership(ctx, req.From.ID)
		if isMember {
			_ = r.api.ApproveChatJoinRequest(req.Chat.ID, req.From.ID)
			welcomeStr := "  **Approved!**\nWelcome to " + req.Chat.Title
			if cw, ok := state.CustomWelcomes[req.Chat.ID]; ok && cw != "" {
				welcomeStr = cw
			}
			msg := tgbotapi.NewMessage(req.From.ID, welcomeStr)
			msg.ParseMode = "Markdown"
			sentMsg, _ := r.bot.Send(msg)

			// Schedule deletion
			_ = r.store.AddScheduledDelete(ctx, &models.ScheduledDelete{
				ChatID:    sentMsg.Chat.ID,
				MessageID: sentMsg.MessageID,
				DeleteAt:  time.Now().Add(60 * time.Second),
			})
		} else {
			msg := tgbotapi.NewMessage(req.From.ID, "  **Declined!**\nJoin Main Channel first!")
			msg.ParseMode = "Markdown"
			r.bot.Send(msg)
			_ = r.api.DeclineChatJoinRequest(req.Chat.ID, req.From.ID)
		}
	} else if _, ok := state.PaidBatches[req.Chat.ID]; ok {
		// Paid batches are handled by /per or /demo
		if req.InviteLink != nil {
			if _, ok := state.LinkMap[req.InviteLink.InviteLink]; ok {
				// We don't auto-approve. We wait for /per or /demo.
				// In python, it revoked the link, but since we want the admin to click it from the support topic... wait!
				// In Python:
				// elif chat.id in DB["PAID_CHANNELS"]:
				//     if req.invite_link and req.invite_link.invite_link in DB["LINK_MAP"]:
				//         await client.revoke_chat_invite_link(...)
				// We can just let it sit pending.
			}
		}
	}
}

func (r *Router) handleChatMember(ctx context.Context, update *tgbotapi.ChatMemberUpdated) {
	if update.Chat.ID == r.scheduler.GetMandatoryChannelID() {
		status := update.NewChatMember.Status
		// ONLY explicitly left, kicked, or banned users are universally kicked
		// "restricted" users might still be members (e.g. muted), so they are spared.
		if status == "left" || status == "kicked" || status == "banned" {
			state, err := r.store.Load(ctx)
			if err == nil {
				targetUserID := update.NewChatMember.User.ID
				r.scheduler.UniversalKick(ctx, targetUserID, state)
			}
		}
	}
}

func (r *Router) handleMyChatMember(ctx context.Context, update *tgbotapi.ChatMemberUpdated) {
	if update.Chat.Type == "private" {
		status := update.NewChatMember.Status
		if status == "kicked" || status == "banned" {
			state, err := r.store.Load(ctx)
			if err == nil {
				targetUserID := update.NewChatMember.User.ID
				r.scheduler.UniversalKick(ctx, targetUserID, state)
			}
		}
	}
}

func (r *Router) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	r.wizardMutex.RLock()
	wizardState, exists := r.adminWizard[msg.From.ID]
	r.wizardMutex.RUnlock()

	if exists && wizardState.Step == "call_cmd_userbototp" {
		msg.Text = "/userbototp " + msg.Text
		msg.Entities = []tgbotapi.MessageEntity{
			{
				Type:   "bot_command",
				Offset: 0,
				Length: len("/userbototp"),
			},
		}
		r.HandleUserbotOTP(ctx, msg)
		return
	}

	if exists && wizardState.Step == "call_cmd_userbotpass" {
		msg.Text = "/userbotpass " + msg.Text
		msg.Entities = []tgbotapi.MessageEntity{
			{
				Type:   "bot_command",
				Offset: 0,
				Length: len("/userbotpass"),
			},
		}
		r.HandleUserbotPass(ctx, msg)
		return
	}

	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			r.HandleStart(ctx, msg)
		case "id":
			r.HandleID(ctx, msg)
		case "ping":
			r.HandlePing(ctx, msg)
		case "myinfo":
			r.HandleMyInfo(ctx, msg)
		case "admin":
			r.HandleAdmin(ctx, msg)
		case "addadmin":
			r.HandleAddAdmin(ctx, msg)
		case "deladmin":
			r.HandleRemoveAdmin(ctx, msg)
		case "ban":
			r.HandleBan(ctx, msg)
		case "unban":
			r.HandleUnban(ctx, msg)
		case "resetuser":
			r.HandleResetUser(ctx, msg)
		case "find":
			r.HandleFindUser(ctx, msg)
		case "user":
			r.HandleUserDetails(ctx, msg)
		case "allusers":
			r.HandleAllUsers(ctx, msg)
		case "extend":
			r.HandleExtend(ctx, msg)
		case "kick":
			r.HandleKick(ctx, msg)
		case "giftcoin":
			r.HandleGiftCoin(ctx, msg)
		case "perm", "per":
			r.HandleApprovePerm(ctx, msg)
		case "stats":
			r.HandleStats(ctx, msg)
		case "userbotphone":
			r.HandleUserbotPhone(ctx, msg)
		case "userbototp":
			r.HandleUserbotOTP(ctx, msg)
		case "userbotpass":
			r.HandleUserbotPass(ctx, msg)
		case "storebatch":
			r.HandleStoreBatch(ctx, msg)
		case "emptybatch":
			r.HandleEmptyBatch(ctx, msg)
		case "joinall":
			r.HandleJoinAll(ctx, msg)
		case "addbatch":
			r.HandleAddBatch(ctx, msg)
		case "delbatch":
			r.HandleDelBatch(ctx, msg)
		case "addcat":
			r.HandleAddCat(ctx, msg)
		case "setcat":
			r.HandleSetCat(ctx, msg)
		case "delcat":
			r.HandleDelCat(ctx, msg)
		case "setvipmaterials":
			r.HandleSetVIPMaterials(ctx, msg)
		case "setvipsticker":
			r.HandleSetVIPSticker(ctx, msg)
		case "batches":
			r.HandleBatches(ctx, msg)
		case "clear":
			r.HandleClear(ctx, msg)
		case "lockdown":
			r.HandleLockdown(ctx, msg)
		case "lockfree":
			r.HandleLockFree(ctx, msg)
		case "lockpaid":
			r.HandleLockPaid(ctx, msg)
		case "locktestbot":
			r.HandleLockTestBot(ctx, msg)
		case "maintenance":
			r.HandleMaintenance(ctx, msg)
		case "settestbot":
			r.HandleSetTestBot(ctx, msg)
		case "sync":
			r.HandleSync(ctx, msg)
		case "broadcast":
			r.HandleBroadcast(ctx, msg)
		case "post":
			r.HandlePost(ctx, msg)
		case "cancel":
			r.HandleCancel(ctx, msg)
		case "backup":
			r.HandleBackup(ctx, msg)
		case "gendemo":
			r.HandleGenDemo(ctx, msg)
		case "setwelcome":
			r.HandleSetWelcome(ctx, msg)
		case "del":
			r.HandleDelMessage(ctx, msg)
		case "batchstats":
			r.HandleBatchStats(ctx, msg)
		case "demo":
			r.HandleDemo(ctx, msg)

		default:
			log.Printf("Unknown command: %s", msg.Command())
		}
		return
	}

	r.wizardMutex.RLock()
	wizardState, hasWizard := r.adminWizard[msg.From.ID]
	r.wizardMutex.RUnlock()
	if hasWizard {
		if r.handleWizardMessage(ctx, msg, wizardState) {
			return
		}
	}

	// Support system forwarding logic
	if msg.Chat.IsPrivate() {
		r.handlePrivateMessage(ctx, msg)
	} else if msg.Chat.ID == r.support.GetSupportGroupID() {
		r.handleSupportReply(ctx, msg)
	}
}

func (r *Router) handleWizardMessage(ctx context.Context, msg *tgbotapi.Message, state *WizardState) bool {
	uid := msg.From.ID
	if state.Step == "ask_id" {
		cid, err := strconv.ParseInt(strings.TrimSpace(msg.Text), 10, 64)
		if err != nil {
			r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "  Error: Ensure valid ID."))
			return true
		}

		// fetch title
		cname := fmt.Sprintf("Batch %d", cid)
		chat, err := r.bot.GetChat(tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: cid}})
		if err == nil && chat.Title != "" {
			cname = chat.Title
		}

		newBatch := &models.Batch{
			ID:       cid,
			Name:     cname,
			Category: state.Category,
			Type:     state.Type,
		}

		if state.Type == "special" {
			r.wizardMutex.Lock()
			state.Step = "ask_coin"
			state.BatchID = cid
			state.BatchName = cname
			r.wizardMutex.Unlock()

			r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("  **Step 4:** '%s' ko unlock karne ke liye kitne Coins chahiye?\nSirf number bhejein (e.g. `3`):", cname)))
			return true
		}

		_ = r.store.SetBatch(ctx, cid, newBatch, state.Type)

		msgTxt := fmt.Sprintf("✅ **Added!**\n📛 Name: %s (%d)\n🏷️ Type: %s\n📂 Category: %s", cname, cid, strings.ToUpper(state.Type), state.Category)
		reply := tgbotapi.NewMessage(msg.Chat.ID, msgTxt)
		reply.ParseMode = "Markdown"
		r.bot.Send(reply)

		r.sendBatchUpdatePost(ctx, cid, cname, state.Type, state.Category, 0)

		r.wizardMutex.Lock()
		delete(r.adminWizard, uid)
		r.wizardMutex.Unlock()
		return true
	} else if state.Step == "ask_coin" {
		coinCost, err := strconv.ParseInt(strings.TrimSpace(msg.Text), 10, 64)
		if err != nil {
			r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "  Error: Please enter a valid number."))
			return true
		}

		newBatch := &models.Batch{
			ID:       state.BatchID,
			Name:     state.BatchName,
			Category: state.Category,
			Type:     state.Type,
		}

		_ = r.store.SetBatch(ctx, state.BatchID, newBatch, state.Type)
		_ = r.store.SetBatchCoin(ctx, state.BatchID, coinCost)

		msgTxt := fmt.Sprintf("✅ **Added Special Batch!**\n📛 Name: %s (%d)\n🏷️ Type: %s\n📂 Category: %s\n💰 Cost: %d Coins", state.BatchName, state.BatchID, strings.ToUpper(state.Type), state.Category, coinCost)
		reply := tgbotapi.NewMessage(msg.Chat.ID, msgTxt)
		reply.ParseMode = "Markdown"
		r.bot.Send(reply)

		r.sendBatchUpdatePost(ctx, state.BatchID, state.BatchName, state.Type, state.Category, coinCost)

		r.wizardMutex.Lock()
		delete(r.adminWizard, uid)
		r.wizardMutex.Unlock()
		return true
	} else if state.Step == "wait_msg" {
		state.MessageID = msg.MessageID
		state.Step = "confirm"

		var kb [][]tgbotapi.InlineKeyboardButton
		kb = append(kb, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ YES", "bc_yes"),
			tgbotapi.NewInlineKeyboardButtonData("❌ NO", "bc_no"),
		))

		reply := tgbotapi.NewMessage(msg.Chat.ID, "❓ **Confirm?**")
		reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(kb...)
		reply.ParseMode = "Markdown"
		r.bot.Send(reply)

		return true
	} else if strings.HasPrefix(state.Step, "call_cmd_") {
		cmdName := strings.TrimPrefix(state.Step, "call_cmd_")

		log.Printf("WIZARD_MESSAGE\nuser=%d\nstate=%s\nhandler=%s", uid, state.Step, cmdName)

		// Temporarily modify the message text to simulate a command so our Handlers work exactly the same
		msg.Text = "/" + cmdName + " " + msg.Text
		msg.Entities = []tgbotapi.MessageEntity{
			{
				Type:   "bot_command",
				Offset: 0,
				Length: len("/" + cmdName),
			},
		}

		switch cmdName {
		case "addadmin":
			r.HandleAddAdmin(ctx, msg)
		case "deladmin":
			r.HandleRemoveAdmin(ctx, msg)
		case "ban":
			r.HandleBan(ctx, msg)
		case "unban":
			r.HandleUnban(ctx, msg)
		case "kick":
			r.HandleKick(ctx, msg)
		case "giftcoin":
			r.HandleGiftCoin(ctx, msg)
		case "find":
			r.HandleFindUser(ctx, msg)
		case "resetuser":
			r.HandleResetUser(ctx, msg)
		case "demo":
			r.HandleDemo(ctx, msg)
		case "perm", "per":
			r.HandleApprovePerm(ctx, msg)
		case "extend":
			r.HandleExtend(ctx, msg)
		case "gendemo":
			r.HandleGenDemo(ctx, msg)
		case "settestbot":
			r.HandleSetTestBot(ctx, msg)
		case "setwelcome":
			r.HandleSetWelcome(ctx, msg)
		case "delbatch":
			r.HandleDelBatch(ctx, msg)
		case "addcat":
			r.HandleAddCat(ctx, msg)
		case "setcat":
			r.HandleSetCat(ctx, msg)
		case "emptybatch":
			r.HandleEmptyBatch(ctx, msg)
		case "advcap":
			r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🚧 Advanced Caption is currently under development in the Go migration."))
		case "cleanbatch":
			r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🚧 Clean Batch is currently under development in the Go migration."))
		case "storebatch":
			r.HandleStoreBatch(ctx, msg)
		case "updatepost":
			r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🚧 Update Post is currently under development in the Go migration."))
		case "superfwd":
			r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🚧 Super Forwarder is currently under development in the Go migration."))
		case "userbotphone":
			r.HandleUserbotPhone(ctx, msg)
		case "userbototp":
			r.HandleUserbotOTP(ctx, msg)
		case "userbotpass":
			r.HandleUserbotPass(ctx, msg)
		case "userlookup":
			r.HandleUserDetails(ctx, msg)
		case "deluser":
			r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🚧 Del User is currently under development in the Go migration."))
		default:
			r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Command mapping not found."))
		}

		r.wizardMutex.Lock()
		delete(r.adminWizard, uid)
		r.wizardMutex.Unlock()
		return true
	}

	return false
}

func (r *Router) handleCallback(ctx context.Context, query *tgbotapi.CallbackQuery) {
	data := query.Data
	msgID := 0
	chatID := int64(0)
	if query.Message != nil {
		msgID = query.Message.MessageID
		chatID = query.Message.Chat.ID
	}

	log.Printf("CALLBACK_RECEIVED:\nuser=%d\nchat=%d\nmessage=%d\ndata=%s", query.From.ID, chatID, msgID, data)

	// Fallback answer to prevent Telegram client from hanging if a handler panics or forgets to answer
	defer func() {
		// We send an empty callback. If a handler already answered, Telegram will just return an error and ignore this.
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
	}()

	log.Printf("CALLBACK_MATCH:\ndata=%s\nhandler=routing", data)

	if strings.HasPrefix(data, "batch_") {
		// handle batch callback
	} else if strings.HasPrefix(data, "admin_") {
		// handle admin callback
	} else if data == "accept_tnc" {
		r.handleAcceptTnC(ctx, query)
		return
	} else if data == "verify" {
		r.handleVerify(ctx, query)
		return
	} else if data == "u_main" {
		state, err := r.store.Load(ctx)
		if err == nil {
			if user, ok := state.Users[query.From.ID]; ok {
				r.showHomeMenu(query.Message.Chat.ID, query.Message.MessageID, user)
			}
		}
		return
	} else if data == "clear_pending_batch" {
		state, err := r.store.Load(ctx)
		if err == nil {
			if user, ok := state.Users[query.From.ID]; ok {
				user.PendingBatch = ""
				_ = r.store.SetUser(ctx, query.From.ID, user)
				r.showHomeMenu(query.Message.Chat.ID, query.Message.MessageID, user)
			}
		}
		return
	} else if strings.HasPrefix(data, "open_batch_") {
		state, err := r.store.Load(ctx)
		if err == nil {
			if user, ok := state.Users[query.From.ID]; ok {
				user.PendingBatch = ""
				_ = r.store.SetUser(ctx, query.From.ID, user)
			}
		}
		cidStr := strings.TrimPrefix(data, "open_batch_")
		cid, _ := strconv.ParseInt(cidStr, 10, 64)
		if _, ok := state.FreeBatches[cid]; ok {
			query.Data = fmt.Sprintf("get_f_%d", cid)
		} else if _, ok := state.PaidBatches[cid]; ok {
			query.Data = fmt.Sprintf("view_p_%d", cid)
		} else {
			query.Data = fmt.Sprintf("view_s_%d", cid)
		}
		r.batch.HandleCallback(ctx, query)
		return
	} else if strings.HasPrefix(data, "my_batches_") {
		page, _ := strconv.Atoi(strings.TrimPrefix(data, "my_batches_"))
		r.handleMyBatches(ctx, query, page)
		return
	} else if strings.HasPrefix(data, "all_batches_") {
		r.handleAllBatches(ctx, query)
		return
	} else if strings.HasPrefix(data, "showcat_") {
		idx, _ := strconv.Atoi(strings.TrimPrefix(data, "showcat_"))
		r.handleShowCat(ctx, query, idx)
		return
	} else if strings.HasPrefix(data, "listcat_") {
		parts := strings.Split(strings.TrimPrefix(data, "listcat_"), "_")
		if len(parts) == 3 {
			idx, _ := strconv.Atoi(parts[0])
			bType := parts[1]
			page, _ := strconv.Atoi(parts[2])
			r.handleListCat(ctx, query, idx, bType, page)
		}
		return
	} else if strings.HasPrefix(data, "get_f_") || strings.HasPrefix(data, "join_f_") || strings.HasPrefix(data, "view_p_") || strings.HasPrefix(data, "view_s_") || strings.HasPrefix(data, "unlock_all_free") || strings.HasPrefix(data, "share_btn_") || strings.HasPrefix(data, "req_access_") || strings.HasPrefix(data, "unlock_s_") {
		r.batch.HandleCallback(ctx, query)
		return
	} else if data == "menu_refer" {
		r.handleMenuRefer(ctx, query)
		return
	} else if data == "my_info" {
		r.handleMyInfo(ctx, query)
		return
	} else if data == "test_bot" {
		r.handleTestBot(ctx, query)
		return
	} else if data == "role_selector" {
		r.handleRoleSelector(ctx, query)
		return
	} else if data == "goto_owner_panel" {
		r.handleGotoOwnerPanel(ctx, query)
		return
	} else if data == "goto_admin_panel" {
		r.handleGotoAdminPanel(ctx, query)
		return
	} else if data == "goto_user_panel" {
		r.handleGotoUserPanel(ctx, query)
		return
	} else if strings.HasPrefix(data, "wcat_") || data == "wiz_free" || data == "wiz_paid" || data == "wiz_special" {
		r.handleWizardCallback(ctx, query)
		return
	} else if strings.HasPrefix(data, "setextcat_") {
		r.handleSetExtCatCallback(ctx, query)
		return
	} else if strings.HasPrefix(data, "delcat_") {
		r.handleDelCatCallback(ctx, query)
		return
	} else if data == "bc_yes" || data == "bc_no" {
		r.handleBroadcastCallback(ctx, query)
		return
	} else if strings.HasPrefix(data, "dash_") || strings.HasPrefix(data, "adash_") || strings.HasPrefix(data, "toggle_") || strings.HasPrefix(data, "userbot_") || strings.HasPrefix(data, "input_") || strings.HasPrefix(data, "act_") || data == "giftcoin_start" {
		r.handleDashboardCallback(ctx, query)
		return
	}

	callback := tgbotapi.NewCallback(query.ID, "Processing...")
	r.bot.Request(callback)
}

func (r *Router) handleBroadcastCallback(ctx context.Context, query *tgbotapi.CallbackQuery) {
	uid := query.From.ID
	r.wizardMutex.RLock()
	state, exists := r.adminWizard[uid]
	r.wizardMutex.RUnlock()

	if !exists || state.Step != "confirm" {
		r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Expired"))
		return
	}

	if query.Data == "bc_no" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		edit := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, "  Cancelled")
		r.bot.Send(edit)
		r.wizardMutex.Lock()
		delete(r.adminWizard, uid)
		r.wizardMutex.Unlock()
		return
	}

	if query.Data == "bc_yes" {
		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))
		edit := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, "⏳ **Processing Broadcast...**")
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)

		isUserBroadcast := state.Type == "broadcast"

		go func() {
			dbState, _ := r.store.Load(context.Background())
			var targets []int64

			if isUserBroadcast {
				for userID := range dbState.Users {
					targets = append(targets, userID)
				}
			} else {
				for batchID := range dbState.FreeBatches {
					targets = append(targets, batchID)
				}
				for batchID := range dbState.PaidBatches {
					targets = append(targets, batchID)
				}
				for batchID := range dbState.SpecialBatches {
					targets = append(targets, batchID)
				}
			}

			count := 0
			blockedCount := 0
			totalTargets := len(targets)

			for index, tid := range targets {
				copyMsg := tgbotapi.NewCopyMessage(tid, query.Message.Chat.ID, state.MessageID)
				_, err := r.bot.Send(copyMsg)
				if err != nil {
					if isUserBroadcast {
						blockedCount++
						// Only kick if explicitly blocked/deactivated, skip for temporary network errors
						errStr := err.Error()
						if strings.Contains(errStr, "Forbidden") || strings.Contains(errStr, "blocked by the user") || strings.Contains(errStr, "user is deactivated") {
							r.scheduler.UniversalKick(context.Background(), tid, dbState)
						}
					}
				} else {
					count++
				}

				time.Sleep(50 * time.Millisecond) // avoid flood wait

				if index > 0 && index%50 == 0 {
					progEdit := tgbotapi.NewEditMessageText(
						query.Message.Chat.ID,
						query.Message.MessageID,
						fmt.Sprintf("📡 **Broadcasting In Progress...**\n\n📤 Sent: `%d / %d`\n🚫 Blocked/Removed: `%d`", count, totalTargets, blockedCount),
					)
					progEdit.ParseMode = "Markdown"
					r.bot.Send(progEdit)
				}
			}

			finalEdit := tgbotapi.NewEditMessageText(
				query.Message.Chat.ID,
				query.Message.MessageID,
				fmt.Sprintf("✅ **Broadcast Completed!**\n\n📤 Successfully Sent: `%d`\n🚫 Blocked & Removed: `%d`\n\n💡 *Jin users ne bot ko block kiya tha, unhe sabhi premium/free batches se nikal diya gaya hai (Mandatory channel me abhi bhi hain).* ", count, blockedCount),
			)
			finalEdit.ParseMode = "Markdown"
			r.bot.Send(finalEdit)

			r.wizardMutex.Lock()
			delete(r.adminWizard, uid)
			r.wizardMutex.Unlock()
		}()
	}
}

func (r *Router) handleSetExtCatCallback(ctx context.Context, query *tgbotapi.CallbackQuery) {
	uid := query.From.ID
	r.wizardMutex.RLock()
	state, exists := r.adminWizard[uid]
	r.wizardMutex.RUnlock()

	if !exists || state.Step != "setcat" {
		r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Expired"))
		return
	}

	catIdxStr := strings.TrimPrefix(query.Data, "setextcat_")
	catIdx, _ := strconv.Atoi(catIdxStr)

	dbState, _ := r.store.Load(ctx)
	categories := dbState.Categories
	if len(categories) == 0 {
		categories = []string{"UPSC", "SSC", "BANKING", "RAILWAY", "DEFENCE", "STATE PSC", "TEACHING", "IIT JEE", "NEET", "GATE"}
	}

	if catIdx >= len(categories) {
		r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Invalid category"))
		return
	}

	newCat := categories[catIdx]

	ids := strings.Split(state.Dest, ",")
	count := 0
	for _, idStr := range ids {
		cid, err := strconv.ParseInt(idStr, 10, 64)
		if err == nil {
			var batch *models.Batch
			if b, ok := dbState.FreeBatches[cid]; ok {
				batch = b
			}
			if b, ok := dbState.PaidBatches[cid]; ok {
				batch = b
			}
			if b, ok := dbState.SpecialBatches[cid]; ok {
				batch = b
			}

			if batch != nil {
				batch.Category = newCat
				r.store.SetBatch(ctx, cid, batch, batch.Type)
				count++
			}
		}
	}

	r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, fmt.Sprintf("✅ %d Batches moved to %s", count, newCat)))

	edit := tgbotapi.NewEditMessageText(
		query.Message.Chat.ID,
		query.Message.MessageID,
		fmt.Sprintf("✅ **Done!** %d batches set to category: `%s`", count, newCat),
	)
	edit.ParseMode = "Markdown"
	r.bot.Send(edit)

	r.wizardMutex.Lock()
	delete(r.adminWizard, uid)
	r.wizardMutex.Unlock()
}

func (r *Router) handleDelCatCallback(ctx context.Context, query *tgbotapi.CallbackQuery) {
	if !r.auth.IsAdmin(ctx, query.From.ID) {
		r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Access Denied!"))
		return
	}

	catIdxStr := strings.TrimPrefix(query.Data, "delcat_")
	catIdx, _ := strconv.Atoi(catIdxStr)

	dbState, _ := r.store.Load(ctx)
	categories := dbState.Categories
	if len(categories) == 0 {
		categories = []string{"UPSC", "SSC", "BANKING", "RAILWAY", "DEFENCE", "STATE PSC", "TEACHING", "IIT JEE", "NEET", "GATE"}
	}

	if catIdx >= len(categories) || catIdx < 0 {
		r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Invalid category"))
		return
	}

	catName := categories[catIdx]
	if catName == "Other Batches" {
		r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Cannot delete this category"))
		return
	}

	// Remove category
	newCats := []string{}
	for i, c := range categories {
		if i != catIdx {
			newCats = append(newCats, c)
		}
	}
	r.store.SetCategories(ctx, newCats)

	// Move batches to "Other Batches"
	count := 0

	updateBatchCategory := func(batches map[int64]*models.Batch) {
		for cid, batch := range batches {
			if batch.Category == catName {
				batch.Category = "Other Batches"
				r.store.SetBatch(ctx, cid, batch, batch.Type)
				count++
			}
		}
	}

	updateBatchCategory(dbState.FreeBatches)
	updateBatchCategory(dbState.PaidBatches)
	updateBatchCategory(dbState.SpecialBatches)

	r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, fmt.Sprintf("Deleted %s", catName)))

	edit := tgbotapi.NewEditMessageText(
		query.Message.Chat.ID,
		query.Message.MessageID,
		fmt.Sprintf("✅ **Deleted!** Category `%s` removed.\n%d batches moved to 'Other Batches'.", catName, count),
	)
	edit.ParseMode = "Markdown"
	r.bot.Send(edit)
}

func (r *Router) handleWizardCallback(ctx context.Context, query *tgbotapi.CallbackQuery) {
	uid := query.From.ID
	r.wizardMutex.RLock()
	state, exists := r.adminWizard[uid]
	r.wizardMutex.RUnlock()

	if !exists {
		r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Expired"))
		return
	}

	data := query.Data
	if strings.HasPrefix(data, "wcat_") {
		catIdxStr := strings.TrimPrefix(data, "wcat_")
		catIdx, _ := strconv.Atoi(catIdxStr)

		dbState, _ := r.store.Load(ctx)
		categories := dbState.Categories
		if len(categories) == 0 {
			categories = []string{"UPSC", "SSC", "BANKING", "RAILWAY", "DEFENCE", "STATE PSC", "TEACHING", "IIT JEE", "NEET", "GATE"}
		}

		r.wizardMutex.Lock()
		state.Category = categories[catIdx]
		state.Step = "ask_type"
		r.wizardMutex.Unlock()

		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))

		kb := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🆓 Free", "wiz_free"),
				tgbotapi.NewInlineKeyboardButtonData("💵 Paid", "wiz_paid"),
				tgbotapi.NewInlineKeyboardButtonData("✨ Special", "wiz_special"),
			),
		)

		edit := tgbotapi.NewEditMessageTextAndMarkup(
			query.Message.Chat.ID,
			query.Message.MessageID,
			fmt.Sprintf("  Category: **%s**\n\n  **Step 2:** Select Batch Type:", state.Category),
			kb,
		)
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)
		return
	}

	if data == "wiz_free" || data == "wiz_paid" || data == "wiz_special" {
		if state.Category == "" {
			r.bot.Request(tgbotapi.NewCallbackWithAlert(query.ID, "Start again"))
			return
		}

		r.wizardMutex.Lock()
		state.Type = strings.TrimPrefix(data, "wiz_")
		state.Step = "ask_id"
		r.wizardMutex.Unlock()

		r.bot.Request(tgbotapi.NewCallback(query.ID, ""))

		edit := tgbotapi.NewEditMessageText(
			query.Message.Chat.ID,
			query.Message.MessageID,
			fmt.Sprintf("  **Step 3:** Send **Channel ID** for %s:", strings.ToUpper(state.Type)),
		)
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)
		return
	}
}

func (r *Router) sendBatchUpdatePost(ctx context.Context, cid int64, cname, btype, category string, cost int64) {
	if r.batchUpdateChannelID == 0 {
		log.Println("BATCH_UPDATE_CHANNEL_ID is not configured, skipping update post")
		return
	}

	botInfo, err := r.bot.GetMe()
	if err != nil {
		log.Println("Failed to get bot info:", err)
		return
	}

	caption := fmt.Sprintf("🎉 NEW BATCH ADDED!\n\n📦 Name: %s\n🏷️ Type: %s\n📂 Category: %s\n", cname, strings.ToUpper(btype), category)
	if btype == "special" && cost > 0 {
		caption += fmt.Sprintf("💰 Unlock Cost: %d Coin(s)\n", cost)
	}
	caption += "\n👉 Click the button below to get direct access via our Bot!"

	deepLink := fmt.Sprintf("https://t.me/%s?start=batch_%d", botInfo.UserName, cid)
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🚀 Get Access Here", deepLink),
		),
	)

	chat, err := r.bot.GetChat(tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: cid}})
	if err == nil && chat.Photo != nil {
		photo := tgbotapi.NewPhoto(r.batchUpdateChannelID, tgbotapi.FileID(chat.Photo.BigFileID))
		photo.Caption = caption
		photo.ReplyMarkup = kb
		if _, err := r.bot.Send(photo); err == nil {
			return
		}
	}

	msg := tgbotapi.NewMessage(r.batchUpdateChannelID, caption)
	msg.ReplyMarkup = kb
	r.bot.Send(msg)
}
