package events

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"sync"
	"time"
)

// generateID creates a short hex ID for events.
func generateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return time.Now().Format("20060102150405")
	}
	return hex.EncodeToString(b)
}

// Subscriber is a channel that receives events.
type Subscriber chan *Event

type Bus struct {
	mu          sync.RWMutex
	subscribers map[Subscriber]struct{}
}

// Global default bus instance.
var defaultBus = &Bus{
	subscribers: make(map[Subscriber]struct{}),
}

func GetBus() *Bus {
	return defaultBus
}

func (b *Bus) Subscribe() Subscriber {
	// 100 capacity prevents slow clients from instantly blocking
	sub := make(Subscriber, 100)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[sub] = struct{}{}
	return sub
}

func (b *Bus) Unsubscribe(sub Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.subscribers[sub]; ok {
		delete(b.subscribers, sub)
		close(sub)
	}
}

func (b *Bus) Publish(eventType EventType, severity EventSeverity, data map[string]interface{}) {
	event := &Event{
		ID:        generateID(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Severity:  severity,
		Data:      data,
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for sub := range b.subscribers {
		select {
		case sub <- event:
		default:
			// Non-blocking. If a subscriber channel is full (slow client), drop the event.
			log.Printf("EventBus: dropped event %s for a slow subscriber", eventType)
		}
	}
}

// Helper to quickly publish to default bus
func Publish(eventType EventType, severity EventSeverity, data map[string]interface{}) {
	defaultBus.Publish(eventType, severity, data)
}
