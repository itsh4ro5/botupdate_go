package services

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/models"
	"github.com/itsh4ro5/botupdate/internal/telegram"
)

// Scheduler runs background cleanup and expiry tasks
type Scheduler struct {
	store              database.Store
	api                *telegram.APIClient
	ticker             *time.Ticker
	wg                 sync.WaitGroup
	mandatoryChannelID int64
}

func NewScheduler(store database.Store, api *telegram.APIClient, mandatoryChannelID int64) *Scheduler {
	return &Scheduler{
		store:              store,
		api:                api,
		mandatoryChannelID: mandatoryChannelID,
	}
}

func (s *Scheduler) GetMandatoryChannelID() int64 {
	return s.mandatoryChannelID
}

// Start launches the background scheduler
func (s *Scheduler) Start(ctx context.Context) {
	s.ticker = time.NewTicker(60 * time.Second)
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-ctx.Done():
				log.Println("Scheduler stopping...")
				s.ticker.Stop()
				return
			case <-s.ticker.C:
				s.runCleanup(ctx)
			}
		}
	}()
}

// Stop waits for background tasks to finish
func (s *Scheduler) Stop() {
	s.wg.Wait()
}

func (s *Scheduler) runCleanup(ctx context.Context) {
	state, err := s.store.Load(ctx)
	if err != nil {
		log.Printf("Scheduler load error: %v", err)
		log.Printf("Membership enforcement skipped: verification unavailable")
		return
	}

	now := time.Now().Unix()

	// Check demo expirations
	for id, user := range state.Users {
		if user.Demos == nil {
			continue
		}

		var expiredBids []string
		for bidStr, val := range user.Demos {
			expTime := models.GetDemoExpiry(val)
			if now > expTime {
				expiredBids = append(expiredBids, bidStr)
			}
		}

		if len(expiredBids) > 0 {
			for _, b := range expiredBids {
				delete(user.Demos, b)
				bid, _ := strconv.ParseInt(b, 10, 64)
				log.Printf("Demo expired for user %d in batch %d", id, bid)

				// Kick user (ban then unban)
				if s.api != nil {
					_ = s.api.BanChatMember(bid, id, 0, false)
					time.Sleep(500 * time.Millisecond) // avoid flood
					_ = s.api.UnbanChatMember(bid, id, false)
				}
			}
			// Should also invalidate membership cache here if exposed
			_ = s.store.SetUser(ctx, id, user)
		}
	}

	// Process ScheduledDeletes
	for _, sd := range state.ScheduledDeletes {
		if now >= sd.DeleteAt.Unix() {
			if s.api != nil {
				err := s.api.DeleteMessage(sd.ChatID, sd.MessageID)
				if err != nil {
					log.Printf("Scheduler: Failed to delete message %d in chat %d: %v", sd.MessageID, sd.ChatID, err)
				} else {
					log.Printf("Scheduler: Successfully deleted scheduled message %d in chat %d", sd.MessageID, sd.ChatID)
				}
				// Remove from DB whether successful or failed (so it doesn't loop forever if message is already deleted)
				_ = s.store.RemoveScheduledDelete(ctx, sd.ChatID, sd.MessageID)
			}
		}
	}
}

// RunSync manually triggers the background sync process (checking mandatory channel membership)
func (s *Scheduler) RunSync(ctx context.Context, cb func(string)) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		state, err := s.store.Load(ctx)
		if err != nil {
			if cb != nil {
				cb("❌ Error loading state")
			}
			log.Printf("Membership enforcement skipped: verification unavailable (MongoDB error)")
			return
		}

		if cb != nil {
			cb("  **Syncing Users...**")
		}

		if s.mandatoryChannelID == 0 {
			if cb != nil {
				cb("✅ Sync skipped: No mandatory channel.")
			}
			return
		}

		count := 0
		failed := 0

		// Extract botID from the token (the part before the colon)
		botIDStr := strings.Split(s.api.Token, ":")[0]
		botID, err := strconv.ParseInt(botIDStr, 10, 64)
		if err != nil {
			log.Printf("Membership enforcement skipped: verification unavailable (cannot parse bot ID from token)")
			if cb != nil {
				cb("❌ Sync skipped: Bot ID could not be parsed.")
			}
			return
		}

		botMember, botErr := s.api.GetChatMember(s.mandatoryChannelID, botID)
		if botErr != nil {
			log.Printf("Membership enforcement skipped: verification unavailable (cannot fetch bot status: %v)", botErr)
			if cb != nil {
				cb("❌ Sync skipped: Bot admin status could not be verified.")
			}
			return
		}
		if botStatus, _ := botMember["status"].(string); botStatus != "administrator" && botStatus != "creator" {
			log.Printf("Membership enforcement skipped: bot is not an administrator in the mandatory channel (status: %s)", botStatus)
			if cb != nil {
				cb("❌ Sync skipped: Bot is not an admin in the mandatory channel.")
			}
			return
		}

		for uid, user := range state.Users {
			// Respect shutdown signal to prevent zombie loops
			select {
			case <-ctx.Done():
				log.Println("RunSync canceled by context (shutdown)")
				return
			default:
			}

			if _, blocked := state.BlockedUsers[uid]; blocked {
				continue
			}
			if _, admin := state.AdminIDs[uid]; admin {
				continue
			}
			if !user.TnCAccepted {
				continue
			}

			member, err := s.api.GetChatMember(s.mandatoryChannelID, uid)
			if err != nil {
				// DO NOT kick on API error (e.g. rate limit, timeout, network failure)
				log.Printf("MEMBERSHIP_CHECK user=%d chat=%d ERROR=%v", uid, s.mandatoryChannelID, err)
				failed++
			} else if status, ok := member["status"].(string); ok {
				// Only explicit left or kicked triggers UniversalKick
				if status == "left" || status == "kicked" {
					log.Printf("MEMBERSHIP_CHECK user=%d chat=%d status=%s -> Action: KICK", uid, s.mandatoryChannelID, status)
					log.Printf("User %d missing from mandatory channel (status %s), kicking...", uid, status)
					s.UniversalKick(ctx, uid, state)
					count++
				} else {
					log.Printf("MEMBERSHIP_CHECK user=%d chat=%d status=%s -> Action: KEEP", uid, s.mandatoryChannelID, status)
				}
			}
			time.Sleep(200 * time.Millisecond) // rate limit
		}

		if cb != nil {
			cb(fmt.Sprintf("✅ **Sync Complete!**\nKicked: %d\nFailed Checks: %d", count, failed))
		}
	}()
}

// UniversalKick removes a user from all known channels
func (s *Scheduler) UniversalKick(ctx context.Context, uid int64, state *models.BotState) {
	// 1. NEVER kick an admin
	if _, isAdmin := state.AdminIDs[uid]; isAdmin {
		log.Printf("UniversalKick aborted: User %d is an admin", uid)
		return
	}

	// 2. NEVER kick the bot itself
	if s.api != nil {
		botIDStr := strings.Split(s.api.Token, ":")[0]
		botID, _ := strconv.ParseInt(botIDStr, 10, 64)
		if uid == botID {
			log.Printf("CRITICAL: UniversalKick aborted to prevent bot self-ban (uid %d)", uid)
			return
		}
	}

	for cid := range state.AllChats {
		_ = s.api.BanChatMember(cid, uid, 0, false)
		time.Sleep(200 * time.Millisecond)
		_ = s.api.UnbanChatMember(cid, uid, false)
		time.Sleep(200 * time.Millisecond)
	}
}
