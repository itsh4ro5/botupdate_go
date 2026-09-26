package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itsh4ro5/botupdate/internal/models"
)

func (r *Router) isAdminMessage(ctx context.Context, msg *tgbotapi.Message) bool {
	if msg.From != nil && r.auth.IsAdmin(ctx, msg.From.ID) {
		return true
	}
	if msg.SenderChat != nil && msg.Chat.ID == r.support.GetSupportGroupID() {
		return true
	}
	return false
}

// HandleBan processes the /ban command
func (r *Router) HandleBan(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	args := msg.CommandArguments()
	if args == "" {
		return
	}
	target, err := strconv.ParseInt(strings.TrimSpace(args), 10, 64)
	if err != nil {
		return
	}

	// Can't ban owner
	if r.auth.IsOwner(ctx, target) {
		return
	}

	state, _ := r.store.Load(ctx)
	if _, blocked := state.BlockedUsers[target]; !blocked {
		// execute_universal_kick with permanent_ban=True
		r.executeUniversalKick(ctx, target, true)
		r.store.SetBlockedUser(ctx, target, true)
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("User `%d` BANNED.", target)))
	}
}

// HandleUnban processes the /unban command
func (r *Router) HandleUnban(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	args := msg.CommandArguments()
	if args == "" {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Error: Kripya ek User ID bhejein."))
		return
	}
	target, err := strconv.ParseInt(strings.TrimSpace(args), 10, 64)
	if err != nil {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Error: Kripya ek valid Numeric User ID bhejein."))
		return
	}

	state, _ := r.store.Load(ctx)
	if _, blocked := state.BlockedUsers[target]; blocked {
		r.store.SetBlockedUser(ctx, target, false)
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("User `%d` UNBANNED.", target)))
	}
}

// HandleResetUser processes the /resetuser command
func (r *Router) HandleResetUser(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	args := msg.CommandArguments()
	if args == "" {
		return
	}
	target, err := strconv.ParseInt(strings.TrimSpace(args), 10, 64)
	if err != nil {
		return
	}

	state, _ := r.store.Load(ctx)
	if user, ok := state.Users[target]; ok {
		user.Demos = make(map[string]interface{})
		user.DemoHistory = []string{}
		user.UnlockedBatches = []string{}

		r.store.SetUser(ctx, target, user)

		if _, blocked := state.BlockedUsers[target]; blocked {
			r.store.SetBlockedUser(ctx, target, false)
		}

		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("User `%d` reset.", target)))
	}
}

// HandleFindUser processes the /find command
func (r *Router) HandleFindUser(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	args := msg.CommandArguments()
	if args == "" {
		return
	}

	state, _ := r.store.Load(ctx)
	query := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(args)), "@", "")

	var found []string
	for id, user := range state.Users {
		if strings.Contains(strings.ToLower(user.Username), query) {
			found = append(found, fmt.Sprintf("  `%d` | @%s", id, user.Username))
		}
	}

	text := "  Not found."
	if len(found) > 0 {
		text = "  **Found:**\n\n" + strings.Join(found, "\n")
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	r.bot.Send(reply)
}

// HandleUserDetails processes the /user command
func (r *Router) HandleUserDetails(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	args := msg.CommandArguments()
	target, err := strconv.ParseInt(strings.TrimSpace(args), 10, 64)
	if err != nil {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /user [id]"))
		return
	}

	state, _ := r.store.Load(ctx)
	user, ok := state.Users[target]

	name := "Unknown"
	if ok {
		name = user.FirstName
	}

	report := fmt.Sprintf("USER DETAILS: %d\nName: %s\n\n", target, name)
	if _, blocked := state.BlockedUsers[target]; blocked {
		report += "  BLOCKED\n\n"
	}
	report += "--- MEMBERSHIP ---\n"

	var allChats []int64
	for id := range state.AllChats {
		allChats = append(allChats, id)
	}
	for id := range state.FreeBatches {
		allChats = append(allChats, id)
	}
	for id := range state.PaidBatches {
		allChats = append(allChats, id)
	}
	for id := range state.SpecialBatches {
		allChats = append(allChats, id)
	}

	found := false
	seen := make(map[int64]bool)

	for _, cid := range allChats {
		if seen[cid] {
			continue
		}
		seen[cid] = true

		memConfig := tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
				ChatID: cid,
				UserID: target,
			},
		}

		m, err := r.bot.GetChatMember(memConfig)
		if err == nil {
			if m.Status == "member" || m.Status == "administrator" || m.Status == "creator" || m.Status == "restricted" {
				chatName := fmt.Sprintf("%d", cid)
				if name, exists := state.AllChats[cid]; exists {
					chatName = name
				}
				report += fmt.Sprintf("%s: Joined\n", chatName)
				found = true
			}
		}
	}

	if !found {
		report += "Not found in any batch.\n"
	}

	if ok && len(user.DemoHistory) > 0 {
		report += "\n--- DEMO HISTORY ---\n"
		for _, h := range user.DemoHistory {
			report += fmt.Sprintf("  %s\n", h)
		}
	}

	doc := tgbotapi.FileBytes{
		Name:  fmt.Sprintf("scan_%d.txt", target),
		Bytes: []byte(report),
	}

	reply := tgbotapi.NewDocument(msg.Chat.ID, doc)
	reply.Caption = "  Deep Scan"
	r.bot.Send(reply)
}

