package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// HandleAddAdmin processes the /addadmin command
func (r *Router) HandleAddAdmin(ctx context.Context, msg *tgbotapi.Message) {
	// Only Owner can add admin
	if !r.auth.IsOwner(ctx, msg.From.ID) {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Only the owner can use this command."))
		return
	}

	args := msg.CommandArguments()
	if args == "" {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /addadmin <user_id>"))
		return
	}

	targetID, err := strconv.ParseInt(strings.TrimSpace(args), 10, 64)
	if err != nil {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Invalid User ID."))
		return
	}

	if err := r.store.SetAdmin(ctx, targetID, true); err == nil {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ User %d added as Admin.", targetID)))
	} else {
		log.Printf("ERROR: SetAdmin failed: %v", err)
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Failed to add admin due to database error."))
	}
}

// HandleRemoveAdmin processes the /removeadmin command
func (r *Router) HandleRemoveAdmin(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsOwner(ctx, msg.From.ID) {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Only the owner can use this command."))
		return
	}

	args := msg.CommandArguments()
	if args == "" {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /removeadmin <user_id>"))
		return
	}

	targetID, err := strconv.ParseInt(strings.TrimSpace(args), 10, 64)
	if err != nil {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Invalid User ID."))
		return
	}

	if err := r.store.SetAdmin(ctx, targetID, false); err == nil {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ User %d removed from Admins.", targetID)))
	}
}

// HandleUserbotPhone initiates MTProto login
func (r *Router) HandleUserbotPhone(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsOwner(ctx, msg.From.ID) {
		return
	}

	args := msg.CommandArguments()
	if args == "" {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /userbotphone <phone_number>"))
		return
	}

	phone := strings.ReplaceAll(args, " ", "")

	r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ OTP request bhej raha hu, kripya wait karein..."))

	err := r.mtproto.StartAuth(ctx, phone, func() {
		r.wizardMutex.Lock()
		r.adminWizard[msg.From.ID] = &WizardState{Step: "call_cmd_userbotpass"}
		r.wizardMutex.Unlock()
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "🔑 **Please enter your 2FA password.**\n(Bina command ke direct password bhejein)"))
	})
	if err != nil {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Error starting auth: %v", err)))
		return
	}

	r.wizardMutex.Lock()
	r.adminWizard[msg.From.ID] = &WizardState{Step: "call_cmd_userbototp"}
	r.wizardMutex.Unlock()

	r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ **OTP Bhej diya gaya hai!**\n\nKripya apna OTP type karein (Direct number bhejein, jaise 12345)"))
}

func (r *Router) HandleUserbotOTP(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsOwner(ctx, msg.From.ID) {
		return
	}

	otp := strings.ReplaceAll(msg.Text, " ", "")
	if strings.HasPrefix(otp, "/userbototp") {
		otp = strings.ReplaceAll(msg.CommandArguments(), " ", "")
	}
	otp = strings.ReplaceAll(otp, "-", "")

	r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ OTP Verify kar raha hu..."))
	err := r.mtproto.SubmitOTP(otp)
	if err != nil {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ OTP Error: %v", err)))
	} else {
		r.wizardMutex.Lock()
		delete(r.adminWizard, msg.From.ID)
		r.wizardMutex.Unlock()
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ OTP submitted. (Aap log in ho gaye hain ya 2FA password prompt ka wait karein)"))
	}
}

func (r *Router) HandleUserbotPass(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsOwner(ctx, msg.From.ID) {
		return
	}

	pass := strings.TrimSpace(msg.Text)
	if strings.HasPrefix(pass, "/userbotpass") {
		pass = strings.TrimSpace(msg.CommandArguments())
	}
	r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Password check kar raha hu..."))
	err := r.mtproto.SubmitPassword(pass)
	if err != nil {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Password Error: %v", err)))
	} else {
		r.wizardMutex.Lock()
		delete(r.adminWizard, msg.From.ID)
		r.wizardMutex.Unlock()
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Password submitted."))
	}
}

