package mail

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// SecMailClient implements Provider for 1secmail.com
type SecMailClient struct {
	baseURL string
	client  *http.Client
}

// NewSecMailClient creates 1secmail client
func NewSecMailClient() *SecMailClient {
	return &SecMailClient{
		baseURL: "https://www.1secmail.com/api/v1/",
		client:  &http.Client{Timeout: 12 * time.Second},
	}
}

func (s *SecMailClient) Name() string {
	return "secmail"
}

// Generate creates a random disposable email
func (s *SecMailClient) Generate() (*MailSession, error) {
	url := fmt.Sprintf("%s?action=genRandomMailbox&count=1", s.baseURL)
	resp, err := s.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var emails []string
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil || len(emails) == 0 {
		return nil, fmt.Errorf("failed decoding secmail mailbox: %v", err)
	}

	return &MailSession{
		Email:     emails[0],
		Token:     "",
		Server:    s.Name(),
		CreatedAt: time.Now(),
	}, nil
}

// GetMessages fetches and parses all incoming emails for the address
func (s *SecMailClient) GetMessages(email, token string) ([]IncomingEmail, error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid email format: %s", email)
	}
	login, domain := parts[0], parts[1]

	listURL := fmt.Sprintf("%s?action=getMessages&login=%s&domain=%s", s.baseURL, login, domain)
	resp, err := s.client.Get(listURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var messageSummaries []struct {
		ID      int    `json:"id"`
		From    string `json:"from"`
		Subject string `json:"subject"`
		Date    string `json:"date"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&messageSummaries); err != nil {
		return nil, nil // No messages or empty array
	}

	var emails []IncomingEmail
	// Fetch up to 5 latest messages full content
	limit := 5
	if len(messageSummaries) < limit {
		limit = len(messageSummaries)
	}

	for i := 0; i < limit; i++ {
		summary := messageSummaries[i]
		readURL := fmt.Sprintf("%s?action=readMessage&login=%s&domain=%s&id=%d", s.baseURL, login, domain, summary.ID)

		readResp, err := s.client.Get(readURL)
		if err != nil {
			continue
		}

		var detail struct {
			ID       int    `json:"id"`
			From     string `json:"from"`
			Subject  string `json:"subject"`
			Date     string `json:"date"`
			Body     string `json:"body"`
			TextBody string `json:"textBody"`
			HTMLBody string `json:"htmlBody"`
		}

		if err := json.NewDecoder(readResp.Body).Decode(&detail); err == nil {
			readResp.Body.Close()

			rawBody := detail.HTMLBody
			if rawBody == "" {
				rawBody = detail.TextBody
			}
			if rawBody == "" {
				rawBody = detail.Body
			}

			cleaned := CleanHTML(rawBody)
			otp := ExtractOTP(cleaned)
			magic, other := ExtractLinks(rawBody)

			emails = append(emails, IncomingEmail{
				ID:         fmt.Sprintf("%d", detail.ID),
				From:       detail.From,
				To:         email,
				Subject:    detail.Subject,
				Date:       detail.Date,
				Body:       cleaned,
				OTP:        otp,
				MagicLinks: magic,
				OtherLinks: other,
				ReceivedAt: time.Now(),
			})
		} else {
			readResp.Body.Close()
		}
	}

	return emails, nil
}

// Delete is a no-op for 1secmail since it's fully stateless
func (s *SecMailClient) Delete(email, token string) error {
	return nil
}
