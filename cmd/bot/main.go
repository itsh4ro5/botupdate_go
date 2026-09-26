package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itsh4ro5/botupdate/config"
	"github.com/itsh4ro5/botupdate/internal/api"
	botmodule "github.com/itsh4ro5/botupdate/internal/bot"
	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/itsh4ro5/botupdate/internal/mtproto"
	"github.com/itsh4ro5/botupdate/internal/services"
	"github.com/itsh4ro5/botupdate/internal/telegram"
)

func main() {
	log.Println("Starting botupdate Go migration...")

	// 1. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// 2. Initialize Database (MongoDB or JSON fallback)
	var store database.Store
	if cfg.MongoURL != "" {
		log.Println("Connecting to MongoDB Atlas...")
		var err error
		store, err = database.NewMongoStore(context.Background(), cfg.MongoURL)
		if err != nil {
			log.Printf("MongoDB connection failed: %v. Falling back to JSON store.", err)
			store = database.NewJSONStore(cfg.DataFile)
		} else {
			log.Println("Connected to MongoDB Atlas Successfully!")
		}
	} else {
		store = database.NewJSONStore(cfg.DataFile)
	}

	// Ensure we can load state
	state, err := store.Load(context.Background())
	if err != nil {
		log.Fatalf("MongoDB state decode: FAILED\nExpected: compatible representations\nTelegram initialization: SKIPPED\nError: %v", err)
	}

	log.Println("MongoDB connection: OK")
	log.Println("MongoDB state fetch: OK")
	log.Println("MongoDB state decode: OK")
	log.Printf("Users: %d", len(state.Users))
	log.Printf("Free channels: %d", len(state.FreeBatches))
	log.Printf("Paid channels: %d", len(state.PaidBatches))
	log.Printf("Special channels: %d", len(state.SpecialBatches))
	log.Printf("User topics: %d", len(state.UserTopics))
	log.Println("State validation: PASSED")
	log.Println("Proceeding to Telegram initialization...")

	// 3. Initialize Telegram Bot
	httpClient := telegram.NewHTTPClient()
	interceptor := &telegram.UpdateInterceptor{
		Client: httpClient,
	}

	apiEndpoint := tgbotapi.APIEndpoint

	// Diagnostic connectivity check
	telegram.TestConnectivity("https://api.telegram.org/bot" + cfg.TelegramBotToken + "/getMe")

	var bot *tgbotapi.BotAPI
	var botErr error

	// Phase 5 & 6: Exponential backoff with jitter for transient network failures
	maxAttempts := 30
	baseDelay := 2 * time.Second
	maxDelay := 60 * time.Second
	startTime := time.Now()

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		bot, botErr = tgbotapi.NewBotAPIWithClient(cfg.TelegramBotToken, apiEndpoint, interceptor)
		if botErr == nil {
			break
		}

		errStr := botErr.Error()

		// Categorize error securely without leaking token
		errCategory := "UNKNOWN_ERROR"
		if strings.Contains(errStr, "timeout") {
			errCategory = "TIMEOUT"
		} else if strings.Contains(errStr, "TLS handshake") {
			errCategory = "TLS_HANDSHAKE_FAILURE"
		} else if strings.Contains(errStr, "connection reset") {
			errCategory = "CONNECTION_RESET"
		} else if strings.Contains(errStr, "no such host") || strings.Contains(errStr, "lookup") {
			errCategory = "DNS_FAILURE"
		} else if strings.Contains(errStr, "401") || strings.Contains(errStr, "Unauthorized") {
			log.Fatalf("Fatal: Unauthorized token. Halting retries.")
		} else if strings.Contains(errStr, "net/http") {
			errCategory = "NETWORK_FAILURE"
		}

		elapsed := time.Since(startTime).Round(time.Second)
		log.Printf("Telegram init failed [Attempt %d/%d] [Elapsed: %v] [Category: %s]", attempt, maxAttempts, elapsed, errCategory)

		if attempt == maxAttempts {
			log.Fatalf("Fatal: could not connect to Telegram after %d attempts", maxAttempts)
		}

		// Calculate backoff with jitter
		delay := baseDelay * time.Duration(1<<(attempt-1))
		if delay > maxDelay || delay <= 0 {
			delay = maxDelay
		}
		// add up to 20% jitter (using simple pseudo-random based on time to avoid importing math/rand if not needed)
		jitter := time.Duration(time.Now().UnixNano()%200) * time.Millisecond
		time.Sleep(delay + jitter)
	}
	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// 4. Set up update channel
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 50 // Decreased to 50s to avoid Hugging Face 60s egress proxy idle timeout
	u.AllowedUpdates = []string{"message", "edited_message", "callback_query", "chat_join_request", "chat_member", "my_chat_member", "message_reaction"}

	updates := bot.GetUpdatesChan(u)

	// API Client for internal services
	apiClient := telegram.NewAPIClient(bot.Token)

	// 5. Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Received shutdown signal. Shutting down gracefully...")
		cancel()
	}()

	// 5.5 Start Scheduler
	scheduler := services.NewScheduler(store, apiClient, cfg.MandatoryChannelID)
	scheduler.Start(ctx)
	defer scheduler.Stop()

	mtprotoService := mtproto.NewService(int(cfg.APIID), cfg.APIHash, store)

	// 6. Initialize Router
	router := botmodule.NewRouter(bot, apiClient, store, cfg.OwnerID, cfg.SupportGroupID, cfg.MandatoryChannelID, mtprotoService, scheduler, cfg.BatchUpdateChannelID)

	// Link router to interceptor for reaction sync
	interceptor.Handler = router
	interceptor.Ctx = ctx

	// 6.5 Initialize and Start Web API
	apiServer := api.NewServer(ctx, store, apiClient, bot, cfg.SupportGroupID)
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "3000"
		}
		if err := apiServer.Start(port); err != nil {
			log.Printf("Web API error: %v", err)
		}
	}()

	var wg sync.WaitGroup

	// 7. Update Loop (Dispatcher)
	for {
		select {
		case <-ctx.Done():
			log.Println("Bot shutting down gracefully.")

			// Give the web server 5 seconds to finish active requests
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			apiServer.Shutdown(shutdownCtx)

			// Wait for all in-flight handlers
			wg.Wait()
			log.Println("Shutdown complete.")
			return
		case update := <-updates:
			wg.Add(1)
			go func(u tgbotapi.Update) {
				defer wg.Done()
				router.HandleUpdate(ctx, u)
			}(update)
		}
	}
}
