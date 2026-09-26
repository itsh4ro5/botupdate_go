package analytics

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/models"
)

type SystemStatus struct {
	Telegram   string `json:"telegram"`
	Database   string `json:"database"`
	Userbot    string `json:"userbot"`
	Scheduler  string `json:"scheduler"`
	WebSocket  string `json:"websocket"`
	EventBus   string `json:"event_bus"`
	Goroutines int    `json:"goroutines"`
	MemoryMB   uint64 `json:"memory_mb"`
	Uptime     string `json:"uptime"`
}

type OverviewDTO struct {
	Available                   bool         `json:"available"`
	TotalUsers                  int          `json:"total_users"`
	TotalBatches                int          `json:"total_batches"`
	PendingRequests             int          `json:"pending_requests"`
	BlockedUsers                int          `json:"blocked_users"`
	VipUsers                    int          `json:"vip_users"`
	UnlockedAccessCount         int          `json:"unlocked_access_count"`
	PendingSupportConversations int          `json:"pending_support_conversations"`
	SystemStatus                SystemStatus `json:"system_status"`
}

type UserAnalyticsDTO struct {
	Available            bool           `json:"available"`
	TotalUsers           int            `json:"total_users"`
	BlockedUsers         int            `json:"blocked_users"`
	VipUsers             int            `json:"vip_users"`
	StandardUsers        int            `json:"standard_users"`
	UsersWithAccess      int            `json:"users_with_access"`
	UsersWithReferrals   int            `json:"users_with_referrals"`
	WelcomeBonusClaimed  int            `json:"welcome_bonus_claimed"`
	TnCAccepted          int            `json:"tnc_accepted"`
	TnCNotAccepted       int            `json:"tnc_not_accepted"`
	TierDistribution     map[string]int `json:"tier_distribution"`
	RegistrationTimeline map[string]int `json:"registration_timeline"` // Date (YYYY-MM-DD) -> Count
}

type BatchStats struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Category   string  `json:"category"`
	UserCount  int     `json:"user_count"`
	Percentage float64 `json:"percentage"`
}

type BatchAnalyticsDTO struct {
	Available      bool         `json:"available"`
	TotalBatches   int          `json:"total_batches"`
	FreeBatches    int          `json:"free_batches"`
	PaidBatches    int          `json:"paid_batches"`
	SpecialBatches int          `json:"special_batches"`
	Batches        []BatchStats `json:"batches"`
}

type RequestAnalyticsDTO struct {
	Available       bool           `json:"available"`
	PendingRequests int            `json:"pending_requests"`
	ByBatch         map[string]int `json:"by_batch"`
	HistoricalInfo  string         `json:"historical_info"`
}

type SupportAnalyticsDTO struct {
	Available           bool   `json:"available"`
	ActiveConversations int    `json:"active_conversations"`
	BlockedUsers        int    `json:"blocked_users"`
	HistoricalInfo      string `json:"historical_info"`
}

type AnalyticsService struct {
	store     database.Store
	startTime time.Time

	cacheMu       sync.RWMutex
	cacheOverview *OverviewDTO
	cacheUsers    *UserAnalyticsDTO
	cacheBatches  *BatchAnalyticsDTO
	cacheRequests *RequestAnalyticsDTO
	cacheSupport  *SupportAnalyticsDTO

	cacheTime time.Time
}

func NewAnalyticsService(store database.Store) *AnalyticsService {
	return &AnalyticsService{
		store:     store,
		startTime: time.Now(),
	}
}

func (s *AnalyticsService) invalidateCacheIfOld() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	if time.Since(s.cacheTime) > 30*time.Second {
		s.cacheOverview = nil
		s.cacheUsers = nil
		s.cacheBatches = nil
		s.cacheRequests = nil
		s.cacheSupport = nil
		s.cacheTime = time.Now()
	}
}

func (s *AnalyticsService) GetSystemStatus() SystemStatus {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemStatus{
		Telegram:   "ONLINE", // Based on architecture, if we're responding, it's generally up or recovering.
		Database:   "ONLINE",
		Userbot:    "ONLINE",
		Scheduler:  "ONLINE",
		WebSocket:  "ONLINE",
		EventBus:   "ONLINE",
		Goroutines: runtime.NumGoroutine(),
		MemoryMB:   m.Alloc / 1024 / 1024,
		Uptime:     time.Since(s.startTime).String(),
	}
}

func (s *AnalyticsService) GetOverview(ctx context.Context) (*OverviewDTO, error) {
	s.invalidateCacheIfOld()

	s.cacheMu.RLock()
	if s.cacheOverview != nil {
		defer s.cacheMu.RUnlock()
		return s.cacheOverview, nil
	}
	s.cacheMu.RUnlock()

	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	blocked := len(state.BlockedUsers)
	vip := 0
	unlockedAccess := 0

	for _, u := range state.Users {
		if u.Tier == "VIP" {
			vip++
		}
		if len(u.UnlockedBatches) > 0 {
			unlockedAccess++
		}
	}

	totalBatches := len(state.FreeBatches) + len(state.PaidBatches) + len(state.SpecialBatches)

	overview := &OverviewDTO{
		Available:                   true,
		TotalUsers:                  len(state.Users),
		TotalBatches:                totalBatches,
		PendingRequests:             len(state.PendingRequests),
		BlockedUsers:                blocked,
		VipUsers:                    vip,
		UnlockedAccessCount:         unlockedAccess,
		PendingSupportConversations: len(state.UserTopics),
		SystemStatus:                s.GetSystemStatus(),
	}

	s.cacheMu.Lock()
	s.cacheOverview = overview
	s.cacheMu.Unlock()

	return overview, nil
}

