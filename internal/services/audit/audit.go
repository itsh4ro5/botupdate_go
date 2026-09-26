package audit

import (
	"strings"
	"sync"
	"time"

	"github.com/itsh4ro5/botupdate/internal/events"
)

type AuditLog struct {
	ID         string                 `json:"id"`
	Timestamp  string                 `json:"timestamp"`
	Actor      string                 `json:"actor_username"`
	ActorRole  string                 `json:"actor_role"`
	Action     string                 `json:"action"`
	TargetType string                 `json:"target_type"`
	TargetID   string                 `json:"target_id"`
	Success    bool                   `json:"success"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type PaginatedAudit struct {
	Items      []AuditLog `json:"items"`
	Page       int        `json:"page"`
	PageSize   int        `json:"pageSize"`
	Total      int        `json:"total"`
	TotalPages int        `json:"totalPages"`
}

type AuditService struct {
	mu     sync.RWMutex
	logs   []AuditLog
	maxLog int
}

func NewAuditService(eventBus *events.Bus) *AuditService {
	s := &AuditService{
		logs:   make([]AuditLog, 0),
		maxLog: 10000, // Bounded in-memory audit buffer
	}
	s.listen(eventBus)
	return s
}

func (s *AuditService) Log(actor, role, action, targetType, targetID string, success bool, metadata map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := AuditLog{
		ID:         generateID(),
		Timestamp:  time.Now().Format(time.RFC3339),
		Actor:      actor,
		ActorRole:  role,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Success:    success,
		Metadata:   metadata,
	}

	// Prepend for reverse chronological order
	s.logs = append([]AuditLog{entry}, s.logs...)
	if len(s.logs) > s.maxLog {
		s.logs = s.logs[:s.maxLog]
	}
}

func (s *AuditService) listen(bus *events.Bus) {
	ch := bus.Subscribe()
	go func() {
		for e := range ch {
			switch string(e.Type) {
			case "ADMIN_CREATED", "ADMIN_ROLE_CHANGED", "SESSION_REVOKED", "SESSION_REVOKED_ALL", "ADMIN_LOGIN", "ADMIN_LOGOUT":
				actor, _ := e.Data["actor"].(string)
				if actor == "" {
					actor = "SYSTEM"
				}
				s.Log(actor, "UNKNOWN", string(e.Type), "SYSTEM", "", true, e.Data)
			}
		}
	}()
}

func (s *AuditService) GetLogs(page, pageSize int, search string, action string) PaginatedAudit {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []AuditLog
	for _, l := range s.logs {
		if search != "" && !strings.Contains(strings.ToLower(l.Actor), strings.ToLower(search)) && !strings.Contains(strings.ToLower(l.TargetID), strings.ToLower(search)) {
			continue
		}
		if action != "" && l.Action != action {
			continue
		}
		filtered = append(filtered, l)
	}

	// Already sorted reverse chronologically during insertion

	total := len(filtered)
	totalPages := total / pageSize
	if total%pageSize != 0 {
		totalPages++
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	var items []AuditLog
	if start < total {
		items = filtered[start:end]
	} else {
		items = []AuditLog{}
	}

	return PaginatedAudit{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}

// Simple ID generator for memory logs
func generateID() string {
	return time.Now().Format("20060102150405.000")
}
