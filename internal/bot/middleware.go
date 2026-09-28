package bot

import (
	"fmt"
	"log"

	"github.com/StdBots/StdMailBot/internal/config"
	"github.com/StdBots/StdMailBot/internal/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Middleware handles authentication, user tracking, and channel subscription verification
type Middleware struct {
	cfg      *config.Config
	userRepo *database.UserRepo
	bot      *tgbotapi.BotAPI
}

// NewMiddleware creates new Middleware instance
func NewMiddleware(cfg *config.Config, userRepo *database.UserRepo, bot *tgbotapi.BotAPI) *Middleware {
	return &Middleware{
		cfg:      cfg,
		userRepo: userRepo,
		bot:      bot,
	}
}

// TrackUser records user interaction in MongoDB
func (m *Middleware) TrackUser(from *tgbotapi.User) {
	if from == nil {
		return
	}
	go func() {
		err := m.userRepo.RegisterOrUpdate(from.ID, from.UserName, from.FirstName, from.LastName)
		if err != nil {
			log.Printf("Error tracking user %d: %v", from.ID, err)
		}
	}()
}

// IsOwner checks if user ID matches the owner
func (m *Middleware) IsOwner(userID int64) bool {
	return userID == m.cfg.OwnerID
}

// CheckForceSub verifies if user is subscribed to the mandatory channel
func (m *Middleware) CheckForceSub(userID int64) (bool, error) {
	if m.cfg.ForceSubChannel == "" {
		return true, nil
	}

	chatConfig := tgbotapi.ChatInfoConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			SuperGroupUsername: "@" + m.cfg.ForceSubChannel,
			UserID:             userID,
		},
	}

	member, err := m.bot.GetChatMember(chatConfig)
	if err != nil {
		return true, nil // If channel doesn't exist or bot not admin, fail open
	}

	status := member.Status
	if status == "creator" || status == "administrator" || status == "member" || status == "restricted" {
		return true, nil
	}

	return false, nil
}

// GetForceSubMarkup returns the keyboard directing user to join
func (m *Middleware) GetForceSubMarkup() tgbotapi.InlineKeyboardMarkup {
	channelURL := fmt.Sprintf("https://t.me/%s", m.cfg.ForceSubChannel)
	btnChannel := tgbotapi.NewInlineKeyboardButtonURL("📢 Join Channel", channelURL)
	btnRetry := tgbotapi.NewInlineKeyboardButtonData("🔄 Joined & Try Again", "fsub:retry")

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(btnChannel),
		tgbotapi.NewInlineKeyboardRow(btnRetry),
	)
}
