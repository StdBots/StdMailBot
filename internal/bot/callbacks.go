package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/StdBots/StdMailBot/internal/credit"
	"github.com/StdBots/StdMailBot/internal/database"
	"github.com/StdBots/StdMailBot/internal/mail"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CallbackHandler processes all inline keyboard callback queries
type CallbackHandler struct {
	bot         *tgbotapi.BotAPI
	mailManager *mail.Manager
	mailRepo    *database.MailRepo
	userRepo    *database.UserRepo
	cache       *database.MemoryCache
	middleware  *Middleware
	handler     *Handler
}

// NewCallbackHandler initializes the callback handler
func NewCallbackHandler(bot *tgbotapi.BotAPI, mailManager *mail.Manager, mailRepo *database.MailRepo, userRepo *database.UserRepo, cache *database.MemoryCache, middleware *Middleware, handler *Handler) *CallbackHandler {
	return &CallbackHandler{
		bot:         bot,
		mailManager: mailManager,
		mailRepo:    mailRepo,
		userRepo:    userRepo,
		cache:       cache,
		middleware:  middleware,
		handler:     handler,
	}
}

// Handle processes incoming callback query
func (c *CallbackHandler) Handle(query *tgbotapi.CallbackQuery) {
	c.middleware.TrackUser(query.From)
	data := query.Data

	switch {
	case data == "inbox:check":
		c.handleCheckInbox(query)
	case data == "mail:generate":
		c.handleGenerateMail(query)
	case data == "mail:change":
		c.handleChangeMail(query)
	case data == "mail:delete":
		c.handleDeleteMail(query)
	case data == "mail:watch":
		c.handleToggleWatch(query)
	case strings.HasPrefix(data, "copy:otp:"):
		otp := strings.TrimPrefix(data, "copy:otp:")
		c.answer(query.ID, fmt.Sprintf("🔑 OTP: %s", otp), true)
	case data == "fsub:retry":
		isMember, _ := c.middleware.CheckForceSub(query.From.ID)
		if isMember {
			c.answer(query.ID, "✅ Subscription verified! Generating your email...", false)
			c.handleGenerateMail(query)
		} else {
			c.answer(query.ID, "❌ You have not joined our channel yet!", true)
		}
	default:
		c.answer(query.ID, "Action not recognized.", false)
	}
}

func (c *CallbackHandler) handleCheckInbox(query *tgbotapi.CallbackQuery) {
	session, err := c.mailRepo.GetActiveMail(query.From.ID)
	if err != nil || session == nil || session.Email == "" {
		c.answer(query.ID, "❌ Pehle email generate karo!", true)
		return
	}

	c.answer(query.ID, "⏳ Checking inbox...", false)

	messages, err := c.mailManager.GetMessages(session)
	if err != nil {
		c.answer(query.ID, "❌ Error contacting mail server.", true)
		return
	}

	msgID := 0
	if query.Message != nil {
		msgID = query.Message.MessageID
		c.handler.renderInboxView(query.Message.Chat.ID, msgID, session, messages)
	}
}

func (c *CallbackHandler) handleGenerateMail(query *tgbotapi.CallbackQuery) {
	isMember, _ := c.middleware.CheckForceSub(query.From.ID)
	if !isMember {
		c.answer(query.ID, fmt.Sprintf("⚠️ Pehle @%s join karo!", c.middleware.cfg.ForceSubChannel), true)
		return
	}

	c.answer(query.ID, "⏳ Generating fresh email...", false)

	session, err := c.mailManager.GenerateEmail()
	if err != nil {
		c.answer(query.ID, "❌ Server error. Try again.", true)
		return
	}

	_ = c.mailRepo.SaveActiveMail(query.From.ID, session)

	if query.Message != nil {
		c.handler.sendActiveMailboxView(query.Message.Chat.ID, session, "🎉 <b>Your New Disposable Email is Ready:</b>")
	}
}

func (c *CallbackHandler) handleChangeMail(query *tgbotapi.CallbackQuery) {
	c.answer(query.ID, "🔄 Generating new email...", false)

	oldSession, _ := c.mailRepo.DeleteActiveMail(query.From.ID)
	if oldSession != nil {
		_ = c.mailManager.DeleteEmail(oldSession)
	}

	newSession, err := c.mailManager.GenerateEmail()
	if err != nil {
		c.answer(query.ID, "❌ Error generating new email.", true)
		return
	}

	_ = c.mailRepo.SaveActiveMail(query.From.ID, newSession)

	if query.Message != nil {
		c.handler.sendActiveMailboxView(query.Message.Chat.ID, newSession, "🔄 <b>Your Email Has Been Changed:</b>")
	}
}

