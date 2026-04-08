package alerting

import (
"bytes"
"encoding/json"
"fmt"
"net/http"
"time"

"github.com/Stratify-Systems/ThreatSIM/internal/core"
)

type WebhookNotifier struct {
url    string
client *http.Client
}

func NewWebhookNotifier(url string) *WebhookNotifier {
return &WebhookNotifier{
url: url,
client: &http.Client{
Timeout: 5 * time.Second,
},
}
}

func (w *WebhookNotifier) Send(score core.RiskScore) error {
if w.url == "" {
return nil // Disabled if URL is empty
}

payload, err := json.Marshal(score)
if err != nil {
return fmt.Errorf("failed to marshal risk score: %w", err)
}

req, err := http.NewRequest(http.MethodPost, w.url, bytes.NewBuffer(payload))
if err != nil {
return fmt.Errorf("failed to create webhook request: %w", err)
}

req.Header.Set("Content-Type", "application/json")
req.Header.Set("User-Agent", "ThreatSIM-AlertManager/1.0")

resp, err := w.client.Do(req)
if err != nil {
return fmt.Errorf("failed to send webhook: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode >= 300 {
return fmt.Errorf("webhook responded with status: %d", resp.StatusCode)
}

return nil
}