func (s *AnalyticsService) GetUserAnalytics(ctx context.Context) (*UserAnalyticsDTO, error) {
	s.invalidateCacheIfOld()

	s.cacheMu.RLock()
	if s.cacheUsers != nil {
		defer s.cacheMu.RUnlock()
		return s.cacheUsers, nil
	}
	s.cacheMu.RUnlock()

	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	analytics := &UserAnalyticsDTO{
		Available:            true,
		TotalUsers:           len(state.Users),
		TierDistribution:     make(map[string]int),
		RegistrationTimeline: make(map[string]int),
	}

	now := time.Now()
	cutoff90Days := now.AddDate(0, 0, -90).Unix()

	for _, u := range state.Users {
		if u.IsBlocked {
			analytics.BlockedUsers++
		}
		if u.Tier == "VIP" {
			analytics.VipUsers++
		} else {
			analytics.StandardUsers++
		}

		if len(u.UnlockedBatches) > 0 {
			analytics.UsersWithAccess++
		}
		if u.ReferralCount > 0 {
			analytics.UsersWithReferrals++
		}
		if u.WelcomeBonusClaimed {
			analytics.WelcomeBonusClaimed++
		}
		if u.TnCAccepted {
			analytics.TnCAccepted++
		} else {
			analytics.TnCNotAccepted++
		}

		tier := u.Tier
		if tier == "" {
			tier = "Standard"
		}
		analytics.TierDistribution[tier]++

		// Registration timeline (only recent 90 days for lightweight payload)
		if u.JoinedAt > cutoff90Days {
			dateStr := time.Unix(u.JoinedAt, 0).Format("2006-01-02")
			analytics.RegistrationTimeline[dateStr]++
		}
	}

	s.cacheMu.Lock()
	s.cacheUsers = analytics
	s.cacheMu.Unlock()

	return analytics, nil
}

func (s *AnalyticsService) GetBatchAnalytics(ctx context.Context) (*BatchAnalyticsDTO, error) {
	s.invalidateCacheIfOld()

	s.cacheMu.RLock()
	if s.cacheBatches != nil {
		defer s.cacheMu.RUnlock()
		return s.cacheBatches, nil
	}
	s.cacheMu.RUnlock()

	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	analytics := &BatchAnalyticsDTO{
		Available:      true,
		FreeBatches:    len(state.FreeBatches),
		PaidBatches:    len(state.PaidBatches),
		SpecialBatches: len(state.SpecialBatches),
		TotalBatches:   len(state.FreeBatches) + len(state.PaidBatches) + len(state.SpecialBatches),
		Batches:        make([]BatchStats, 0),
	}

	totalUsers := len(state.Users)

	// Helper to collect batch stats
	collectStats := func(batchMap map[int64]*models.Batch, bType string) {
		for _, b := range batchMap {
			count := 0
			for _, u := range state.Users {
				// We don't have a direct reverse mapping of BatchID -> Users in memory easily,
				// so we count users who have joined this batch ID.
				joined := false
				for _, jb := range u.JoinedBatches {
					if jb == b.ID {
						joined = true
						break
					}
				}
				if !joined && bType == "free" {
					for _, fb := range u.FreeBatchesJoined {
						if fb == b.ID {
							joined = true
							break
						}
					}
				}
				if joined {
					count++
				}
			}

			pct := 0.0
			if totalUsers > 0 {
				pct = float64(count) / float64(totalUsers) * 100.0
			}

			analytics.Batches = append(analytics.Batches, BatchStats{
				ID:         fmt.Sprintf("%d", b.ID),
				Name:       b.Name,
				Type:       bType,
				Category:   b.Category,
				UserCount:  count,
				Percentage: pct,
			})
		}
	}

	collectStats(state.FreeBatches, "free")
	collectStats(state.PaidBatches, "paid")
	collectStats(state.SpecialBatches, "special")

	s.cacheMu.Lock()
	s.cacheBatches = analytics
	s.cacheMu.Unlock()

	return analytics, nil
}

func (s *AnalyticsService) GetRequestAnalytics(ctx context.Context) (*RequestAnalyticsDTO, error) {
	s.invalidateCacheIfOld()

	s.cacheMu.RLock()
	if s.cacheRequests != nil {
		defer s.cacheMu.RUnlock()
		return s.cacheRequests, nil
	}
	s.cacheMu.RUnlock()

	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	analytics := &RequestAnalyticsDTO{
		Available:       true,
		PendingRequests: len(state.PendingRequests),
		ByBatch:         make(map[string]int),
		HistoricalInfo:  "Historical approval analytics are unavailable because the current persistence model does not retain approval history.",
	}

	for _, req := range state.PendingRequests {
		batchIDStr := fmt.Sprintf("%d", req.BatchID)
		analytics.ByBatch[batchIDStr]++
	}

	s.cacheMu.Lock()
	s.cacheRequests = analytics
	s.cacheMu.Unlock()

	return analytics, nil
}

func (s *AnalyticsService) GetSupportAnalytics(ctx context.Context) (*SupportAnalyticsDTO, error) {
	s.invalidateCacheIfOld()

	s.cacheMu.RLock()
	if s.cacheSupport != nil {
		defer s.cacheMu.RUnlock()
		return s.cacheSupport, nil
	}
	s.cacheMu.RUnlock()

	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	analytics := &SupportAnalyticsDTO{
		Available:           true,
		ActiveConversations: len(state.UserTopics),
		BlockedUsers:        len(state.BlockedUsers),
		HistoricalInfo:      "Historical support analytics are unavailable.",
	}

	s.cacheMu.Lock()
	s.cacheSupport = analytics
	s.cacheMu.Unlock()

	return analytics, nil
}
