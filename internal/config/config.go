package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config represents runtime configurations for StdMailBot
type Config struct {
	BotToken        string
	OwnerID         int64
	MongoURI        string
	DBName          string
	ForceSubChannel string
	LogChannelID    int64
	DefaultEngine   string
	Env             string
}

// Load loads configurations from environment variables or .env file
func Load() *Config {
	_ = godotenv.Load()

	botToken := strings.TrimSpace(os.Getenv("BOT_TOKEN"))
	if botToken == "" {
		log.Panic("FATAL: BOT_TOKEN is required in environment variables or .env file")
	}

	ownerIDStr := strings.TrimSpace(os.Getenv("OWNER_ID"))
	if ownerIDStr == "" {
		ownerIDStr = "7394590844"
	}
	ownerID, err := strconv.ParseInt(ownerIDStr, 10, 64)
	if err != nil {
		log.Panicf("FATAL: Invalid OWNER_ID '%s': %v", ownerIDStr, err)
	}

	mongoURI := strings.TrimSpace(os.Getenv("MONGO_URI"))
	if mongoURI == "" {
		mongoURI = strings.TrimSpace(os.Getenv("DB_URL"))
	}
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017/stdmailbot"
	}

	dbName := strings.TrimSpace(os.Getenv("DB_NAME"))
	if dbName == "" {
		dbName = "stdmailbot"
	}

	forceSubChannel := strings.TrimPrefix(strings.TrimSpace(os.Getenv("FORCE_SUB_CHANNEL")), "@")

	var logChannelID int64
	logChannelStr := strings.TrimSpace(os.Getenv("LOG_CHANNEL_ID"))
	if logChannelStr != "" {
		logChannelID, _ = strconv.ParseInt(logChannelStr, 10, 64)
	}

	defaultEngine := strings.ToLower(strings.TrimSpace(os.Getenv("DEFAULT_MAIL_ENGINE")))
	if defaultEngine == "" {
		defaultEngine = "tempmail"
	}

	env := strings.TrimSpace(os.Getenv("ENV"))
	if env == "" {
		env = "production"
	}

	return &Config{
		BotToken:        botToken,
		OwnerID:         ownerID,
		MongoURI:        mongoURI,
		DBName:          dbName,
		ForceSubChannel: forceSubChannel,
		LogChannelID:    logChannelID,
		DefaultEngine:   defaultEngine,
		Env:             env,
	}
}