// HandleAllUsers processes the /allusers command
func (r *Router) HandleAllUsers(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsOwner(ctx, msg.From.ID) {
		return
	}

	state, _ := r.store.Load(ctx)
	report := fmt.Sprintf("ALL USERS DUMP - %v\n", time.Now())
	report += "----------------------------------------\nID | Name | Username\n"

	for id, user := range state.Users {
		report += fmt.Sprintf("%d | %s | @%s\n", id, user.FirstName, user.Username)
	}

	// Create temporary file
	// Send as document
	doc := tgbotapi.FileBytes{
		Name:  "all_users.txt",
		Bytes: []byte(report),
	}

	reply := tgbotapi.NewDocument(msg.Chat.ID, doc)
	reply.Caption = "All Users List"
	r.bot.Send(reply)
}

// HandleExtend processes the /extend command
func (r *Router) HandleExtend(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	args := strings.Fields(msg.CommandArguments())
	if len(args) < 3 {
		return
	}

	uid, _ := strconv.ParseInt(args[0], 10, 64)
	bid := args[1]
	hours, _ := strconv.ParseFloat(args[2], 64)

	state, _ := r.store.Load(ctx)
	if user, ok := state.Users[uid]; ok {
		if val, exists := user.Demos[bid]; exists {
			exp := models.GetDemoExpiry(val)
			newExp := time.Now().Unix()
			if exp > newExp {
				newExp = exp
			}
			newExp += int64(hours * 3600)

			user.Demos[bid] = models.SetDemo(newExp)
			r.store.SetUser(ctx, uid, user)
			r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Extended."))
		}
	}
}

// HandleKick processes the /kick command
func (r *Router) HandleKick(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	args := strings.Fields(msg.CommandArguments())
	if len(args) < 2 {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /kick <user_id> <batch_id>"))
		return
	}

	uid, _ := strconv.ParseInt(args[0], 10, 64)
	bid, _ := strconv.ParseInt(args[1], 10, 64)

	// Kick user (ban then unban immediately)
	banReq := tgbotapi.BanChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: bid,
			UserID: uid,
		},
	}
	r.bot.Request(banReq)
	time.Sleep(500 * time.Millisecond)

	unbanReq := tgbotapi.UnbanChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: bid,
			UserID: uid,
		},
	}
	r.bot.Request(unbanReq)

	state, _ := r.store.Load(ctx)
	if user, ok := state.Users[uid]; ok {
		delete(user.Demos, fmt.Sprintf("%d", bid))
		r.store.SetUser(ctx, uid, user)
	}

	r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "User Kicked from batch."))
}

