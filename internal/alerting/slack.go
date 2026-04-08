package alerting

import (
"bytes"
"encoding/json"
"fmt"
"net/http"
"time"

"github.com/Stratify-Systems/ThreatSIM/internal/core"
)

type SlackNotifier struct {
webhookURL string
client     *http.Client
}

func NewSlackNotifier(webhookURL string) *SlackNotifier {
return &SlackNotifier{
webhookURL: webhookURL,
client: &http.Client{
Timeout: 5 * time.Second,
},
}
}

func (s *SlackNotifier) Send(score core.RiskScore) error {
if s.webhookURL == "" {
return nil
}

color := "#ff0000" // Red for CRITICAL
if score.ThreatLevel == core.ThreatHigh {
color = "#ff8c00" // Orange for HIGH
}

factors := ""
for i, f := range score.Factors {
if i > 0 {
factors += ", "
}
factors += f
}

// Build Slack message block format
msg := map[string]interface{}{
"attachments": []map[string]interface{}{
{
"color": color,
"blocks": []map[string]interface{}{
{
"type": "header",
"text": map[string]interface{}{
"type": "plain_text",
"text": fmt.Sprintf("🚨 Security Alert: %s Threat Detected", score.ThreatLevel),
"emoji": true,
},
},
{
"type": "section",
"fields": []map[string]interface{}{
{
"type": "mrkdwn",
"text": fmt.Sprintf("*Source IP:*\n`%s`", score.SourceIP),
},
{
"type": "mrkdwn",
"text": fmt.Sprintf("*Risk Score:*\n*%d*", score.Score),
},
{
"type": "mrkdwn",
"text": fmt.Sprintf("*Detected Rules:*\n%s", factors),
},
{
"type": "mrkdwn",
"text": fmt.Sprintf("*Time:*\n%s", score.UpdatedAt.UTC().Format(time.RFC3339)),
},
},
},
},
},
},
}

payload, err := json.Marshal(msg)
if err != nil {
return fmt.Errorf("failed to marshal slack message: %w", err)
}

req, err := http.NewRequest(http.MethodPost, s.webhookURL, bytes.NewBuffer(payload))
if err != nil {
return fmt.Errorf("failed to create slack request: %w", err)
}
req.Header.Set("Content-Type", "application/json")

resp, err := s.client.Do(req)
if err != nil {
return fmt.Errorf("failed to send slack webhook: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode >= 300 {
return fmt.Errorf("slack responded with status: %d", resp.StatusCode)
}

return nil
}
