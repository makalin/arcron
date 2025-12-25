package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/makalin/arcron/internal/config"
	"github.com/makalin/arcron/internal/types"
	"github.com/sirupsen/logrus"
)

// Manager manages alerting
type Manager struct {
	config *config.Config
	client *http.Client
}

// New creates a new alert manager
func New(cfg *config.Config) (*Manager, error) {
	return &Manager{
		config: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// Alert represents an alert
type Alert struct {
	Level       string    `json:"level"`
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	JobName     string    `json:"job_name,omitempty"`
	ExecutionID string    `json:"execution_id,omitempty"`
	Metrics     interface{} `json:"metrics,omitempty"`
}

// SendJobAlert sends an alert for a job execution
func (m *Manager) SendJobAlert(execution *types.JobExecution) error {
	if !m.config.Alerts.Enabled {
		return nil
	}

	var level string
	var title string

	switch execution.Status {
	case types.StatusFailed:
		level = "error"
		title = fmt.Sprintf("Job Failed: %s", execution.JobName)
	case types.StatusCompleted:
		level = "info"
		title = fmt.Sprintf("Job Completed: %s", execution.JobName)
	default:
		return nil // Don't alert for other statuses
	}

	alert := Alert{
		Level:       level,
		Title:       title,
		Message:     fmt.Sprintf("Job %s %s. Duration: %.2fs", execution.JobName, execution.Status, execution.Duration),
		Timestamp:   time.Now(),
		JobName:     execution.JobName,
		ExecutionID: execution.ID,
	}

	return m.sendAlert(alert)
}

// SendSystemAlert sends a system-level alert
func (m *Manager) SendSystemAlert(level, title, message string, metrics interface{}) error {
	if !m.config.Alerts.Enabled {
		return nil
	}

	alert := Alert{
		Level:     level,
		Title:     title,
		Message:   message,
		Timestamp: time.Now(),
		Metrics:   metrics,
	}

	return m.sendAlert(alert)
}

// sendAlert sends an alert through all configured channels
func (m *Manager) sendAlert(alert Alert) error {
	var errors []string

	// Send email alert
	if m.config.Alerts.Email.Enabled {
		if err := m.sendEmailAlert(alert); err != nil {
			errors = append(errors, fmt.Sprintf("email: %v", err))
		}
	}

	// Send Slack alert
	if m.config.Alerts.Slack.Enabled {
		if err := m.sendSlackAlert(alert); err != nil {
			errors = append(errors, fmt.Sprintf("slack: %v", err))
		}
	}

	// Send webhook alert
	if m.config.Alerts.Webhook.Enabled {
		if err := m.sendWebhookAlert(alert); err != nil {
			errors = append(errors, fmt.Sprintf("webhook: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("alert sending errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// sendEmailAlert sends an email alert
func (m *Manager) sendEmailAlert(alert Alert) error {
	emailCfg := m.config.Alerts.Email

	if emailCfg.SMTPHost == "" || emailCfg.Username == "" {
		return fmt.Errorf("email configuration incomplete")
	}

	auth := smtp.PlainAuth("", emailCfg.Username, emailCfg.Password, emailCfg.SMTPHost)

	subject := fmt.Sprintf("[%s] %s", strings.ToUpper(alert.Level), alert.Title)
	body := fmt.Sprintf(`
Alert: %s
Level: %s
Time: %s
Message: %s
`, alert.Title, alert.Level, alert.Timestamp.Format(time.RFC3339), alert.Message)

	msg := []byte(fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, body))

	addr := fmt.Sprintf("%s:%d", emailCfg.SMTPHost, emailCfg.SMTPPort)
	
	for _, to := range emailCfg.To {
		if err := smtp.SendMail(addr, auth, emailCfg.From, []string{to}, msg); err != nil {
			logrus.Errorf("Failed to send email to %s: %v", to, err)
			return err
		}
	}

	logrus.Infof("Email alert sent: %s", alert.Title)
	return nil
}

// sendSlackAlert sends a Slack alert
func (m *Manager) sendSlackAlert(alert Alert) error {
	slackCfg := m.config.Alerts.Slack

	if slackCfg.WebhookURL == "" {
		return fmt.Errorf("slack webhook URL not configured")
	}

	color := "#36a64f" // Green
	if alert.Level == "error" || alert.Level == "critical" {
		color = "#ff0000" // Red
	} else if alert.Level == "warning" {
		color = "#ffaa00" // Orange
	}

	payload := map[string]interface{}{
		"channel":  slackCfg.Channel,
		"username": slackCfg.Username,
		"attachments": []map[string]interface{}{
			{
				"color":     color,
				"title":     alert.Title,
				"text":      alert.Message,
				"timestamp": alert.Timestamp.Unix(),
				"fields": []map[string]interface{}{
					{
						"title": "Level",
						"value": alert.Level,
						"short": true,
					},
					{
						"title": "Time",
						"value": alert.Timestamp.Format(time.RFC3339),
						"short": true,
					},
				},
			},
		},
	}

	if alert.JobName != "" {
		payload["attachments"].([]map[string]interface{})[0]["fields"] = append(
			payload["attachments"].([]map[string]interface{})[0]["fields"].([]map[string]interface{}),
			map[string]interface{}{
				"title": "Job",
				"value": alert.JobName,
				"short": true,
			},
		)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %v", err)
	}

	resp, err := m.client.Post(slackCfg.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send Slack alert: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	logrus.Infof("Slack alert sent: %s", alert.Title)
	return nil
}

// sendWebhookAlert sends a webhook alert
func (m *Manager) sendWebhookAlert(alert Alert) error {
	webhookCfg := m.config.Alerts.Webhook

	if webhookCfg.URL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	jsonData, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %v", err)
	}

	req, err := http.NewRequest(webhookCfg.Method, webhookCfg.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %v", err)
	}

	for k, v := range webhookCfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook alert: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	logrus.Infof("Webhook alert sent: %s", alert.Title)
	return nil
}

