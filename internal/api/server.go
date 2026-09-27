package api

import (
	"context"
	"log"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/itsh4ro5/botupdate/internal/api/auth"
	"github.com/itsh4ro5/botupdate/internal/api/routes"
	"github.com/itsh4ro5/botupdate/internal/api/ws"
	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/events"
	"github.com/itsh4ro5/botupdate/internal/models"
	"github.com/itsh4ro5/botupdate/internal/services/admin"
	"github.com/itsh4ro5/botupdate/internal/services/analytics"
	"github.com/itsh4ro5/botupdate/internal/services/audit"
	"github.com/itsh4ro5/botupdate/internal/services/requests"
	"github.com/itsh4ro5/botupdate/internal/services/support"
	"github.com/itsh4ro5/botupdate/internal/services/users"
	"github.com/itsh4ro5/botupdate/internal/telegram"
	"golang.org/x/crypto/bcrypt"
	"github.com/itsh4ro5/botupdate/internal/mtproto"
)

// Server represents the web API server foundation.
type Server struct {
	app   *fiber.App
	store database.Store
}

// NewServer initializes the Fiber application and its middlewares.
func NewServer(ctx context.Context, store database.Store, apiClient *telegram.APIClient, bot *tgbotapi.BotAPI, supportGroupID int64, mtprotoService *mtproto.Service, updates chan tgbotapi.Update) *Server {
	// Initialize default owner if none exist and ENV vars are provided
	state, err := store.Load(context.Background())
	if err == nil && len(state.WebAdmins) == 0 {
		envUser := os.Getenv("WEB_ADMIN_USERNAME")
		envPass := os.Getenv("WEB_ADMIN_PASSWORD")
		if envUser != "" && envPass != "" {
			hash, _ := bcrypt.GenerateFromPassword([]byte(envPass), bcrypt.DefaultCost)
			defaultAdmin := &models.WebAdmin{
				ID:                 "default-owner",
				Username:           envUser,
				PasswordHash:       string(hash),
				Role:               "OWNER",
				CreatedAt:          time.Now(),
				MustChangePassword: true,
			}
			store.SetWebAdmin(context.Background(), envUser, defaultAdmin)
		} else {
			log.Println("WARNING: No web admins exist and WEB_ADMIN_USERNAME / WEB_ADMIN_PASSWORD are not set. Web panel will be inaccessible.")
		}
	}

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	// Security / CORS foundation
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5173", // Vite default dev port
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Csrf-Token",
		AllowCredentials: true,
	}))

	app.Use(csrf.New(csrf.Config{
		KeyLookup:      "header:X-Csrf-Token",
		CookieName:     "csrf_",
		CookieSameSite: "Lax",
		CookieSecure:   false, // Set to true in prod with HTTPS
		CookieHTTPOnly: false,
	}))

	// Security Headers
	app.Use(func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("X-Frame-Options", "DENY")
		return c.Next()
	})

	// Structured logging for API requests
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} - ${latency}\n",
	}))

	// Rate limiting for auth routes
	authLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
	})

	// Start session cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				state, err := store.Load(context.Background())
				if err == nil {
					changed := false
					for sid, sess := range state.WebSessions {
						if time.Now().After(sess.ExpiresAt) {
							delete(state.WebSessions, sid)
							changed = true
						}
					}
					if changed {
						_ = store.Save(context.Background(), state)
					}
				}
			}
		}
	}()

	// Register Routes
	v1 := app.Group("/api/v1")
	routes.RegisterHealthRoutes(v1, store)
	routes.RegisterDashboardRoutes(v1, store)

	userService := users.NewService(store)
	routes.RegisterUserRoutes(v1, store, userService)
	routes.RegisterBatchRoutes(v1, store)

	requestService := requests.NewRequestService(store, events.GetBus(), apiClient, bot)
	routes.RegisterRequestRoutes(v1, store, requestService)

	supportService := support.NewSupportService(store, events.GetBus(), bot, supportGroupID)
	routes.RegisterSupportRoutes(v1, store, supportService)

	analyticsService := analytics.NewAnalyticsService(store)
	routes.RegisterAnalyticsRoutes(v1, store, analyticsService)

	auditService := audit.NewAuditService(events.GetBus())
	adminService := admin.NewAdminService(store, events.GetBus())
	routes.RegisterAdminRoutes(v1, store, adminService, auditService)
	routes.RegisterOperationsRoutes(v1, store, bot, mtprotoService)

	// WebSockets Hub
	hub := ws.NewHub()
	go hub.Run()
	ws.RegisterWebSocketRoutes(v1, store, hub)

	authHandler := auth.NewAuthHandler(store)
	authGroup := v1.Group("/auth")
	authGroup.Post("/login", authLimiter, authHandler.Login)
	authGroup.Post("/logout", authHandler.Logout)
	authGroup.Get("/me", authHandler.Me)
	authGroup.Post("/change-password", authHandler.ChangePassword)

	// Webhook Endpoint for Telegram (No auth required)
	app.Post("/webhook", func(c *fiber.Ctx) error {
		var update tgbotapi.Update
		if err := c.BodyParser(&update); err != nil {
			log.Printf("Webhook parse error: %v", err)
			return c.SendStatus(fiber.StatusBadRequest)
		}
		select {
		case updates <- update:
		default:
			log.Println("WARNING: updates channel full, dropping update")
		}
		return c.SendStatus(fiber.StatusOK)
	})

	// Serve Static Frontend (Vite Build)
	if _, err := os.Stat("./web/dist"); err == nil {
		app.Static("/", "./web/dist")
		// SPA catch-all (must be registered last)
		app.Get("/*", func(c *fiber.Ctx) error {
			return c.SendFile("./web/dist/index.html")
		})
	} else {
		log.Println("WARNING: ./web/dist not found. Frontend will not be served.")
	}

	return &Server{
		app:   app,
		store: store,
	}
}

// Start runs the web server.
func (s *Server) Start(port string) error {
	log.Printf("Starting Web API on port %s", port)
	return s.app.Listen(":" + port)
}

// Shutdown gracefully stops the web server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}
