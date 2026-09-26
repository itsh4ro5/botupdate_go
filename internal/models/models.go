package models

import (
	"time"
)

type WebAdmin struct {
	ID                 string    `json:"id" bson:"_id"`
	Username           string    `json:"username" bson:"username"`
	PasswordHash       string    `json:"-" bson:"password_hash"`
	Role               string    `json:"role" bson:"role"` // OWNER, ADMIN, SUPPORT
	TelegramID         int64     `json:"telegram_id,omitempty" bson:"telegram_id,omitempty"`
	CreatedAt          time.Time `json:"created_at" bson:"created_at"`
	MustChangePassword bool      `json:"must_change_password" bson:"must_change_password"`
}

type WebSession struct {
	SessionID string    `json:"session_id" bson:"_id"`
	AdminID   string    `json:"admin_id" bson:"admin_id"`
	Role      string    `json:"role" bson:"role"`
	ExpiresAt time.Time `json:"expires_at" bson:"expires_at"`
}

type DashboardOverview struct {
	Users struct {
		Total  int `json:"total"`
		Active int `json:"active"`
	} `json:"users"`
	Batches struct {
		Total int `json:"total"`
	} `json:"batches"`
	Requests struct {
		Pending int `json:"pending"`
	} `json:"requests"`
	System struct {
		Telegram  string `json:"telegram"`
		Database  string `json:"database"`
		Scheduler string `json:"scheduler"`
		Userbot   string `json:"userbot"`
	} `json:"system"`
	GeneratedAt time.Time `json:"generated_at"`
}

type User struct {
	ID                  int64                  `json:"id" bson:"_id"`
	Username            string                 `json:"username" bson:"username"`
	FirstName           string                 `json:"first_name" bson:"first_name"`
	LastName            string                 `json:"last_name" bson:"last_name"`
	JoinedAt            int64                  `json:"joined_at" bson:"joined_at"`
	IsBlocked           bool                   `json:"is_blocked" bson:"is_blocked"`
	Demos               map[string]interface{} `json:"demos" bson:"demos"`
	DemoHistory         []string               `json:"demo_history" bson:"demo_history"`
	TnCAccepted         bool                   `json:"tnc_accepted" bson:"tnc_accepted"`
	UnlockedBatches     []string               `json:"unlocked_batches" bson:"unlocked_batches"`
	ReferralCount       int                    `json:"referral_count" bson:"referral_count"`
	TotalInvited        int                    `json:"total_invited" bson:"total_invited"`
	PendingReferral     int64                  `json:"pending_referral,omitempty" bson:"pending_referral,omitempty"`
	PendingBatch        string                 `json:"pending_batch,omitempty" bson:"pending_batch,omitempty"`
	Tier                string                 `json:"tier,omitempty" bson:"tier,omitempty"`
	JoinedBatches       []int64                `json:"joined_batches" bson:"joined_batches"`
	ReferredBy          int64                  `json:"referred_by,omitempty" bson:"referred_by,omitempty"`
	WelcomeBonusClaimed bool                   `json:"welcome_bonus_claimed" bson:"welcome_bonus_claimed"`
	FreeBatchesJoined   []int64                `json:"free_batches_joined" bson:"free_batches_joined"`
	FreeUnlocked        bool                   `json:"free_unlocked" bson:"free_unlocked"`
}

// Batch represents a generic batch (free, paid, special)
type Batch struct {
	ID          int64  `json:"id" bson:"_id"`
	Name        string `json:"name" bson:"name"`
	Category    string `json:"category" bson:"category"`
	Type        string `json:"type" bson:"type"` // free, paid, special
	WelcomeText string `json:"welcome_text,omitempty" bson:"welcome_text,omitempty"`
}

// SupportTopic maps a user to their dedicated forum topic in the support group
type SupportTopic struct {
	UserID        int64 `json:"user_id" bson:"user_id"`
	TopicID       int   `json:"topic_id" bson:"topic_id"`
	MessageThread int   `json:"message_thread_id" bson:"message_thread_id"`
}

