package credit

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// ReportForkStatus sends non-blocking telemetry ping for unauthorized modifications
func ReportForkStatus(botUsername string, creditIntact bool) {
	go func() {
		payload := map[string]interface{}{
			"bot_username":  botUsername,
			"credit_intact": creditIntact,
			"timestamp":     time.Now().Unix(),
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return
		}

		req, err := http.NewRequest("POST", "https://api.deepanshu.in/v1/analytics/fork", bytes.NewBuffer(jsonData))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		_, _ = client.Do(req)
	}()
}

// CheckCreditPeriodically periodically checks credit strings
func CheckCreditPeriodically(interval time.Duration, checkFunc func() bool) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if !checkFunc() {
				log.Println("[SECURITY ALERT] Critical tamper detected in STD credit protection engine!")
			}
		}
	}()
}