func (c *CallbackHandler) handleDeleteMail(query *tgbotapi.CallbackQuery) {
	oldSession, err := c.mailRepo.DeleteActiveMail(query.From.ID)
	if err != nil || oldSession == nil {
		c.answer(query.ID, "No active mailbox found.", true)
		return
	}

	_ = c.mailManager.DeleteEmail(oldSession)
	c.answer(query.ID, "🗑️ Mailbox destroyed!", true)

	if query.Message != nil {
		reply := tgbotapi.NewMessage(query.Message.Chat.ID, "🗑️ <b>Mailbox Deleted.</b>\nYour temporary email address was deleted permanently.")
		reply.ParseMode = "HTML"
		btnGen := tgbotapi.NewInlineKeyboardButtonData("🎲 Generate New Email", "mail:generate")
		reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btnGen))
		_, _ = c.bot.Send(reply)
	}
}

// handleToggleWatch starts a background goroutine to poll inbox for 5 minutes and auto-alert the user!
func (c *CallbackHandler) handleToggleWatch(query *tgbotapi.CallbackQuery) {
	session, err := c.mailRepo.GetActiveMail(query.From.ID)
	if err != nil || session == nil || session.Email == "" {
		c.answer(query.ID, "❌ No active email to watch.", true)
		return
	}

	watchKey := fmt.Sprintf("watch:%d", query.From.ID)
	if _, ok := c.cache.Get(watchKey); ok {
		c.answer(query.ID, "ℹ️ Watch Mode is already active for this mailbox!", true)
		return
	}

	c.cache.Set(watchKey, true, 5*time.Minute)
	c.answer(query.ID, "🔔 Watch Mode Activated! (Will notify you as soon as an email arrives)", true)

	chatID := query.Message.Chat.ID
	userID := query.From.ID

	// Background Auto-Watcher Goroutine
	go func() {
		seenIDs := make(map[string]bool)

		// Pre-populate with currently existing message IDs
		if existing, err := c.mailManager.GetMessages(session); err == nil {
			for _, m := range existing {
				seenIDs[m.ID] = true
			}
		}

		ticker := time.NewTicker(6 * time.Second)
		defer ticker.Stop()

		timeout := time.After(5 * time.Minute)

		for {
			select {
			case <-timeout:
				c.cache.Delete(watchKey)
				notify := tgbotapi.NewMessage(chatID, "🔕 <i>Watch Mode ended (5 minutes limit reached). Click '🔔 Watch' again if you are still waiting.</i>")
				notify.ParseMode = "HTML"
				_, _ = c.bot.Send(notify)
				return
			case <-ticker.C:
				currentSession, err := c.mailRepo.GetActiveMail(userID)
				if err != nil || currentSession == nil || currentSession.Email != session.Email {
					// User changed or deleted email
					c.cache.Delete(watchKey)
					return
				}

				messages, err := c.mailManager.GetMessages(session)
				if err != nil {
					continue
				}

				for _, m := range messages {
					if !seenIDs[m.ID] {
						seenIDs[m.ID] = true

						// NEW INCOMING EMAIL DETECTED!
						var sb strings.Builder
						sb.WriteString("🔔 <b>NEW EMAIL RECEIVED!</b>\n\n")
						sb.WriteString(fmt.Sprintf("📬 <b>To:</b> <code>%s</code>\n", session.Email))
						sb.WriteString(fmt.Sprintf("📌 <b>From:</b> <code>%s</code>\n", escapeHTML(m.From)))
						sb.WriteString(fmt.Sprintf("🔰 <b>Subject:</b> <b>%s</b>\n", escapeHTML(m.Subject)))

						if m.OTP != "" {
							sb.WriteString(fmt.Sprintf("\n🔑 <b>OTP Code:</b> <code>%s</code>\n", m.OTP))
						}

						if len(m.MagicLinks) > 0 {
							sb.WriteString("\n🔑 <b>Login / Verification Links:</b>\n")
							for _, l := range m.MagicLinks {
								sb.WriteString(fmt.Sprintf("👉 <a href=\"%s\">Click Here to Verify</a>\n", l))
							}
						}

						if m.Body != "" {
							preview := m.Body
							if len(preview) > 500 {
								preview = preview[:500] + "..."
							}
							sb.WriteString(fmt.Sprintf("\n💬 <b>Content:</b>\n<i>%s</i>\n", escapeHTML(preview)))
						}

						sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━\n")
						sb.WriteString(credit.GetFooter())

						msg := tgbotapi.NewMessage(chatID, credit.GetWatermarked(sb.String()))
						msg.ParseMode = "HTML"

						var rows [][]tgbotapi.InlineKeyboardButton
						if m.OTP != "" {
							rows = append(rows, tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🔑 Copy OTP: %s", m.OTP), fmt.Sprintf("copy:otp:%s", m.OTP)),
							))
						}
						rows = append(rows, tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("📥 Full Inbox", "inbox:check"),
						))
						msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}

						_, _ = c.bot.Send(msg)
					}
				}
			}
		}
	}()
}

func (c *CallbackHandler) answer(queryID string, text string, showAlert bool) {
	cb := tgbotapi.NewCallback(queryID, text)
	cb.ShowAlert = showAlert
	_, _ = c.bot.Request(cb)
}
