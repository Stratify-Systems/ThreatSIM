package alerting

import (
	"fmt"
	"net/smtp"

	"github.com/Stratify-Systems/ThreatSIM/internal/core"
)

type EmailConfig struct {
	SMTPHost string
	SMTPPort string
	Username string
	Password string
	From     string
	To       []string
}

type EmailNotifier struct {
	config EmailConfig
}

func NewEmailNotifier(config EmailConfig) *EmailNotifier {
	return &EmailNotifier{
		config: config,
	}
}

func (e *EmailNotifier) Send(score core.RiskScore) error {
	// If not configured, just return
	if e.config.SMTPHost == "" || len(e.config.To) == 0 {
		return nil
	}

	auth := smtp.PlainAuth("", e.config.Username, e.config.Password, e.config.SMTPHost)

	subject := fmt.Sprintf("Subject: ThreatSIM Alert - %s Threat Detected\n", score.ThreatLevel)
	contentType := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	factors := ""
	for _, f := range score.Factors {
		factors += fmt.Sprintf("<li>%s</li>", f)
	}

	body := fmt.Sprintf(`
<html>
<body>
<h2>🚨 Security Alert: %s Threat Detected</h2>
<ul>
<li><strong>Source IP:</strong> %s</li>
<li><strong>Risk Score:</strong> %d</li>
<li><strong>Detected Rules:</strong>
<ul>
%s
</ul>
</li>
<li><strong>Time (UTC):</strong> %s</li>
</ul>
</body>
</html>
`, score.ThreatLevel, score.SourceIP, score.Score, factors, score.UpdatedAt.UTC().Format("2006-01-02 15:04:05"))

	msg := []byte(subject + contentType + body)

	addr := fmt.Sprintf("%s:%s", e.config.SMTPHost, e.config.SMTPPort)
	if err := smtp.SendMail(addr, auth, e.config.From, e.config.To, msg); err != nil {
		return fmt.Errorf("failed to send email alert: %w", err)
	}

	return nil
}
