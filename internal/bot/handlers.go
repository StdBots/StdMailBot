package bot

import (
	"fmt"
	"strings"

	"github.com/StdBots/StdMailBot/internal/credit"
	"github.com/StdBots/StdMailBot/internal/database"
	"github.com/StdBots/StdMailBot/internal/mail"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Handler processes all user commands
type Handler struct {
	bot         *tgbotapi.BotAPI
	mailManager *mail.Manager
	mailRepo    *database.MailRepo
	userRepo    *database.UserRepo
	cache       *database.MemoryCache
	middleware  *Middleware
}

// NewHandler initializes bot command handler
func NewHandler(bot *tgbotapi.BotAPI, mailManager *mail.Manager, mailRepo *database.MailRepo, userRepo *database.UserRepo, cache *database.MemoryCache, middleware *Middleware) *Handler {
	return &Handler{
		bot:         bot,
		mailManager: mailManager,
		mailRepo:    mailRepo,
		userRepo:    userRepo,
		cache:       cache,
		middleware:  middleware,
	}
}

// HandleStart handles /start command
func (h *Handler) HandleStart(msg *tgbotapi.Message) {
	h.middleware.TrackUser(msg.From)

	isMember, _ := h.middleware.CheckForceSub(msg.From.ID)
	if !isMember {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "⚠️ <b>Channel Membership Required</b>\n\nPlease join our updates channel before accessing disposable temp mail:")
		reply.ParseMode = "HTML"
		reply.ReplyMarkup = h.middleware.GetForceSubMarkup()
		_, _ = h.bot.Send(reply)
		return
	}

	session, err := h.mailRepo.GetActiveMail(msg.From.ID)
	if err == nil && session != nil && session.Email != "" {
		// User already has an active mailbox
		h.sendActiveMailboxView(msg.Chat.ID, session, "👋 Welcome back! Here is your active disposable mailbox:")
		return
	}

	welcomeText := fmt.Sprintf(
		"📧 <b>Welcome to StdMailBot!</b>\n\n"+
			"The fastest, most reliable disposable temporary email generator on Telegram.\n\n"+
			"✨ <b>Features:</b>\n"+
			"• 1-Click instant temporary email address\n"+
			"• Automatic <b>OTP code detection</b> for 1-tap copying\n"+
			"• Automatic <b>Magic login / verification link extraction</b>\n"+
			"• <b>🔔 Inbox Watch Mode:</b> Background instant notification when emails arrive\n"+
			"• <b>Dual Mail Engines:</b> 100%% uptime with automatic failover\n"+
			"• Persistent sessions (your email stays active even if bot restarts)\n\n"+
			"⚡ <i>Engineered by STD DEEPANSHU (%s)</i>\n%s",
		credit.Domain,
		credit.GetFooter(),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.GetWatermarked(welcomeText))
	reply.ParseMode = "HTML"

	btnGenerate := tgbotapi.NewInlineKeyboardButtonData("🎲 Generate Email", "mail:generate")
	btnChannel := tgbotapi.NewInlineKeyboardButtonURL("📢 Updates", "https://t.me/StdBots")
	btnDev := tgbotapi.NewInlineKeyboardButtonURL("👨‍💻 Developer", "https://"+credit.Domain)

	reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(btnGenerate),
		tgbotapi.NewInlineKeyboardRow(btnChannel, btnDev),
	)

	_, _ = h.bot.Send(reply)
}

// HandleGetMail handles /getmail command
func (h *Handler) HandleGetMail(msg *tgbotapi.Message) {
	h.middleware.TrackUser(msg.From)

	isMember, _ := h.middleware.CheckForceSub(msg.From.ID)
	if !isMember {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "⚠️ Please join our channel to generate emails:")
		reply.ReplyMarkup = h.middleware.GetForceSubMarkup()
		_, _ = h.bot.Send(reply)
		return
	}

	loadingMsg, _ := h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Generating fresh disposable email..."))

	session, err := h.mailManager.GenerateEmail()
	if err != nil {
		edit := tgbotapi.NewEditMessageText(msg.Chat.ID, loadingMsg.MessageID, "❌ Failed to generate email. Please try again in a few seconds.")
		_, _ = h.bot.Send(edit)
		return
	}

	_ = h.mailRepo.SaveActiveMail(msg.From.ID, session)

	del := tgbotapi.NewDeleteMessage(msg.Chat.ID, loadingMsg.MessageID)
	_, _ = h.bot.Request(del)

	h.sendActiveMailboxView(msg.Chat.ID, session, "🎉 <b>Your New Disposable Email is Ready:</b>")
}

