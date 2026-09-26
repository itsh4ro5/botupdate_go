package ws

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/itsh4ro5/botupdate/internal/database"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

func (c *Client) readPump(h *Hub) {
	defer func() {
		h.unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		// Just read to keep connection alive and detect close
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
				log.Printf("WS debug (unexpected): %v", err)
			}
			break
		}
	}
}

func (c *Client) writePump(h *Hub) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func RegisterWebSocketRoutes(router fiber.Router, store database.Store, hub *Hub) {
	// Middleware to check if upgrade is requested
	router.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			// Require authentication for websocket (assuming a middleware upstream has attached the session,
			// or we can read the session cookie directly here).
			// Fiber v2 allows getting cookies
			sessionCookie := c.Cookies("session_id")
			if sessionCookie == "" {
				return fiber.ErrUnauthorized
			}

			state, err := store.Load(c.Context())
			if err != nil || state == nil {
				return fiber.ErrUnauthorized
			}

			sess, ok := state.WebSessions[sessionCookie]
			if !ok || time.Now().After(sess.ExpiresAt) || (sess.Role != "OWNER" && sess.Role != "ADMIN") {
				return fiber.ErrUnauthorized
			}

			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	router.Get("/ws", websocket.New(func(c *websocket.Conn) {
		client := &Client{
			Conn: c,
			Send: make(chan []byte, 256),
		}
		hub.register <- client

		// Send initial connected event
		client.Send <- []byte(`{"type":"connected"}`)

		go client.writePump(hub)
		client.readPump(hub)
	}))
}