// HandleApprovePerm processes the /perm and /per commands
func (r *Router) HandleApprovePerm(ctx context.Context, msg *tgbotapi.Message) {
	if !r.isAdminMessage(ctx, msg) {
		return
	}
	args := strings.Fields(msg.CommandArguments())
	link := ""

	linkRegex := regexp.MustCompile(`(https?://t\.me/(?:\+|joinchat/)[a-zA-Z0-9_\-]+)`)

	if msg.ReplyToMessage != nil {
		msgText := msg.ReplyToMessage.Text
		if msgText == "" {
			msgText = msg.ReplyToMessage.Caption
		}
		if match := linkRegex.FindStringSubmatch(msgText); len(match) > 1 {
			link = match[1]
		}
	}

	if link == "" && len(args) > 0 {
		for _, arg := range args {
			if strings.Contains(arg, "t.me") {
				link = arg
				break
			}
		}
	}

	if link == "" {
		resp := tgbotapi.NewMessage(msg.Chat.ID, "Error: Link nahi mila.")
		resp.ReplyToMessageID = msg.MessageID
		r.bot.Send(resp)
		return
	}

	state, _ := r.store.Load(ctx)

	ld, ok := state.LinkMap[link]
	if !ok {
		resp := tgbotapi.NewMessage(msg.Chat.ID, "Error: Ye link database me registered nahi hai.")
		resp.ReplyToMessageID = msg.MessageID
		r.bot.Send(resp)
		return
	}

	targetUID := ld.UserID
	batchID := ld.BatchID

	if targetUID <= 0 || batchID == 0 {
		resp := tgbotapi.NewMessage(msg.Chat.ID, "Error: invalid LinkMap mapping: missing/invalid user ID or batch ID")
		resp.ReplyToMessageID = msg.MessageID
		r.bot.Send(resp)
		return
	}

	// Approve join request
	err := r.api.ApproveChatJoinRequest(batchID, targetUID)
	if err != nil {
		resp := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("  Approval failed: %v", err))
		resp.ReplyToMessageID = msg.MessageID
		r.bot.Send(resp)
		return
	}

	// Upgrade to Permanent (Remove Demo Timer)
	user, exists := state.Users[targetUID]
	if exists && user.Demos != nil {
		delete(user.Demos, fmt.Sprintf("%d", batchID))
		r.store.SetUser(ctx, targetUID, user)
	}

	reqID := fmt.Sprintf("%d_%d", targetUID, batchID)
	// Clear active request and link
	if state.PendingRequests != nil {
		delete(state.PendingRequests, reqID)
	}
	if state.LinkMap != nil {
		delete(state.LinkMap, link)
	}
	r.store.Save(ctx, state)

	resp := tgbotapi.NewMessage(msg.Chat.ID, "  **APPROVED (PERM)**\nUser now has lifetime permanent access.")
	resp.ReplyToMessageID = msg.MessageID
	r.bot.Send(resp)

	bname := fmt.Sprintf("Batch %d", batchID)
	if n, ok := state.AllChats[batchID]; ok {
		bname = n
	}

	userMsg := tgbotapi.NewMessage(targetUID, fmt.Sprintf("  **Congratulations!**\n\nAapki request **%s** ke liye approve ho gayi hai.\n\n  **Access Type:** Lifetime Premium Access\n\nWelcome to the premium community! Ab aap jab chahein apne batches section se isey access kar sakte hain.", bname))
	userMsg.ParseMode = "Markdown"
	r.bot.Send(userMsg)
}

// HandleStats processes the /stats command
func (r *Router) HandleStats(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	state, _ := r.store.Load(ctx)

	totalUsers := len(state.Users)
	totalBlocked := len(state.BlockedUsers)
	totalAdmins := len(state.AdminIDs)

	report := fmt.Sprintf("📊 **BOT STATISTICS**\n\n👥 **Total Users:** `%d`\n🚫 **Blocked Users:** `%d`\n🛡️ **Admins:** `%d`\n",
		totalUsers, totalBlocked, totalAdmins)

	reply := tgbotapi.NewMessage(msg.Chat.ID, report)
	reply.ParseMode = "Markdown"
	r.bot.Send(reply)
}

