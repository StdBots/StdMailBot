package mail

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TempMailClient implements Provider for Temp-Mail.io API v3
type TempMailClient struct {
	baseURL string
	client  *http.Client
}

// NewTempMailClient creates TempMail.io client
func NewTempMailClient() *TempMailClient {
	return &TempMailClient{
		baseURL: "https://api.internal.temp-mail.io/api/v3",
		client:  &http.Client{Timeout: 12 * time.Second},
	}
}

func (t *TempMailClient) Name() string {
	return "tempmail"
}

// Generate creates a new disposable email address
func (t *TempMailClient) Generate() (*MailSession, error) {
	reqBody, _ := json.Marshal(map[string]int{
		"min_name_length": 10,
		"max_name_length": 10,
	})

	req, err := http.NewRequest("POST", t.baseURL+"/email/new", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tempmail API returned status: %d", resp.StatusCode)
	}

	var res struct {
		Email string `json:"email"`
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &MailSession{
		Email:     res.Email,
		Token:     res.Token,
		Server:    t.Name(),
		CreatedAt: time.Now(),
	}, nil
}

// GetMessages fetches inbox emails
func (t *TempMailClient) GetMessages(email, token string) ([]IncomingEmail, error) {
	url := fmt.Sprintf("%s/email/%s/messages", t.baseURL, email)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed fetching messages, status: %d", resp.StatusCode)
	}

	var rawMsgs []struct {
		ID        string `json:"id"`
		From      string `json:"from"`
		To        string `json:"to"`
		Subject   string `json:"subject"`
		CreatedAt string `json:"created_at"`
		BodyText  string `json:"body_text"`
		BodyHTML  string `json:"body_html"`
		Text      string `json:"text"`
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(bodyBytes, &rawMsgs); err != nil {
		return nil, nil // Empty or non-array
	}

	var emails []IncomingEmail
	for _, m := range rawMsgs {
		rawBody := m.BodyHTML
		if rawBody == "" {
			rawBody = m.BodyText
		}
		if rawBody == "" {
			rawBody = m.Text
		}

		cleaned := CleanHTML(rawBody)
		otp := ExtractOTP(cleaned)
		magic, other := ExtractLinks(rawBody)

		emails = append(emails, IncomingEmail{
			ID:         m.ID,
			From:       m.From,
			To:         m.To,
			Subject:    m.Subject,
			Date:       m.CreatedAt,
			Body:       cleaned,
			OTP:        otp,
			MagicLinks: magic,
			OtherLinks: other,
			ReceivedAt: time.Now(),
		})
	}

	return emails, nil
}

// Delete permanently removes mailbox
func (t *TempMailClient) Delete(email, token string) error {
	reqBody, _ := json.Marshal(map[string]string{"token": token})
	url := fmt.Sprintf("%s/email/%s", t.baseURL, email)
	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