func (r *Router) HandleStoreBatch(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsOwner(ctx, msg.From.ID) {
		return
	}

	args := msg.CommandArguments()
	chatID, err := strconv.ParseInt(strings.TrimSpace(args), 10, 64)
	if err != nil {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Error: ID numbers me honi chahiye (jaise -10012345678)."))
		return
	}

	progressMsg := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("⏳ **Starting Userbot to scan Batch `%d`...**", chatID))
	progressMsg.ParseMode = "Markdown"
	sentMsg, _ := r.bot.Send(progressMsg)

	cb := func(vid, pdf int, cName string) {
		edit := tgbotapi.NewEditMessageText(msg.Chat.ID, sentMsg.MessageID, fmt.Sprintf("⏳ **Scanning '%s' via Userbot...**\n\nFound so far:\n🎥 Videos: `%d`\n📄 PDFs: `%d`", cName, vid, pdf))
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)
	}

	go func() {
		result, err := r.mtproto.ScanBatch(context.Background(), chatID, cb)
		if err != nil {
			edit := tgbotapi.NewEditMessageText(msg.Chat.ID, sentMsg.MessageID, fmt.Sprintf("❌ **Userbot Error during scan:** `%v`", err))
			r.bot.Send(edit)
		} else {
			edit := tgbotapi.NewEditMessageText(msg.Chat.ID, sentMsg.MessageID, result)
			r.bot.Send(edit)
		}
	}()
}

func (r *Router) HandleEmptyBatch(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsOwner(ctx, msg.From.ID) {
		return
	}
	chatID, err := strconv.ParseInt(strings.TrimSpace(msg.CommandArguments()), 10, 64)
	if err != nil {
		return
	}

	r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Emptying batch..."))
	err = r.mtproto.EmptyBatch(ctx, chatID)
	if err != nil {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Error: %v", err)))
	} else {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Batch Emptied!"))
	}
}

func (r *Router) HandleJoinAll(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsOwner(ctx, msg.From.ID) {
		return
	}

	r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Auto-joining userbot..."))
	userbotID, err := r.mtproto.GetUserbotID(ctx)
	if err != nil {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Error: %v", err)))
		return
	}

	state, _ := r.store.Load(ctx)
	var allChats []int64
	for id := range state.FreeBatches {
		allChats = append(allChats, id)
	}
	for id := range state.PaidBatches {
		allChats = append(allChats, id)
	}
	for id := range state.SpecialBatches {
		allChats = append(allChats, id)
	}

	success, failed := 0, 0
	for _, cid := range allChats {
		promoteReq := tgbotapi.PromoteChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{
				ChatID: cid,
				UserID: userbotID,
			},
			CanInviteUsers: true,
			CanManageChat:  true,
		}
		_, err := r.bot.Request(promoteReq)
		if err == nil {
			success++
		} else {
			failed++
		}
		time.Sleep(500 * time.Millisecond)
	}

	r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ **Join All Complete!**\nSuccess: `%d`\nFailed: `%d`", success, failed)))
}

// JSONConfigHack is a temporary interface cast for pulling owner ID without circular deps
type JSONConfigHack interface {
	OwnerID() int64
}

// HandleAddBatch starts the add batch wizard
func (r *Router) HandleAddBatch(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}

	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}

	categories := state.Categories
	if len(categories) == 0 {
		categories = []string{"UPSC", "SSC", "BANKING", "RAILWAY", "DEFENCE", "STATE PSC", "TEACHING", "IIT JEE", "NEET", "GATE"}
	}

	r.wizardMutex.Lock()
	r.adminWizard[msg.From.ID] = &WizardState{Step: "ask_cat"}
	r.wizardMutex.Unlock()

	var kb [][]tgbotapi.InlineKeyboardButton
	for i := 0; i < len(categories); i += 2 {
		var row []tgbotapi.InlineKeyboardButton
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("📁 %s", categories[i]), fmt.Sprintf("wcat_%d", i)))
		if i+1 < len(categories) {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("📁 %s", categories[i+1]), fmt.Sprintf("wcat_%d", i+1)))
		}
		kb = append(kb, row)
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, "  **Add Batch Wizard**\nSelect Category:")
	reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(kb...)
	reply.ParseMode = "Markdown"
	r.bot.Send(reply)
}

// HandleDelBatch processes the /delbatch command
func (r *Router) HandleDelBatch(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}

	args := msg.CommandArguments()
	if args == "" {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /delbatch <batch_id>"))
		return
	}

	batchID, err := strconv.ParseInt(strings.TrimSpace(args), 10, 64)
	if err != nil {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Invalid Batch ID."))
		return
	}

	_ = r.store.RemoveBatch(ctx, batchID)
	r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "  Batch poori tarah database se Delete ho gaya."))
}

