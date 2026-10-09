package mail

import (
	"errors"
	"log"
	"time"
)

// IncomingEmail represents an incoming email message
type IncomingEmail struct {
	ID         string    `json:"id" bson:"id"`
	From       string    `json:"from" bson:"from"`
	To         string    `json:"to" bson:"to"`
	Subject    string    `json:"subject" bson:"subject"`
	Date       string    `json:"date" bson:"date"`
	Body       string    `json:"body" bson:"body"`
	OTP        string    `json:"otp" bson:"otp"`
	MagicLinks []string  `json:"magic_links" bson:"magic_links"`
	OtherLinks []string  `json:"other_links" bson:"other_links"`
	ReceivedAt time.Time `json:"received_at" bson:"received_at"`
}

// MailSession represents an active user mailbox
type MailSession struct {
	Email     string    `json:"email" bson:"email"`
	Token     string    `json:"token" bson:"token"`
	Server    string    `json:"server" bson:"server"` // "tempmail" or "secmail"
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

// Provider defines the contract for a temp mail service
type Provider interface {
	Name() string
	Generate() (*MailSession, error)
	GetMessages(email, token string) ([]IncomingEmail, error)
	Delete(email, token string) error
}

// Manager orchestrates dual mail providers with failover
type Manager struct {
	primary   Provider
	secondary Provider
}

// NewManager creates a resilient dual-engine mail manager
func NewManager(defaultEngine string) *Manager {
	tm := NewTempMailClient()
	sm := NewSecMailClient()

	if defaultEngine == "secmail" {
		return &Manager{primary: sm, secondary: tm}
	}
	return &Manager{primary: tm, secondary: sm}
}

// GenerateEmail generates an email with automatic fallback
func (m *Manager) GenerateEmail() (*MailSession, error) {
	// Try primary
	session, err := m.primary.Generate()
	if err == nil && session != nil && session.Email != "" {
		return session, nil
	}
	log.Printf("[MAIL FAILOVER] Primary engine %s failed (%v), switching to secondary %s...", m.primary.Name(), err, m.secondary.Name())

	// Try secondary
	session, err = m.secondary.Generate()
	if err == nil && session != nil && session.Email != "" {
		return session, nil
	}

	return nil, errors.New("all mail engines are currently unavailable, please try again in a few moments")
}

// GetMessages fetches inbox messages using the session's specific engine or fallback
func (m *Manager) GetMessages(session *MailSession) ([]IncomingEmail, error) {
	if session == nil || session.Email == "" {
		return nil, errors.New("no active email session")
	}

	if session.Server == m.secondary.Name() {
		return m.secondary.GetMessages(session.Email, session.Token)
	}

	msgs, err := m.primary.GetMessages(session.Email, session.Token)
	if err != nil {
		log.Printf("[MAIL WARNING] Failed fetching from %s (%v), attempting secondary...", session.Server, err)
		return m.secondary.GetMessages(session.Email, session.Token)
	}
	return msgs, nil
}

// DeleteEmail deletes mailbox
func (m *Manager) DeleteEmail(session *MailSession) error {
	if session == nil {
		return nil
	}
	if session.Server == m.secondary.Name() {
		return m.secondary.Delete(session.Email, session.Token)
	}
	return m.primary.Delete(session.Email, session.Token)
}
