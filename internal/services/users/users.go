package users

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/itsh4ro5/botupdate/internal/database"
)

type UserListItem struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	JoinedAt  int64  `json:"joined_at"`
	IsBlocked bool   `json:"is_blocked"`
}

type UserProfile struct {
	ID                  int64    `json:"id"`
	Username            string   `json:"username"`
	FirstName           string   `json:"first_name"`
	LastName            string   `json:"last_name"`
	JoinedAt            int64    `json:"joined_at"`
	IsBlocked           bool     `json:"is_blocked"`
	TnCAccepted         bool     `json:"tnc_accepted"`
	ReferralCount       int      `json:"referral_count"`
	TotalInvited        int      `json:"total_invited"`
	Tier                string   `json:"tier,omitempty"`
	WelcomeBonusClaimed bool     `json:"welcome_bonus_claimed"`
	FreeUnlocked        bool     `json:"free_unlocked"`
	UnlockedBatches     []string `json:"unlocked_batches"`
}

type ListUsersResponse struct {
	Users      []UserListItem `json:"users"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"pageSize"`
	TotalPages int            `json:"totalPages"`
}

type Service struct {
	store database.Store
}

func NewService(store database.Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListUsers(ctx context.Context, page, pageSize int, search, sortField string, isBlocked *bool) (*ListUsersResponse, error) {
	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	search = strings.ToLower(strings.TrimSpace(search))

	var filtered []UserListItem
	for id, u := range state.Users {
		_, isBlockedActual := state.BlockedUsers[u.ID]
		if isBlocked != nil && isBlockedActual != *isBlocked {
			continue
		}

		if search != "" {
			// Search by ID, username, first name, last name
			idStr := fmt.Sprintf("%d", id)

			if !strings.Contains(idStr, search) &&
				!strings.Contains(strings.ToLower(u.Username), search) &&
				!strings.Contains(strings.ToLower(u.FirstName), search) &&
				!strings.Contains(strings.ToLower(u.LastName), search) {
				continue
			}
		}

		filtered = append(filtered, UserListItem{
			ID:        id,
			Username:  u.Username,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			JoinedAt:  u.JoinedAt,
			IsBlocked: isBlockedActual,
		})
	}

	// Sort
	if sortField == "joined_at_asc" {
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].JoinedAt < filtered[j].JoinedAt })
	} else if sortField == "name" {
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].FirstName < filtered[j].FirstName })
	} else {
		// default joined_at desc
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].JoinedAt > filtered[j].JoinedAt })
	}

	total := len(filtered)
	totalPages := total / pageSize
	if total%pageSize != 0 {
		totalPages++
	}

	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	return &ListUsersResponse{
		Users:      filtered[start:end],
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) GetUser(ctx context.Context, id int64) (*UserProfile, error) {
	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	u, ok := state.Users[id]
	if !ok {
		return nil, nil // Not found
	}

	_, isBlockedActual := state.BlockedUsers[id]

	var batchNames []string
	if u.FreeUnlocked {
		batchNames = append(batchNames, "All Free Batches")
	}
	for _, bid := range u.FreeBatchesJoined {
		if b, ok := state.FreeBatches[bid]; ok {
			batchNames = append(batchNames, b.Name)
		}
	}
	for _, bid := range u.JoinedBatches {
		if b, ok := state.PaidBatches[bid]; ok {
			batchNames = append(batchNames, "[Paid] "+b.Name)
		} else if b, ok := state.SpecialBatches[bid]; ok {
			batchNames = append(batchNames, "[Special] "+b.Name)
		}
	}

	return &UserProfile{
		ID:                  u.ID,
		Username:            u.Username,
		FirstName:           u.FirstName,
		LastName:            u.LastName,
		JoinedAt:            u.JoinedAt,
		IsBlocked:           isBlockedActual,
		TnCAccepted:         u.TnCAccepted,
		ReferralCount:       u.ReferralCount,
		TotalInvited:        u.TotalInvited,
		Tier:                u.Tier,
		WelcomeBonusClaimed: u.WelcomeBonusClaimed,
		FreeUnlocked:        u.FreeUnlocked,
		UnlockedBatches:     batchNames,
	}, nil
}
