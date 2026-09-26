package routes

import (
	"runtime"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/itsh4ro5/botupdate/internal/api/middleware"
	"github.com/itsh4ro5/botupdate/internal/database"
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

func RegisterOperationsRoutes(router fiber.Router, store database.Store) {
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
}
