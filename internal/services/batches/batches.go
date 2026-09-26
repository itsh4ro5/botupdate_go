package batches

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/events"
	"github.com/itsh4ro5/botupdate/internal/models"
)

type BatchDTO struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Type        string `json:"type"` // free, paid, special
	WelcomeText string `json:"welcome_text"`
	UserCount   int    `json:"user_count"`
}

type BatchUserDTO struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Tier      string `json:"tier"`
}

type BatchDetailDTO struct {
	Batch BatchDTO       `json:"batch"`
	Users []BatchUserDTO `json:"users"`
}

type BatchService struct {
	store    database.Store
	eventBus *events.Bus
}

func NewBatchService(store database.Store, eventBus *events.Bus) *BatchService {
	return &BatchService{
		store:    store,
		eventBus: eventBus,
	}
}

func (s *BatchService) ListBatches(ctx context.Context, search string, category string, typeFilter string) ([]BatchDTO, error) {
	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	var allBatches []BatchDTO

	appendBatches := func(batchMap map[int64]*models.Batch, bType string) {
		for _, b := range batchMap {
			if typeFilter != "" && b.Type != typeFilter {
				continue
			}
			if category != "" && b.Category != category {
				continue
			}
			if search != "" {
				sLower := strings.ToLower(search)
				idStr := strconv.FormatInt(b.ID, 10)
				if !strings.Contains(strings.ToLower(b.Name), sLower) && !strings.Contains(idStr, sLower) {
					continue
				}
			}

			// Calculate UserCount
			count := 0
			idStr := strconv.FormatInt(b.ID, 10)
			for _, u := range state.Users {
				if bType == "free" {
					for _, fb := range u.FreeBatchesJoined {
						if fb == b.ID {
							count++
							break
						}
					}
				} else if bType == "paid" {
					for _, pb := range u.JoinedBatches {
						if pb == b.ID {
							count++
							break
						}
					}
				} else if bType == "special" {
					for _, ub := range u.UnlockedBatches {
						if ub == idStr {
							count++
							break
						}
					}
				}
			}

			allBatches = append(allBatches, BatchDTO{
				ID:          b.ID,
				Name:        b.Name,
				Category:    b.Category,
				Type:        b.Type,
				WelcomeText: b.WelcomeText,
				UserCount:   count,
			})
		}
	}

	appendBatches(state.FreeBatches, "free")
	appendBatches(state.PaidBatches, "paid")
	appendBatches(state.SpecialBatches, "special")

	// Sort by ID desc by default
	sort.Slice(allBatches, func(i, j int) bool {
		return allBatches[i].ID > allBatches[j].ID
	})

	return allBatches, nil
}

func (s *BatchService) GetBatch(ctx context.Context, id int64) (*BatchDetailDTO, error) {
	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	var batch *models.Batch
	bType := ""

	if b, ok := state.FreeBatches[id]; ok {
		batch = b
		bType = "free"
	} else if b, ok := state.PaidBatches[id]; ok {
		batch = b
		bType = "paid"
	} else if b, ok := state.SpecialBatches[id]; ok {
		batch = b
		bType = "special"
	}

	if batch == nil {
		return nil, nil // not found
	}

	var usersWithAccess []BatchUserDTO
	idStr := strconv.FormatInt(id, 10)

	for _, u := range state.Users {
		hasAccess := false
		if bType == "free" {
			for _, fb := range u.FreeBatchesJoined {
				if fb == id {
					hasAccess = true
					break
				}
			}
		} else if bType == "paid" {
			for _, pb := range u.JoinedBatches {
				if pb == id {
					hasAccess = true
					break
				}
			}
		} else if bType == "special" {
			for _, ub := range u.UnlockedBatches {
				if ub == idStr {
					hasAccess = true
					break
				}
			}
		}

		if hasAccess {
			usersWithAccess = append(usersWithAccess, BatchUserDTO{
				ID:        u.ID,
				Username:  u.Username,
				FirstName: u.FirstName,
				LastName:  u.LastName,
				Tier:      u.Tier,
			})
		}
	}

	// Sort users by ID desc
	sort.Slice(usersWithAccess, func(i, j int) bool {
		return usersWithAccess[i].ID > usersWithAccess[j].ID
	})

	return &BatchDetailDTO{
		Batch: BatchDTO{
			ID:          batch.ID,
			Name:        batch.Name,
			Category:    batch.Category,
			Type:        batch.Type,
			WelcomeText: batch.WelcomeText,
			UserCount:   len(usersWithAccess),
		},
		Users: usersWithAccess,
	}, nil
}