// PendingRequest represents a user's join request to a batch channel
type PendingRequest struct {
	UserID    int64     `json:"user_id" bson:"user_id"`
	BatchID   int64     `json:"batch_id" bson:"batch_id"`
	Requested time.Time `json:"requested_at" bson:"requested_at"`
}

// InviteMapping links a generated invite hash to a specific user and batch
type InviteMapping struct {
	Hash    string `json:"hash" bson:"_id"`
	UserID  int64  `json:"user_id" bson:"user_id"`
	BatchID int64  `json:"batch_id" bson:"batch_id"`
	OneTime bool   `json:"one_time" bson:"one_time"`
}

// ScheduledDelete tracks messages that should be deleted after a delay
type ScheduledDelete struct {
	ChatID    int64     `json:"chat_id" bson:"chat_id"`
	MessageID int       `json:"message_id" bson:"message_id"`
	DeleteAt  time.Time `json:"delete_at" bson:"delete_at"`
}

type SupportMessage struct {
	ID             int64     `json:"id"`
	ConversationID int64     `json:"conversation_id"`
	SenderType     string    `json:"sender_type"` // "incoming" or "outgoing"
	SenderName     string    `json:"sender_name"`
	Text           string    `json:"text"`
	Timestamp      time.Time `json:"timestamp"`
	TelegramMsgID  int       `json:"telegram_msg_id"`
}

// BotState holds the full application state for JSON/Memory persistence
type BotState struct {
	AdminIDs         map[int64]struct{}         `json:"admin_ids" bson:"admin_ids"`
	FreeBatches      map[int64]*Batch           `json:"free_batches" bson:"free_batches"`
	PaidBatches      map[int64]*Batch           `json:"paid_batches" bson:"paid_batches"`
	SpecialBatches   map[int64]*Batch           `json:"special_batches" bson:"special_batches"`
	AllChats         map[int64]string           `json:"all_chats" bson:"all_chats"`
	Users            map[int64]*User            `json:"users" bson:"users"`
	BlockedUsers     map[int64]struct{}         `json:"blocked_users" bson:"blocked_users"`
	UserTopics       map[int64]*SupportTopic    `json:"user_topics" bson:"user_topics"`
	PendingRequests  map[string]*PendingRequest `json:"pending_requests" bson:"pending_requests"`
	LinkMap          map[string]*InviteMapping  `json:"link_map" bson:"link_map"`
	CustomWelcomes   map[int64]string           `json:"custom_welcomes" bson:"custom_welcomes"`
	BatchCategories  map[int64]string           `json:"batch_categories" bson:"batch_categories"`
	Categories       []string                   `json:"categories" bson:"categories"`
	ScheduledDeletes []*ScheduledDelete         `json:"scheduled_deletes" bson:"scheduled_deletes"`
	BatchCoins       map[int64]int64            `json:"batch_coins" bson:"batch_coins"`

	WebAdmins   map[string]*WebAdmin   `json:"web_admins" bson:"web_admins"`
	WebSessions map[string]*WebSession `json:"web_sessions" bson:"web_sessions"`
	MessageMap  map[string]string      `json:"message_map" bson:"message_map"`

	NewUsersAllowed bool   `json:"new_users_allowed" bson:"new_users_allowed"`
	FreeLocked      bool   `json:"free_locked" bson:"free_locked"`
	PaidLocked      bool   `json:"paid_locked" bson:"paid_locked"`
	TestBotLocked   bool   `json:"test_bot_locked" bson:"test_bot_locked"`
	MaintenanceMode bool   `json:"maintenance_mode" bson:"maintenance_mode"`
	TestBotLink     string `json:"test_bot_link" bson:"test_bot_link"`
	UserbotSession  string `json:"userbot_session" bson:"userbot_session"`
	UserbotPhone    string `json:"userbot_phone" bson:"userbot_phone"`

	VIPMaterialsLink string `json:"vip_materials_link" bson:"vip_materials_link"`
	VIPStickerID     string `json:"vip_sticker_id" bson:"vip_sticker_id"`
	VIPStickerType   string `json:"vip_sticker_type" bson:"vip_sticker_type"`
}
