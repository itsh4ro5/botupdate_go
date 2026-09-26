package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds the environment configuration for the bot
type Config struct {
	TelegramBotToken       string
	APIID                  int64
	APIHash                string
	SessionString          string
	OwnerID                int64
	SupportGroupID         int64
	MandatoryChannelID     int64
	LogChannelID           int64
	BatchUpdateChannelID   int64
	BatchUpdateChannelLink string
	MandatoryChannelLink   string
	MongoURL               string
	DataFile               string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it, relying on environment variables")
	}

	apiID, _ := strconv.ParseInt(os.Getenv("API_ID"), 10, 64)
	ownerID, _ := strconv.ParseInt(os.Getenv("OWNER_ID"), 10, 64)
	supportGroupID, _ := strconv.ParseInt(os.Getenv("SUPPORT_GROUP_ID"), 10, 64)
	mandatoryChannelID, _ := strconv.ParseInt(os.Getenv("MANDATORY_CHANNEL_ID"), 10, 64)
	logChannelID, _ := strconv.ParseInt(os.Getenv("LOG_CHANNEL_ID"), 10, 64)
	batchUpdateChannelID, _ := strconv.ParseInt(os.Getenv("BATCH_UPDATE_CHANNEL_ID"), 10, 64)

	cfg := &Config{
		TelegramBotToken:       os.Getenv("TELEGRAM_BOT_TOKEN"),
		APIID:                  apiID,
		APIHash:                os.Getenv("API_HASH"),
		SessionString:          os.Getenv("SESSION_STRING"),
		OwnerID:                ownerID,
		SupportGroupID:         supportGroupID,
		MandatoryChannelID:     mandatoryChannelID,
		LogChannelID:           logChannelID,
		BatchUpdateChannelID:   batchUpdateChannelID,
		BatchUpdateChannelLink: os.Getenv("BATCH_UPDATE_CHANNEL_LINK"),
		MandatoryChannelLink:   os.Getenv("MANDATORY_CHANNEL_LINK"),
		MongoURL:               os.Getenv("MONGO_URL"),
		DataFile:               os.Getenv("DATA_FILE"),
	}

	if cfg.DataFile == "" {
		cfg.DataFile = "bot_data.json"
	}

	return cfg, Validate(cfg)
}

// Validate checks for required configuration values
func Validate(cfg *Config) error {
	if cfg.TelegramBotToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}
	if cfg.OwnerID == 0 {
		return fmt.Errorf("OWNER_ID is required")
	}
	// Add more validation as required by business logic
	return nil
}
