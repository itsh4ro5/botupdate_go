package services

import (
	"context"
	"fmt"
	"log"
	"strconv"
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
		return
	}

	now := time.Now().Unix()

	// Trigger background membership sync
	go s.RunSync(ctx, nil)

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

		for uid, user := range state.Users {
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
				// Might be kicked or not joined
				log.Printf("User %d missing from mandatory channel, kicking...", uid)
				s.UniversalKick(ctx, uid, state)
				count++
			} else if status, ok := member["status"].(string); ok && (status == "left" || status == "kicked") {
				// Universal Kick
				log.Printf("User %d missing from mandatory channel, kicking...", uid)
				s.UniversalKick(ctx, uid, state)
				count++
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
	for cid := range state.AllChats {
		_ = s.api.BanChatMember(cid, uid, 0, false)
		time.Sleep(200 * time.Millisecond)
		_ = s.api.UnbanChatMember(cid, uid, false)
		time.Sleep(200 * time.Millisecond)
	}
}
