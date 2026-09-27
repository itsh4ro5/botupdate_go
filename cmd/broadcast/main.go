package main

import (
	"context"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/itsh4ro5/botupdate/internal/database"
	"github.com/joho/godotenv"
	"os"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	mongoURI := os.Getenv("MONGO_URL")
	
	if botToken == "" || mongoURI == "" {
		log.Fatal("Missing required env variables")
	}

	store, err := database.NewMongoStore(context.Background(), mongoURI)
	if err != nil {
		log.Fatal(err)
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal(err)
	}

	state, err := store.Load(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded state. Found %d users.\n", len(state.Users))

	text := "bot still in maintain please wait"
	
	successCount := 0
	failCount := 0
	
	for uid := range state.Users {
		msg := tgbotapi.NewMessage(uid, text)
		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Failed to send to %d: %v", uid, err)
			failCount++
		} else {
			successCount++
		}
		time.Sleep(50 * time.Millisecond)
	}
	
	fmt.Printf("Broadcast complete. Success: %d, Failed: %d\n", successCount, failCount)
}