// HandleInbox handles /inbox command
func (h *Handler) HandleInbox(msg *tgbotapi.Message) {
	h.middleware.TrackUser(msg.From)

	session, err := h.mailRepo.GetActiveMail(msg.From.ID)
	if err != nil || session == nil || session.Email == "" {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ You don't have an active email yet.\nSend /getmail or click below to generate one:")
		btnGen := tgbotapi.NewInlineKeyboardButtonData("🎲 Generate Email", "mail:generate")
		reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btnGen))
		_, _ = h.bot.Send(reply)
		return
	}

	loadingMsg, _ := h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Checking mailbox for incoming emails..."))

	messages, err := h.mailManager.GetMessages(session)
	if err != nil {
		edit := tgbotapi.NewEditMessageText(msg.Chat.ID, loadingMsg.MessageID, "❌ Error contacting mail server. Please try refreshing again in a few seconds.")
		_, _ = h.bot.Send(edit)
		return
	}

	del := tgbotapi.NewDeleteMessage(msg.Chat.ID, loadingMsg.MessageID)
	_, _ = h.bot.Request(del)

	h.renderInboxView(msg.Chat.ID, 0, session, messages)
}

// HandleChangeMail handles /changemail command
func (h *Handler) HandleChangeMail(msg *tgbotapi.Message) {
	h.middleware.TrackUser(msg.From)

	loadingMsg, _ := h.bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Discarding old mailbox and generating new email..."))

	// Delete old
	oldSession, _ := h.mailRepo.DeleteActiveMail(msg.From.ID)
	if oldSession != nil {
		_ = h.mailManager.DeleteEmail(oldSession)
	}

	// Generate new
	newSession, err := h.mailManager.GenerateEmail()
	if err != nil {
		edit := tgbotapi.NewEditMessageText(msg.Chat.ID, loadingMsg.MessageID, "❌ Error generating new email. Please try again.")
		_, _ = h.bot.Send(edit)
		return
	}

	_ = h.mailRepo.SaveActiveMail(msg.From.ID, newSession)

	del := tgbotapi.NewDeleteMessage(msg.Chat.ID, loadingMsg.MessageID)
	_, _ = h.bot.Request(del)

	h.sendActiveMailboxView(msg.Chat.ID, newSession, "🔄 <b>Your Email Has Been Changed:</b>")
}

// HandleDeleteMail handles /deletemail command
func (h *Handler) HandleDeleteMail(msg *tgbotapi.Message) {
	h.middleware.TrackUser(msg.From)

	oldSession, err := h.mailRepo.DeleteActiveMail(msg.From.ID)
	if err != nil || oldSession == nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "ℹ️ You don't have an active email to delete.")
		_, _ = h.bot.Send(reply)
		return
	}

	_ = h.mailManager.DeleteEmail(oldSession)

	reply := tgbotapi.NewMessage(msg.Chat.ID, "🗑️ <b>Mailbox Deleted Successfully.</b>\n\nYour temporary email address was destroyed.")
	reply.ParseMode = "HTML"
	btnGen := tgbotapi.NewInlineKeyboardButtonData("🎲 Generate New Email", "mail:generate")
	reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btnGen))
	_, _ = h.bot.Send(reply)
}

// HandleHelp handles /help command
func (h *Handler) HandleHelp(msg *tgbotapi.Message) {
	h.middleware.TrackUser(msg.From)

	helpText := fmt.Sprintf(
		"📖 <b>StdMailBot Help & Tips:</b>\n\n"+
			"<b>Commands:</b>\n"+
			"• /start — Launch bot and view your mailbox\n"+
			"• /getmail — Generate a fresh disposable email\n"+
			"• /inbox — Check inbox for incoming messages & OTPs\n"+
			"• /watch — Toggle background auto-notifier for 5 minutes\n"+
			"• /changemail — Discard current email and get a new one\n"+
			"• /deletemail — Delete and destroy your active email\n"+
			"• /help — Show this help message\n\n"+
			"💡 <b>Tips:</b>\n"+
			"1. Tap directly on the email in monospace to copy it instantly!\n"+
			"2. If an OTP is received, an instant 1-tap copy button will appear!\n"+
			"3. Turn on <b>'🔔 Watch Mode'</b> to automatically receive incoming emails without refreshing manually.\n\n"+
			"%s",
		credit.GetFooter(),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, credit.GetWatermarked(helpText))
	reply.ParseMode = "HTML"
	_, _ = h.bot.Send(reply)
}

