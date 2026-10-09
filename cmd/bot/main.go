/*
 * Copyright (C) 2024-2026 STD DEEPANSHU <https://deepanshu.in>
 * STD BOTS - Telegram: @STD_DEEPANSHU, @STDBOTS
 *
 * This file is part of StdMailBot.
 * Licensed under the GNU Affero General Public License v3 (AGPL-3.0).
 */

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/StdBots/StdMailBot/internal/bot"
	"github.com/StdBots/StdMailBot/internal/config"
	"github.com/StdBots/StdMailBot/internal/credit"
	"github.com/StdBots/StdMailBot/internal/database"
)

func main() {
	log.Println("🚀 Starting StdMailBot Engine (v2.0.0)...")

	// 1. Load Configurations
	cfg := config.Load()

	// 2. Run Background Credit Integrity Monitor
	credit.CheckCreditPeriodically(30*time.Minute, func() bool {
		intact, _ := credit.VerifyIntegrity()
		return intact
	})

	// 3. Connect to MongoDB
	db, err := database.NewMongoDB(cfg.MongoURI, cfg.DBName)
	if err != nil {
		log.Fatalf("❌ MongoDB connection error: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("⚠️ Error closing MongoDB connection: %v", err)
		}
	}()

	// 4. Initialize Telegram Bot Instance
	botInstance, err := bot.New(cfg, db)
	if err != nil {
		log.Fatalf("❌ Failed to initialize Telegram Bot: %v", err)
	}

	// 5. Setup Graceful Shutdown Handler
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("🛑 Received termination signal (%v). Initiating graceful shutdown...", sig)
		cancel()
	}()

	// 6. Run Bot Polling Loop
	if err := botInstance.Start(ctx); err != nil {
		log.Fatalf("❌ Bot engine stopped with error: %v", err)
	}

	log.Println("👋 StdMailBot has shut down cleanly. Goodbye!")
}