// executeUniversalKick kicks user from all batches
func (r *Router) executeUniversalKick(ctx context.Context, target int64, permanent bool) {
	state, _ := r.store.Load(ctx)
	var allChannels []int64

	for id := range state.FreeBatches {
		allChannels = append(allChannels, id)
	}
	for id := range state.SpecialBatches {
		allChannels = append(allChannels, id)
	}
	if permanent {
		for id := range state.PaidBatches {
			allChannels = append(allChannels, id)
		}
	}

	for _, bid := range allChannels {
		go func(chat int64) {
			banReq := tgbotapi.BanChatMemberConfig{
				ChatMemberConfig: tgbotapi.ChatMemberConfig{
					ChatID: chat,
					UserID: target,
				},
			}
			r.bot.Request(banReq)

			if !permanent {
				time.Sleep(500 * time.Millisecond)
				unbanReq := tgbotapi.UnbanChatMemberConfig{
					ChatMemberConfig: tgbotapi.ChatMemberConfig{
						ChatID: chat,
						UserID: target,
					},
				}
				r.bot.Request(unbanReq)
			}
		}(bid)
	}

	// Invalidate cache if there is any, and clear demos
	if user, ok := state.Users[target]; ok {
		changed := false
		for _, bid := range allChannels {
			bidStr := fmt.Sprintf("%d", bid)
			if _, exists := user.Demos[bidStr]; exists {
				delete(user.Demos, bidStr)
				changed = true
			}
		}
		if changed {
			r.store.SetUser(ctx, target, user)
		}
	}
}

// HandleGenDemo processes the /gendemo command
func (r *Router) HandleGenDemo(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	args := strings.Fields(msg.CommandArguments())
	if len(args) < 2 {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Usage: /gendemo <user_id> <batch_id>"))
		return
	}

	targetUID, _ := strconv.ParseInt(args[0], 10, 64)
	batchID, _ := strconv.ParseInt(args[1], 10, 64)

	expireDate := time.Now().Unix() + 3*3600
	link, err := r.api.CreateChatInviteLink(batchID, expireDate, 1, false)
	if err != nil {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Error creating demo link: %v", err)))
		return
	}

	state, _ := r.store.Load(ctx)
	if state.LinkMap == nil {
		state.LinkMap = make(map[string]*models.InviteMapping)
	}
	state.LinkMap[link] = &models.InviteMapping{
		UserID:  targetUID,
		BatchID: batchID,
	}

	user, exists := state.Users[targetUID]
	if !exists {
		user = &models.User{ID: targetUID, Demos: make(map[string]interface{})}
		state.Users[targetUID] = user
	}
	if user.Demos == nil {
		user.Demos = make(map[string]interface{})
	}
	user.Demos[fmt.Sprintf("%d", batchID)] = models.SetDemo(time.Now().Unix() + 3*3600)

	r.store.Save(ctx, state)

	text := fmt.Sprintf("✅ **Auto-Demo Link Generated!**\n\n👤 **User:** `%d`\n📦 **Batch:** `%d`\n⏱ **Validity:** `3 Hours`\n\n🔗 **Link to share:**\n`%s`\n\n🔄 *To make it permanent later, just reply to this with:* `/per %s`", targetUID, batchID, link, link)
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "Markdown"
	r.bot.Send(reply)
}

// HandleSetWelcome processes the /setwelcome command
func (r *Router) HandleSetWelcome(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}
	args := strings.Fields(msg.CommandArguments())
	if len(args) < 2 {
		return
	}

	batchID, _ := strconv.ParseInt(args[0], 10, 64)
	welcomeMsg := strings.Join(args[1:], " ")

	state, _ := r.store.Load(ctx)
	if state.CustomWelcomes == nil {
		state.CustomWelcomes = make(map[int64]string)
	}
	state.CustomWelcomes[batchID] = welcomeMsg
	r.store.Save(ctx, state)

	r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "  Welcome Set."))
}

// HandleBatchStats processes the /batchstats command
func (r *Router) HandleBatchStats(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsAdmin(ctx, msg.From.ID) {
		return
	}

	state, _ := r.store.Load(ctx)
	report := fmt.Sprintf("📊 **Batch Statistics**\n\nFree Batches: `%d`\nPaid Batches: `%d`\nSpecial Batches: `%d`\n",
		len(state.FreeBatches), len(state.PaidBatches), len(state.SpecialBatches))

	reply := tgbotapi.NewMessage(msg.Chat.ID, report)
	reply.ParseMode = "Markdown"
	r.bot.Send(reply)
}

