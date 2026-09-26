package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
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
		log.Fatalf("Failed to load state: %v", err)
	}
	log.Printf("Loaded state with %d users and %d free batches", len(state.Users), len(state.FreeBatches))

	// 3. Initialize Telegram Bot
	interceptor := &telegram.UpdateInterceptor{
		Client: telegram.NewHTTPClient(), // Use our robust client
	}

	apiEndpoint := tgbotapi.APIEndpoint
	if cfg.CustomBaseURL != "" {
		apiEndpoint = cfg.CustomBaseURL + "/bot%s/%s"
	}
	
	// Diagnostic connectivity check
	telegram.TestConnectivity(fmt.Sprintf(apiEndpoint, cfg.TelegramBotToken, "getMe"))

	var bot *tgbotapi.BotAPI
	var botErr error

	// Exponential backoff for initial connection to prevent crash loops
	for attempts := 1; attempts <= 5; attempts++ {
		bot, botErr = tgbotapi.NewBotAPIWithClient(cfg.TelegramBotToken, apiEndpoint, interceptor)
		if botErr == nil {
			break
		}
		log.Printf("Failed to create Telegram bot (attempt %d/5): %v", attempts, botErr)
		if attempts < 5 {
			time.Sleep(time.Duration(attempts*attempts) * time.Second)
		}
	}

	if botErr != nil {
		log.Fatalf("Fatal: could not connect to Telegram after 5 attempts: %v", botErr)
	}
	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// 4. Set up update channel
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "edited_message", "callback_query", "chat_join_request", "chat_member", "my_chat_member", "message_reaction"}

	updates := bot.GetUpdatesChan(u)

	// API Client for internal services
	apiClient := telegram.NewAPIClient(bot.Token, cfg.CustomBaseURL)

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