// HandleAddCat processes the /addcat command
func (r *Router) HandleAddCat(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}

	args := msg.CommandArguments()

	log.Printf("ADD_CATEGORY_INPUT\nuser=%d\ntext=%s", msg.From.ID, args)

	if args == "" {
		return
	}
	newCat := strings.TrimSpace(args)

	state, _ := r.store.Load(ctx)
	found := false
	for _, c := range state.Categories {
		if c == newCat {
			found = true
			break
		}
	}
	if !found {
		log.Printf("CATEGORY_MUTATION\nuser=%d\ncategory=%s\naction=create", msg.From.ID, newCat)
		state.Categories = append(state.Categories, newCat)
		err := r.store.SetCategories(ctx, state.Categories)
		status := "success"
		if err != nil {
			status = "error"
		}
		log.Printf("CATEGORY_PERSIST\ncategory=%s\nstatus=%s", newCat, status)
	}
	r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("  Added Category: %s", newCat)))
	log.Printf("CATEGORY_UI_REFRESH\ncategory=%s", newCat)
}

// HandleSetCat processes the /setcat command
func (r *Router) HandleSetCat(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}

	rawText := msg.Text
	// simple regex for -?\d+
	var ids []string
	words := strings.Fields(rawText)
	for _, w := range words {
		// if it's a number, save it
		_, err := strconv.ParseInt(w, 10, 64)
		if err == nil {
			ids = append(ids, w)
		}
	}

	if len(ids) == 0 {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "  Error: Koi valid ID nahi mili."))
		return
	}

	r.wizardMutex.Lock()
	r.adminWizard[msg.From.ID] = &WizardState{
		Step: "setcat",
		Dest: strings.Join(ids, ","),
	}
	r.wizardMutex.Unlock()

	state, _ := r.store.Load(ctx)
	categories := state.Categories
	if len(categories) == 0 {
		categories = []string{"UPSC", "SSC", "BANKING", "RAILWAY", "DEFENCE", "STATE PSC", "TEACHING", "IIT JEE", "NEET", "GATE"}
	}

	var kb [][]tgbotapi.InlineKeyboardButton
	for i, c := range categories {
		kb = append(kb, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("📁 %s", c), fmt.Sprintf("setextcat_%d", i))))
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("  **%d Batches** detect hue hain.\nIn sabhi ke liye nayi category select karein:", len(ids)))
	reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(kb...)
	reply.ParseMode = "Markdown"
	r.bot.Send(reply)
}

// HandleDelCat processes the /delcat command
func (r *Router) HandleDelCat(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}

	state, _ := r.store.Load(ctx)
	categories := state.Categories
	if len(categories) == 0 {
		categories = []string{"UPSC", "SSC", "BANKING", "RAILWAY", "DEFENCE", "STATE PSC", "TEACHING", "IIT JEE", "NEET", "GATE"}
	}

	var kb [][]tgbotapi.InlineKeyboardButton
	for i, c := range categories {
		if c != "Other Batches" {
			kb = append(kb, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🗑️ Delete: %s", c), fmt.Sprintf("delcat_%d", i))))
		}
	}
	kb = append(kb, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", "u_main")))

	reply := tgbotapi.NewMessage(msg.Chat.ID, "  **Delete Category:**")
	reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(kb...)
	reply.ParseMode = "Markdown"
	r.bot.Send(reply)
}

func (r *Router) HandleSetVIPMaterials(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	args := msg.CommandArguments()
	if args == "" {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /setvipmaterials <link>"))
		return
	}

	state, _ := r.store.Load(ctx)
	state.VIPMaterialsLink = strings.TrimSpace(args)
	r.store.Save(ctx, state)

	r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "  👑 VIP Materials link updated."))
}

func (r *Router) HandleSetVIPSticker(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}

	if msg.ReplyToMessage == nil || (msg.ReplyToMessage.Sticker == nil && msg.ReplyToMessage.Animation == nil) {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Kisi sticker ya GIF ko reply karke /setvipsticker bhejein."))
		return
	}

	var fileID string
	var fileType string
	if msg.ReplyToMessage.Sticker != nil {
		fileID = msg.ReplyToMessage.Sticker.FileID
		fileType = "sticker"
	} else if msg.ReplyToMessage.Animation != nil {
		fileID = msg.ReplyToMessage.Animation.FileID
		fileType = "animation"
	}

	state, _ := r.store.Load(ctx)
	state.VIPStickerID = fileID
	state.VIPStickerType = fileType
	r.store.Save(ctx, state)

	r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "  👑 VIP treat sticker/GIF set ho gaya."))
}

