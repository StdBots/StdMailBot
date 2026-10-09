package broadcast

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/StdBots/StdMailBot/internal/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Engine manages admin broadcasting with worker pools
type Engine struct {
	api       *tgbotapi.BotAPI
	userRepo  *database.UserRepo
	ownerID   int64
	awaiting  map[int64]bool
	awaitLock sync.RWMutex
}

// NewEngine creates broadcast engine
func NewEngine(api *tgbotapi.BotAPI, userRepo *database.UserRepo, ownerID int64) *Engine {
	return &Engine{
		api:      api,
		userRepo: userRepo,
		ownerID:  ownerID,
		awaiting: make(map[int64]bool),
	}
}

// IsAwaitingBroadcast checks if admin is currently composing a broadcast message
func (e *Engine) IsAwaitingBroadcast(userID int64) bool {
	if userID != e.ownerID {
		return false
	}
	e.awaitLock.RLock()
	defer e.awaitLock.RUnlock()
	return e.awaiting[userID]
}

// HandleBroadcast starts broadcast mode
func (e *Engine) HandleBroadcast(msg *tgbotapi.Message) {
	if msg.From.ID != e.ownerID {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "⛔ Unauthorized: Admin access only.")
		_, _ = e.api.Send(reply)
		return
	}

	if msg.ReplyToMessage != nil {
		e.runBroadcast(msg.Chat.ID, msg.ReplyToMessage)
		return
	}

	e.awaitLock.Lock()
	e.awaiting[msg.From.ID] = true
	e.awaitLock.Unlock()

	reply := tgbotapi.NewMessage(msg.Chat.ID, "📢 <b>Broadcast Mode Active</b>\n\nPlease forward or send the message you wish to broadcast to all bot users.\n<i>(Send /cancel to abort)</i>")
	reply.ParseMode = "HTML"
	_, _ = e.api.Send(reply)
}

// ExecuteBroadcast processes the target broadcast payload
func (e *Engine) ExecuteBroadcast(msg *tgbotapi.Message) {
	e.awaitLock.Lock()
	delete(e.awaiting, msg.From.ID)
	e.awaitLock.Unlock()

	e.runBroadcast(msg.Chat.ID, msg)
}

// CancelBroadcast aborts active broadcast composer
func (e *Engine) CancelBroadcast(userID int64) {
	e.awaitLock.Lock()
	delete(e.awaiting, userID)
	e.awaitLock.Unlock()
}

func (e *Engine) runBroadcast(adminChatID int64, targetMsg *tgbotapi.Message) {
	userIDs, err := e.userRepo.GetAllUserIDs()
	if err != nil || len(userIDs) == 0 {
		reply := tgbotapi.NewMessage(adminChatID, "❌ No users found in database to broadcast.")
		_, _ = e.api.Send(reply)
		return
	}

	statusMsg := tgbotapi.NewMessage(adminChatID, fmt.Sprintf("🚀 Starting broadcast to %d users...", len(userIDs)))
	sentStatus, _ := e.api.Send(statusMsg)

	go func() {
		start := time.Now()
		var success, failed, blocked int64
		var mu sync.Mutex

		jobs := make(chan int64, 100)
		var wg sync.WaitGroup

		workers := 10
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for uid := range jobs {
					copyMsg := tgbotapi.NewCopyMessage(uid, targetMsg.Chat.ID, targetMsg.MessageID)
					_, err := e.api.CopyMessage(copyMsg)

					mu.Lock()
					if err != nil {
						errStr := strings.ToLower(err.Error())
						if strings.Contains(errStr, "blocked") || strings.Contains(errStr, "deactivated") {
							blocked++
						} else if strings.Contains(errStr, "429") || strings.Contains(errStr, "flood") {
							time.Sleep(3 * time.Second)
							failed++
						} else {
							failed++
						}
					} else {
						success++
					}
					mu.Unlock()

					time.Sleep(25 * time.Millisecond) // Pacing to respect Telegram rate limits
				}
			}()
		}

		for _, uid := range userIDs {
			jobs <- uid
		}
		close(jobs)
		wg.Wait()

		duration := time.Since(start).Round(time.Second)
		report := fmt.Sprintf(
			"✅ <b>Broadcast Completed!</b>\n\n"+
				"👥 <b>Total Users:</b> %d\n"+
				"🟢 <b>Delivered:</b> %d\n"+
				"🔴 <b>Failed:</b> %d\n"+
				"🚫 <b>Blocked/Deleted:</b> %d\n"+
				"⏱️ <b>Time Taken:</b> %s",
			len(userIDs), success, failed, blocked, duration,
		)

		if sentStatus.MessageID != 0 {
			edit := tgbotapi.NewEditMessageText(adminChatID, sentStatus.MessageID, report)
			edit.ParseMode = "HTML"
			_, _ = e.api.Send(edit)
		} else {
			msg := tgbotapi.NewMessage(adminChatID, report)
			msg.ParseMode = "HTML"
			_, _ = e.api.Send(msg)
		}

		log.Printf("Broadcast completed: success=%d, failed=%d, blocked=%d in %s", success, failed, blocked, duration)
	}()
}
