package events

import "time"

type EventType string

const (
	TypeUserJoined         EventType = "USER_JOINED"
	TypeUserUpdated        EventType = "USER_UPDATED"
	TypeUserActivity       EventType = "USER_ACTIVITY"
	TypeSupportMessage     EventType = "SUPPORT_MESSAGE"
	TypeAccessRequest      EventType = "ACCESS_REQUEST"
	TypeAccessGranted      EventType = "ACCESS_GRANTED"
	TypeAccessRevoked      EventType = "ACCESS_REVOKED"
	TypeRequestApproved    EventType = "REQUEST_APPROVED"
	TypeRequestRejected    EventType = "REQUEST_REJECTED"
	TypeBatchScanStarted   EventType = "BATCH_SCAN_STARTED"
	TypeBatchScanCompleted EventType = "BATCH_SCAN_COMPLETED"
	TypeBroadcastStarted   EventType = "BROADCAST_STARTED"
	TypeBroadcastCompleted EventType = "BROADCAST_COMPLETED"
	TypeUserbotStatus      EventType = "USERBOT_STATUS"
	TypeSchedulerJob       EventType = "SCHEDULER_JOB"
	TypeSystemError        EventType = "SYSTEM_ERROR"
	TypeSystemInfo         EventType = "SYSTEM_INFO"
)

type EventSeverity string

const (
	SeverityInfo    EventSeverity = "info"
	SeverityWarning EventSeverity = "warning"
	SeverityError   EventSeverity = "error"
)

type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Severity  EventSeverity          `json:"severity"`
	Data      map[string]interface{} `json:"data"`
}