func (h *Handler) sendActiveMailboxView(chatID int64, session *mail.MailSession, header string) {
	text := fmt.Sprintf(
		"%s\n\n"+
			"📧 <b>Your Disposable Email:</b>\n"+
			"<code>%s</code>\n\n"+
			"<i>(Tap the email above to copy to clipboard)</i>\n\n"+
			"🟢 <b>Status:</b> Active\n"+
			"🌐 <b>Engine:</b> %s\n"+
			"━━━━━━━━━━━━━━━━━━━━\n"+
			"%s",
		header,
		session.Email,
		strings.ToUpper(session.Server),
		credit.GetFooter(),
	)

	reply := tgbotapi.NewMessage(chatID, credit.GetWatermarked(text))
	reply.ParseMode = "HTML"

	btnInbox := tgbotapi.NewInlineKeyboardButtonData("📥 Check Inbox", "inbox:check")
	btnWatch := tgbotapi.NewInlineKeyboardButtonData("🔔 Watch (Auto-Notify)", "mail:watch")
	btnChange := tgbotapi.NewInlineKeyboardButtonData("🔄 Change Mail", "mail:change")
	btnDelete := tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete", "mail:delete")

	reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(btnInbox, btnWatch),
		tgbotapi.NewInlineKeyboardRow(btnChange, btnDelete),
	)

	_, _ = h.bot.Send(reply)
}

func (h *Handler) renderInboxView(chatID int64, editMsgID int, session *mail.MailSession, messages []mail.IncomingEmail) {
	if len(messages) == 0 {
		emptyText := fmt.Sprintf(
			"📭 <b>Inbox is Empty</b>\n\n"+
				"No messages received yet for:\n"+
				"<code>%s</code>\n\n"+
				"💡 Send your verification email to this address, then click <b>'🔄 Refresh Inbox'</b> below or turn on <b>'🔔 Watch Mode'</b> to get notified automatically!\n\n"+
				"%s",
			session.Email,
			credit.GetFooter(),
		)

		var markup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh Inbox", "inbox:check"),
				tgbotapi.NewInlineKeyboardButtonData("🔔 Watch Mode", "mail:watch"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🎲 Change Mail", "mail:change"),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete Mailbox", "mail:delete"),
				),
			),
		)

		if editMsgID != 0 {
			edit := tgbotapi.NewEditMessageText(chatID, editMsgID, credit.GetWatermarked(emptyText))
			edit.ParseMode = "HTML"
			edit.ReplyMarkup = &markup
			_, _ = h.bot.Send(edit)
		} else {
			msg := tgbotapi.NewMessage(chatID, credit.GetWatermarked(emptyText))
			msg.ParseMode = "HTML"
			msg.ReplyMarkup = markup
			_, _ = h.bot.Send(msg)
		}
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📬 <b>Inbox Messages (%d)</b> for <code>%s</code>:\n\n", len(messages), session.Email))

	var detectedOTP string

	for i, m := range messages {
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString(fmt.Sprintf("📨 <b>Mail #%d</b>\n", i+1))
		sb.WriteString(fmt.Sprintf("📌 <b>From:</b> <code>%s</code>\n", escapeHTML(m.From)))
		sb.WriteString(fmt.Sprintf("🔰 <b>Subject:</b> <b>%s</b>\n", escapeHTML(m.Subject)))

		if m.OTP != "" {
			detectedOTP = m.OTP
			sb.WriteString(fmt.Sprintf("🔑 <b>OTP Code:</b> <code>%s</code>\n", m.OTP))
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
	}

	sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(credit.GetFooter())

	var rows [][]tgbotapi.InlineKeyboardButton

	// If OTP was detected, provide a 1-tap copy button!
	if detectedOTP != "" {
		btnCopyOTP := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("🔑 Copy OTP: %s", detectedOTP),
			fmt.Sprintf("copy:otp:%s", detectedOTP),
		)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btnCopyOTP))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh Inbox", "inbox:check"),
		tgbotapi.NewInlineKeyboardButtonData("🔔 Watch Mode", "mail:watch"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🎲 Change Mail", "mail:change"),
		tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete Mailbox", "mail:delete"),
	))

	markup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}

	if editMsgID != 0 {
		edit := tgbotapi.NewEditMessageText(chatID, editMsgID, credit.GetWatermarked(sb.String()))
		edit.ParseMode = "HTML"
		edit.ReplyMarkup = &markup
		_, _ = h.bot.Send(edit)
	} else {
		msg := tgbotapi.NewMessage(chatID, credit.GetWatermarked(sb.String()))
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = markup
		_, _ = h.bot.Send(msg)
	}
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
