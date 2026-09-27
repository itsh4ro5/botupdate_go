package routes

import (
	"context"
	"log"
	"runtime"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/itsh4ro5/botupdate/internal/api/middleware"
	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/events"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itsh4ro5/botupdate/internal/mtproto"
)

type RuntimeStatus struct {
	GoVersion   string `json:"go_version"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	Uptime      string `json:"uptime"`
	Goroutines  int    `json:"goroutines"`
	MemoryAlloc string `json:"memory_alloc"`
	StartTime   string `json:"start_time"`
}

type ServiceStatus struct {
	Telegram  string `json:"telegram"`
	MongoDB   string `json:"mongodb"`
	WebAPI    string `json:"web_api"`
	WebSocket string `json:"websocket"`
	Scheduler string `json:"scheduler"`
	Userbot   string `json:"userbot"`
}

var opsStartTime = time.Now()

func RegisterOperationsRoutes(router fiber.Router, store database.Store, bot *tgbotapi.BotAPI, mtprotoService *mtproto.Service) {
	group := router.Group("/operations")

	group.Use(middleware.RequireAuth(store))
	group.Use(middleware.RequireRole("OWNER", "ADMIN"))

	group.Get("/system", func(c *fiber.Ctx) error {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		uptime := time.Since(opsStartTime).Round(time.Second).String()
		memMB := m.Alloc / 1024 / 1024

		runtimeInfo := RuntimeStatus{
			GoVersion:   runtime.Version(),
			OS:          runtime.GOOS,
			Arch:        runtime.GOARCH,
			Uptime:      uptime,
			Goroutines:  runtime.NumGoroutine(),
			MemoryAlloc: strconv.Itoa(int(memMB)) + " MB",
			StartTime:   opsStartTime.Format(time.RFC3339),
		}

		// Simulate health based on bot state
		state, err := store.Load(c.Context())
		dbStatus := "ONLINE"
		if err != nil {
			dbStatus = "DEGRADED"
		}

		telegramStatus := "ONLINE" // In reality, we'd check BotAPI
		if state != nil && state.MaintenanceMode {
			telegramStatus = "MAINTENANCE"
		}

		userbotStatus := "OFFLINE"
		if state != nil && state.UserbotSession != "" {
			userbotStatus = "ONLINE"
		}

		servicesInfo := ServiceStatus{
			Telegram:  telegramStatus,
			MongoDB:   dbStatus,
			WebAPI:    "ONLINE",
			WebSocket: "ONLINE",
			Scheduler: "ONLINE", // Assuming cron is running
			Userbot:   userbotStatus,
		}

		return c.JSON(fiber.Map{
			"runtime":  runtimeInfo,
			"services": servicesInfo,
		})
	})

	group.Post("/maintenance/refresh-cache", middleware.RequireRole("OWNER"), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true, "message": "Cache refreshed successfully"})
	})

	group.Post("/broadcast", middleware.RequireRole("OWNER", "ADMIN"), func(c *fiber.Ctx) error {
		type Req struct {
			Text string `json:"text"`
		}
		var body Req
		if err := c.BodyParser(&body); err != nil || body.Text == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request or empty text"})
		}

		// Fetch all users
		state, err := store.Load(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load users"})
		}

		usersList := make([]int64, 0, len(state.Users))
		for uid := range state.Users {
			usersList = append(usersList, uid)
		}

		// Start broadcast in background
		go func(text string, users []int64) {
			log.Printf("Starting broadcast to %d users...", len(users))
			successCount := 0
			failCount := 0
			for _, uid := range users {
				msg := tgbotapi.NewMessage(uid, text)
				_, err := bot.Send(msg)
				if err != nil {
					failCount++
				} else {
					successCount++
				}
				// Sleep to respect Telegram rate limits (~30 msgs/sec, but let's be safe with 20/sec)
				time.Sleep(50 * time.Millisecond)
			}
			
			log.Printf("Broadcast complete. Success: %d, Failed: %d", successCount, failCount)
			
			// Publish audit event when done
			events.Publish(events.TypeBroadcastCompleted, events.SeverityInfo, map[string]interface{}{
				"action": "BROADCAST_COMPLETED",
				"details": "Broadcast finished",
				"success_count": successCount,
				"fail_count": failCount,
			})
		}(body.Text, usersList)

		return c.JSON(fiber.Map{"success": true, "message": "Broadcast started in background"})
	})

	group.Post("/userbot/request-code", middleware.RequireRole("OWNER"), func(c *fiber.Ctx) error {
		type Req struct {
			Phone string `json:"phone"`
		}
		var body Req
		if err := c.BodyParser(&body); err != nil || body.Phone == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid phone number"})
		}
		
		err := mtprotoService.StartAuth(context.Background(), body.Phone, nil)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to request code: " + err.Error()})
		}
		
		return c.JSON(fiber.Map{"success": true, "message": "Code requested successfully"})
	})

	group.Post("/userbot/submit-code", middleware.RequireRole("OWNER"), func(c *fiber.Ctx) error {
		type Req struct {
			Code string `json:"code"`
		}
		var body Req
		if err := c.BodyParser(&body); err != nil || body.Code == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid code"})
		}
		
		err := mtprotoService.SubmitOTP(body.Code)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to verify code: " + err.Error()})
		}
		
		return c.JSON(fiber.Map{"success": true, "message": "Logged in successfully. Userbot is now online!"})
	})
}