// HandleDemo processes the /demo command
func (r *Router) HandleDemo(ctx context.Context, msg *tgbotapi.Message) {
	if !r.isAdminMessage(ctx, msg) {
		return
	}
	args := strings.Fields(msg.CommandArguments())
	link := ""
	hours := 3.0

	linkRegex := regexp.MustCompile(`(https?://t\.me/(?:\+|joinchat/)[a-zA-Z0-9_\-]+)`)

	if msg.ReplyToMessage != nil {
		msgText := msg.ReplyToMessage.Text
		if msgText == "" {
			msgText = msg.ReplyToMessage.Caption
		}
		if match := linkRegex.FindStringSubmatch(msgText); len(match) > 1 {
			link = match[1]
		}
		if len(args) > 0 {
			val, err := strconv.ParseFloat(strings.ReplaceAll(strings.ToLower(args[0]), "h", ""), 64)
			if err == nil {
				hours = val
			}
		}
	}

	if link == "" && len(args) > 0 {
		for _, arg := range args {
			if strings.Contains(arg, "t.me") {
				link = arg
			} else if strings.Contains(strings.ToLower(arg), "h") {
				val, err := strconv.ParseFloat(strings.ReplaceAll(strings.ToLower(arg), "h", ""), 64)
				if err == nil {
					hours = val
				}
			}
		}
	}

	if link == "" {
		resp := tgbotapi.NewMessage(msg.Chat.ID, "Error: Link nahi mila.")
		resp.ReplyToMessageID = msg.MessageID
		r.bot.Send(resp)
		return
	}

	state, _ := r.store.Load(ctx)

	ld, ok := state.LinkMap[link]
	if !ok {
		resp := tgbotapi.NewMessage(msg.Chat.ID, "Error: Ye link database me registered nahi hai.")
		resp.ReplyToMessageID = msg.MessageID
		r.bot.Send(resp)
		return
	}

	targetUID := ld.UserID
	batchID := ld.BatchID

	if targetUID <= 0 || batchID == 0 {
		resp := tgbotapi.NewMessage(msg.Chat.ID, "Error: invalid LinkMap mapping: missing/invalid user ID or batch ID")
		resp.ReplyToMessageID = msg.MessageID
		r.bot.Send(resp)
		return
	}

	// Approve join request
	err := r.api.ApproveChatJoinRequest(batchID, targetUID)
	if err != nil {
		resp := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("  Approval failed: %v", err))
		resp.ReplyToMessageID = msg.MessageID
		r.bot.Send(resp)
		return
	}

	user, exists := state.Users[targetUID]
	if !exists {
		user = &models.User{ID: targetUID, Demos: make(map[string]interface{})}
	}
	if user.Demos == nil {
		user.Demos = make(map[string]interface{})
	}
	user.Demos[fmt.Sprintf("%d", batchID)] = models.SetDemo(time.Now().Unix() + int64(hours*3600))
	r.store.SetUser(ctx, targetUID, user)

	reqID := fmt.Sprintf("%d_%d", targetUID, batchID)
	// Clear active request and link
	if state.PendingRequests != nil {
		delete(state.PendingRequests, reqID)
	}
	if state.LinkMap != nil {
		delete(state.LinkMap, link)
	}
	r.store.Save(ctx, state)

	resp := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("  **APPROVED (DEMO)**\n  Time Given: `%v Hours`", hours))
	resp.ReplyToMessageID = msg.MessageID
	r.bot.Send(resp)

	bname := fmt.Sprintf("Batch %d", batchID)
	if n, ok := state.AllChats[batchID]; ok {
		bname = n
	}

	userMsg := tgbotapi.NewMessage(targetUID, fmt.Sprintf("  **Congratulations!**\n\nAapki request **%s** ke liye approve ho gayi hai.\n\n  **Access Type:** Demo Trial\n  **Duration:** `%v Hours`\n\nKripya diye gaye samay me batch access kar lein.", bname, hours))
	userMsg.ParseMode = "Markdown"
	r.bot.Send(userMsg)
}

// HandleBackup processes the /backup command
func (r *Router) HandleBackup(ctx context.Context, msg *tgbotapi.Message) {
	if !r.auth.IsOwner(ctx, msg.From.ID) {
		return
	}

	state, _ := r.store.Load(ctx)
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		r.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Error generating backup."))
		return
	}

	doc := tgbotapi.FileBytes{
		Name:  fmt.Sprintf("backup_%d.json", time.Now().Unix()),
		Bytes: b,
	}

	reply := tgbotapi.NewDocument(msg.Chat.ID, doc)
	reply.Caption = "DB Backup"
	r.bot.Send(reply)
}
