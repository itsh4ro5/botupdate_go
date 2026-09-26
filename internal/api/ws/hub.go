package ws

import (
	"encoding/json"
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/itsh4ro5/botupdate/internal/events"
)

type Client struct {
	Conn *websocket.Conn
	Send chan []byte
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *events.Event
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan *events.Event, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Run() {
	busSub := events.GetBus().Subscribe()
	defer events.GetBus().Unsubscribe(busSub)

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
		case event := <-busSub:
			msg := map[string]interface{}{
				"type":  "event",
				"event": event,
			}
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}

			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- data:
				default:
					// Connection is dead/slow. Drop client to prevent blocking
					go func(c *Client) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}
