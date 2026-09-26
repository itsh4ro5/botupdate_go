package requests

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/events"
	"github.com/itsh4ro5/botupdate/internal/telegram"
)

type RequestDTO struct {
	ID          string    `json:"id"`
	UserID      int64     `json:"user_id"`
	Username    string    `json:"username"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	BatchID     int64     `json:"batch_id"`
	BatchName   string    `json:"batch_name"`
	RequestedAt time.Time `json:"requested_at"`
	Status      string    `json:"status"` // "pending"
}

type ListRequestsResponse struct {
	Items      []RequestDTO `json:"items"`
	Page       int          `json:"page"`
	PageSize   int          `json:"pageSize"`
	Total      int          `json:"total"`
	TotalPages int          `json:"totalPages"`
}

type RequestService struct {
	store    database.Store
	eventBus *events.Bus
	api      *telegram.APIClient
	bot      *tgbotapi.BotAPI
}

func NewRequestService(store database.Store, eventBus *events.Bus, api *telegram.APIClient, bot *tgbotapi.BotAPI) *RequestService {
	return &RequestService{
		store:    store,
		eventBus: eventBus,
		api:      api,
		bot:      bot,
	}
}

func (s *RequestService) ListRequests(ctx context.Context, page, pageSize int, search, batchIDStr string) (*ListRequestsResponse, error) {
	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	var all []RequestDTO
	sLower := strings.ToLower(search)

	for reqID, req := range state.PendingRequests {
		if req == nil {
			continue
		}

		// Map user details
		var username, firstName, lastName string
		if u, ok := state.Users[req.UserID]; ok {
			username = u.Username
			firstName = u.FirstName
			lastName = u.LastName
		} else {
			firstName = "Unknown"
		}

		// Map batch details
		batchName := fmt.Sprintf("Batch %d", req.BatchID)
		if bn, ok := state.AllChats[req.BatchID]; ok {
			batchName = bn
		}

		// Apply filters
		if batchIDStr != "" && fmt.Sprintf("%d", req.BatchID) != batchIDStr {
			continue
		}

		if sLower != "" {
			uidStr := fmt.Sprintf("%d", req.UserID)
			if !strings.Contains(strings.ToLower(username), sLower) &&
				!strings.Contains(strings.ToLower(firstName), sLower) &&
				!strings.Contains(strings.ToLower(lastName), sLower) &&
				!strings.Contains(uidStr, sLower) {
				continue
			}
		}

		all = append(all, RequestDTO{
			ID:          reqID,
			UserID:      req.UserID,
			Username:    username,
			FirstName:   firstName,
			LastName:    lastName,
			BatchID:     req.BatchID,
			BatchName:   batchName,
			RequestedAt: req.Requested,
			Status:      "pending",
		})
	}

	// Sort by requested_at descending (newest first)
	sort.Slice(all, func(i, j int) bool {
		return all[i].RequestedAt.After(all[j].RequestedAt)
	})

	total := len(all)
	totalPages := (total + pageSize - 1) / pageSize

	// Pagination
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	end := start + pageSize
	if start >= total {
		return &ListRequestsResponse{
			Items:      []RequestDTO{},
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		}, nil
	}
	if end > total {
		end = total
	}

	return &ListRequestsResponse{
		Items:      all[start:end],
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *RequestService) GetRequest(ctx context.Context, id string) (*RequestDTO, error) {
	state, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}

	req, ok := state.PendingRequests[id]
	if !ok || req == nil {
		return nil, nil // not found
	}

	var username, firstName, lastName string
	if u, ok := state.Users[req.UserID]; ok {
		username = u.Username
		firstName = u.FirstName
		lastName = u.LastName
	} else {
		firstName = "Unknown"
	}

	batchName := fmt.Sprintf("Batch %d", req.BatchID)
	if bn, ok := state.AllChats[req.BatchID]; ok {
		batchName = bn
	}

	return &RequestDTO{
		ID:          id,
		UserID:      req.UserID,
		Username:    username,
		FirstName:   firstName,
		LastName:    lastName,
		BatchID:     req.BatchID,
		BatchName:   batchName,
		RequestedAt: req.Requested,
		Status:      "pending",
	}, nil
}

func (s *RequestService) ApproveRequest(ctx context.Context, id string, adminID string) error {
	state, err := s.store.Load(ctx)
	if err != nil {
		return err
	}

	req, ok := state.PendingRequests[id]
	if !ok || req == nil {
		return errors.New("request not found")
	}

	// 1. Tell Telegram to approve
	err = s.api.ApproveChatJoinRequest(req.BatchID, req.UserID)
	if err != nil {
		return fmt.Errorf("telegram approval failed: %v", err)
	}

	// 2. Clear Demo Timer if exists
	user, exists := state.Users[req.UserID]
	if exists && user.Demos != nil {
		delete(user.Demos, fmt.Sprintf("%d", req.BatchID))
		_ = s.store.SetUser(ctx, req.UserID, user)
	}

	// 3. Find and remove associated link map entry
	var linkToRemove string
	for l, ld := range state.LinkMap {
		if ld.UserID == req.UserID && ld.BatchID == req.BatchID {
			linkToRemove = l
			break
		}
	}

	// Since we are modifying the state struct directly and we want targeted if possible,
	// Wait, deleting from PendingRequests and LinkMap using store.SetPendingRequest and store.SetInviteLink
	_ = s.store.SetPendingRequest(ctx, id, nil)
	if linkToRemove != "" {
		_ = s.store.SetInviteLink(ctx, linkToRemove, nil)
	}

	// Send user telegram message
	batchName := fmt.Sprintf("Batch %d", req.BatchID)
	if bn, ok := state.AllChats[req.BatchID]; ok {
		batchName = bn
	}

	if s.bot != nil {
		userMsg := tgbotapi.NewMessage(req.UserID, fmt.Sprintf("  **Congratulations!**\n\nAapki request **%s** ke liye approve ho gayi hai.\n\n  **Access Type:** Lifetime Premium Access\n\nWelcome to the premium community! Ab aap jab chahein apne batches section se isey access kar sakte hain.", batchName))
		userMsg.ParseMode = "Markdown"
		_, _ = s.bot.Send(userMsg)
	}

	if s.eventBus != nil {
		s.eventBus.Publish(events.TypeRequestApproved, events.SeverityInfo, map[string]interface{}{
			"request_id": id,
			"user_id":    req.UserID,
			"batch_id":   req.BatchID,
			"admin_id":   adminID,
		})
	}

	return nil
}

func (s *RequestService) RejectRequest(ctx context.Context, id string, adminID string) error {
	state, err := s.store.Load(ctx)
	if err != nil {
		return err
	}

	req, ok := state.PendingRequests[id]
	if !ok || req == nil {
		return errors.New("request not found")
	}

	// 1. Tell Telegram to decline
	err = s.api.DeclineChatJoinRequest(req.BatchID, req.UserID)
	if err != nil {
		return fmt.Errorf("telegram rejection failed: %v", err)
	}

	// 2. Find and remove associated link map entry
	var linkToRemove string
	for l, ld := range state.LinkMap {
		if ld.UserID == req.UserID && ld.BatchID == req.BatchID {
			linkToRemove = l
			break
		}
	}

	// Delete using targeted functions
	_ = s.store.SetPendingRequest(ctx, id, nil)
	if linkToRemove != "" {
		_ = s.store.SetInviteLink(ctx, linkToRemove, nil)
	}

	// Send user telegram message
	batchName := fmt.Sprintf("Batch %d", req.BatchID)
	if bn, ok := state.AllChats[req.BatchID]; ok {
		batchName = bn
	}

	if s.bot != nil {
		userMsg := tgbotapi.NewMessage(req.UserID, fmt.Sprintf("  **Request Declined**\n\nAapki request **%s** ke liye decline kar di gayi hai.", batchName))
		userMsg.ParseMode = "Markdown"
		_, _ = s.bot.Send(userMsg)
	}

	if s.eventBus != nil {
		s.eventBus.Publish(events.TypeRequestRejected, events.SeverityWarning, map[string]interface{}{
			"request_id": id,
			"user_id":    req.UserID,
			"batch_id":   req.BatchID,
			"admin_id":   adminID,
		})
	}

	return nil
}
