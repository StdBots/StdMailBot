package bot

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/StdBots/StdMailBot/internal/analytics"
	"github.com/StdBots/StdMailBot/internal/broadcast"
	"github.com/StdBots/StdMailBot/internal/config"
	"github.com/StdBots/StdMailBot/internal/credit"
	"github.com/StdBots/StdMailBot/internal/database"
	"github.com/StdBots/StdMailBot/internal/mail"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot orchestrates telegram polling and handlers
type Bot struct {
	api             *tgbotapi.BotAPI
	cfg             *config.Config
	db              *database.MongoDB
	mailManager     *mail.Manager
	mailRepo        *database.MailRepo
	userRepo        *database.UserRepo
	cache           *database.MemoryCache
	middleware      *Middleware
	handler         *Handler
	callbackHandler *CallbackHandler
	broadcastEngine *broadcast.Engine
	analyticsEngine *analytics.Engine
}

// New creates and wires up all dependencies for the bot
func New(cfg *config.Config, db *database.MongoDB) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, err
	}

	if cfg.Env == "development" {
		api.Debug = true
	}

	mailManager := mail.NewManager(cfg.DefaultEngine)
	mailRepo := database.NewMailRepo(db)
	userRepo := database.NewUserRepo(db)
	cache := database.NewMemoryCache(10 * time.Minute)
	middleware := NewMiddleware(cfg, userRepo, api)

	handler := NewHandler(api, mailManager, mailRepo, userRepo, cache, middleware)
	callbackHandler := NewCallbackHandler(api, mailManager, mailRepo, userRepo, cache, middleware, handler)
	broadcastEngine := broadcast.NewEngine(api, userRepo, cfg.OwnerID)
	analyticsEngine := analytics.NewEngine(api, mailRepo, userRepo, cfg.OwnerID)

	return &Bot{
		api:             api,
		cfg:             cfg,
		db:              db,
		mailManager:     mailManager,
		mailRepo:        mailRepo,
		userRepo:        userRepo,
		cache:           cache,
		middleware:      middleware,
		handler:         handler,
		callbackHandler: callbackHandler,
		broadcastEngine: broadcastEngine,
		analyticsEngine: analyticsEngine,
	}, nil
}

// Start begins polling for updates
func (b *Bot) Start(ctx context.Context) error {
	log.Printf("🤖 StdMailBot authorized on account @%s (ID: %d)", b.api.Self.UserName, b.api.Self.ID)

	// Verify Credit Integrity
	intact, tampered := credit.VerifyIntegrity()
	if !intact {
		log.Printf("[SECURITY WARNING] Integrity violation detected: %v", tampered)
	}
	credit.ReportForkStatus(b.api.Self.UserName, intact)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Shutting down bot update loop...")
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			go b.processUpdate(update)
		}
	}
}

func (b *Bot) processUpdate(update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC RECOVERED] in update %d: %v", update.UpdateID, r)
		}
	}()

	// 1. Handle Callback Queries
	if update.CallbackQuery != nil {
		b.callbackHandler.Handle(update.CallbackQuery)
		return
	}

	// 2. Handle Messages
	if update.Message != nil {
		msg := update.Message

		if msg.IsCommand() {
			cmd := strings.ToLower(msg.Command())
			switch cmd {
			case "start":
				b.handler.HandleStart(msg)
			case "getmail", "mail", "new":
				b.handler.HandleGetMail(msg)
			case "inbox", "check":
				b.handler.HandleInbox(msg)
			case "watch":
				// Trigger watch mode via message
				cb := &tgbotapi.CallbackQuery{
					ID:      "cmd_watch",
					From:    msg.From,
					Message: msg,
				}
				b.callbackHandler.handleToggleWatch(cb)
			case "changemail", "change":
				b.handler.HandleChangeMail(msg)
			case "deletemail", "delete":
				b.handler.HandleDeleteMail(msg)
			case "help":
				b.handler.HandleHelp(msg)
			case "stats", "std":
				b.analyticsEngine.HandleStats(msg)
			case "broadcast":
				b.broadcastEngine.HandleBroadcast(msg)
			case "cancel":
				b.broadcastEngine.CancelBroadcast(msg.From.ID)
				reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Operation cancelled.")
				_, _ = b.api.Send(reply)
			default:
				reply := tgbotapi.NewMessage(msg.Chat.ID, "❓ Unknown command. Send /help to view available commands.")
				_, _ = b.api.Send(reply)
			}
			return
		}

		// Check if broadcast is waiting for content
		if b.broadcastEngine.IsAwaitingBroadcast(msg.From.ID) {
			b.broadcastEngine.ExecuteBroadcast(msg)
			return
		}
	}
}
