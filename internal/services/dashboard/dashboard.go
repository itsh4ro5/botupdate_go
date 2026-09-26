package dashboard

import (
	"context"
	"sync"
	"time"

	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/models"
)

type Service struct {
	store       database.Store
	mu          sync.RWMutex
	cache       *models.DashboardOverview
	lastFetched time.Time
	cacheTTL    time.Duration
}

func NewService(store database.Store) *Service {
	return &Service{
		store:    store,
		cacheTTL: 15 * time.Second,
	}
}

func (s *Service) GetOverview(ctx context.Context) (*models.DashboardOverview, error) {
	s.mu.RLock()
	if s.cache != nil && time.Since(s.lastFetched) < s.cacheTTL {
		cachedCopy := *s.cache // Return a value copy to prevent mutation
		s.mu.RUnlock()
		return &cachedCopy, nil
	}
	s.mu.RUnlock()

	// Cache expired or missing, fetch from store
	overview, err := s.store.GetDashboardOverview(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.cache = overview
	s.lastFetched = time.Now()
	cachedCopy := *overview
	s.mu.Unlock()

	return &cachedCopy, nil
}