func (r *Router) HandleBatches(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}

	state, _ := r.store.Load(ctx)

	var lines []string
	lines = append(lines, "ALL BATCHES\n==============================")

	for cid, name := range state.AllChats {
		lines = append(lines, fmt.Sprintf("%d | %s", cid, name))
	}

	if len(lines) == 1 {
		lines = append(lines, "No batches found.")
	}

	content := strings.Join(lines, "\n")

	fileBytes := tgbotapi.FileBytes{
		Name:  "batches.txt",
		Bytes: []byte(content),
	}

	doc := tgbotapi.NewDocument(msg.Chat.ID, fileBytes)
	r.bot.Send(doc)
}

func (r *Router) HandleClear(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsOwner(ctx, msg.From.ID) {
		return
	}

	progressMsg := tgbotapi.NewMessage(msg.Chat.ID, "  **Super Exit /clear Start...**")
	progressMsg.ParseMode = "Markdown"
	sentMsg, err := r.bot.Send(progressMsg)
	if err != nil {
		return
	}

	cb := func(text string) {
		edit := tgbotapi.NewEditMessageText(msg.Chat.ID, sentMsg.MessageID, text)
		edit.ParseMode = "Markdown"
		r.bot.Send(edit)
	}

	go func() {
		err := r.mtproto.ClearAll(context.Background(), cb)
		if err != nil {
			cb(fmt.Sprintf("❌ **Userbot Error:** `%v`", err))
		}
	}()
}

// HandleLockdown processes the /lockdown command
func (r *Router) HandleLockdown(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	state, _ := r.store.Load(ctx)
	state.NewUsersAllowed = !state.NewUsersAllowed
	r.store.Save(ctx, state)

	msgTxt := "  **Lockdown Enabled!**"
	if state.NewUsersAllowed {
		msgTxt = "  **Lockdown Lifted!**"
	}
	reply := tgbotapi.NewMessage(msg.Chat.ID, msgTxt)
	reply.ParseMode = "Markdown"
	r.bot.Send(reply)
}

// HandleLockFree processes the /lockfree command
func (r *Router) HandleLockFree(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	state, _ := r.store.Load(ctx)
	state.FreeLocked = !state.FreeLocked
	r.store.Save(ctx, state)

	msgTxt := "Free Batches **UNLOCKED  **."
	if state.FreeLocked {
		msgTxt = "Free Batches **LOCKED  **."
	}
	reply := tgbotapi.NewMessage(msg.Chat.ID, msgTxt)
	reply.ParseMode = "Markdown"
	r.bot.Send(reply)
}

// HandleLockPaid processes the /lockpaid command
func (r *Router) HandleLockPaid(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	state, _ := r.store.Load(ctx)
	state.PaidLocked = !state.PaidLocked
	r.store.Save(ctx, state)

	msgTxt := "Paid Batches **UNLOCKED  **."
	if state.PaidLocked {
		msgTxt = "Paid Batches **LOCKED  **."
	}
	reply := tgbotapi.NewMessage(msg.Chat.ID, msgTxt)
	reply.ParseMode = "Markdown"
	r.bot.Send(reply)
}

// HandleLockTestBot processes the /locktestbot command
func (r *Router) HandleLockTestBot(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	state, _ := r.store.Load(ctx)
	state.TestBotLocked = !state.TestBotLocked
	r.store.Save(ctx, state)

	msgTxt := "Test Bot **UNLOCKED  **."
	if state.TestBotLocked {
		msgTxt = "Test Bot **LOCKED  **."
	}
	reply := tgbotapi.NewMessage(msg.Chat.ID, msgTxt)
	reply.ParseMode = "Markdown"
	r.bot.Send(reply)
}

// HandleMaintenance processes the /maintenance command
func (r *Router) HandleMaintenance(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	state, _ := r.store.Load(ctx)
	state.MaintenanceMode = !state.MaintenanceMode
	r.store.Save(ctx, state)

	msgTxt := "  **Maintenance Mode Disabled!**\nBot ab normally kaam kar raha hai."
	if state.MaintenanceMode {
		msgTxt = "  **Maintenance Mode Enabled!**\nNormal users ka support message ab aana band ho gaya hai."
	}
	reply := tgbotapi.NewMessage(msg.Chat.ID, msgTxt)
	reply.ParseMode = "Markdown"
	r.bot.Send(reply)
}

// HandleSetTestBot processes the /settestbot command
func (r *Router) HandleSetTestBot(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	args := msg.CommandArguments()
	if args == "" {
		return
	}

	state, _ := r.store.Load(ctx)
	state.TestBotLink = strings.TrimSpace(args)
	r.store.Save(ctx, state)

	r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "  Test bot link updated."))
}

// HandleSync processes the /sync command
func (r *Router) HandleSync(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}

	progressMsg := tgbotapi.NewMessage(msg.Chat.ID, "  Background sync started manually.")
	sentMsg, err := r.bot.Send(progressMsg)
	if err == nil {
		r.scheduler.RunSync(ctx, func(text string) {
			edit := tgbotapi.NewEditMessageText(msg.Chat.ID, sentMsg.MessageID, text)
			edit.ParseMode = "Markdown"
			r.bot.Send(edit)
		})
	}
}

// HandleBroadcast processes the /broadcast command
func (r *Router) HandleBroadcast(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsOwner(ctx, msg.From.ID) { // Python uses owner check for broadcast? Let's assume yes, or admin. Let's use IsAdmin.
		return
	}

	r.wizardMutex.Lock()
	r.adminWizard[msg.From.ID] = &WizardState{
		Step: "wait_msg",
		Type: "broadcast",
	}
	r.wizardMutex.Unlock()

	r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "  Send message to broadcast."))
}

// HandlePost processes the /post command
func (r *Router) HandlePost(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}

	r.wizardMutex.Lock()
	r.adminWizard[msg.From.ID] = &WizardState{
		Step: "wait_msg",
		Type: "post",
	}
	r.wizardMutex.Unlock()

	msgOut := tgbotapi.NewMessage(msg.Chat.ID, "Send the message you want to post. (Supports formatting, images, buttons).")
	r.bot.Send(msgOut)
}

// HandleAdminList handles showing the list of admins
func (r *Router) HandleAdminList(ctx context.Context, query *tgbotapi.CallbackQuery) {
	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}

	msgText := "📋 *Current Administrators*\n\n"
	if len(state.AdminIDs) == 0 {
		msgText += "No administrators found."
	} else {
		for id := range state.AdminIDs {
			msgText += fmt.Sprintf("• `%d`\n", id)
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔙 Back to Staff", "dash_staff")),
	)

	edit := tgbotapi.NewEditMessageTextAndMarkup(query.Message.Chat.ID, query.Message.MessageID, msgText, keyboard)
	edit.ParseMode = "Markdown"
	r.bot.Send(edit)
}

// HandleCancel processes the /cancel command to cancel admin operations
func (r *Router) HandleCancel(ctx context.Context, msg *tgbotapi.Message) {
	r.wizardMutex.Lock()
	delete(r.adminWizard, msg.From.ID)
	r.wizardMutex.Unlock()
	r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Operation cancelled."))
}

// HandleGiftCoin processes the /giftcoin command
func (r *Router) HandleGiftCoin(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsOwner(ctx, msg.From.ID) {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Only the owner can use this command."))
		return
	}

	args := msg.CommandArguments()
	if args == "" {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /giftcoin <user_id> <amount>"))
		return
	}

	parts := strings.Fields(args)
	if len(parts) != 2 {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /giftcoin <user_id> <amount>"))
		return
	}

	targetID, err1 := strconv.ParseInt(parts[0], 10, 64)
	amount, err2 := strconv.ParseInt(parts[1], 10, 64)

	if err1 != nil || err2 != nil || amount < 1 {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ Invalid input. User ID and Amount must be valid numbers, and amount must be >= 1."))
		return
	}

	state, err := r.store.Load(ctx)
	if err != nil {
		return
	}

	user, ok := state.Users[targetID]
	if !ok {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ User `%d` database me nahi mila.", targetID)))
		return
	}

	user.ReferralCount += int(amount)
	_ = r.store.SetUser(ctx, targetID, user)

	msgTxt := fmt.Sprintf("✅ **Gift Sent!**\nUser `%d` ko **%d Coin(s)** gift kar diye gaye.", targetID, amount)
	reply := tgbotapi.NewMessage(msg.Chat.ID, msgTxt)
	reply.ParseMode = "Markdown"
	r.bot.Send(reply)

	notifyMsg := tgbotapi.NewMessage(targetID, fmt.Sprintf("🎁 **GIFT RECEIVED!**\n\nAdmin ne aapko **%d Coin(s)** gift kiye hain! 🎉", amount))
	notifyMsg.ParseMode = "Markdown"
	r.bot.Send(notifyMsg)
}
