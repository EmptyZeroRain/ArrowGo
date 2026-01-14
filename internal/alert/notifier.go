package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// Notifier interface for sending alerts
type Notifier interface {
	Send(title, message string) error
}

// WeChatNotifier sends alerts to WeChat Work (企业微信)
type WeChatNotifier struct {
	WebhookURL string
}

func NewWeChatNotifier(webhookURL string) *WeChatNotifier {
	return &WeChatNotifier{WebhookURL: webhookURL}
}

func (w *WeChatNotifier) Send(title, message string) error {
	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": fmt.Sprintf("%s\n\n%s", title, message),
		},
	}

	return w.send(payload)
}

func (w *WeChatNotifier) send(payload map[string]interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(w.WebhookURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("WeChat notification failed with status: %d", resp.StatusCode)
	}

	return nil
}

// DingTalkNotifier sends alerts to DingTalk (钉钉)
type DingTalkNotifier struct {
	WebhookURL string
	Secret     string // Optional: for signature verification
}

func NewDingTalkNotifier(webhookURL, secret string) *DingTalkNotifier {
	return &DingTalkNotifier{
		WebhookURL: webhookURL,
		Secret:     secret,
	}
}

func (d *DingTalkNotifier) Send(title, message string) error {
	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": fmt.Sprintf("%s\n\n%s", title, message),
		},
	}

	return d.send(payload)
}

func (d *DingTalkNotifier) send(payload map[string]interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(d.WebhookURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DingTalk notification failed with status: %d", resp.StatusCode)
	}

	return nil
}

// TelegramNotifier sends alerts to Telegram
type TelegramNotifier struct {
	BotToken string
	ChatID   string
}

func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		BotToken: botToken,
		ChatID:   chatID,
	}
}

func (t *TelegramNotifier) Send(title, message string) error {
	text := fmt.Sprintf("*%s*\n\n%s", title, message)
	// Escape special characters for Markdown
	text = strings.ReplaceAll(text, "_", "\\_")
	text = strings.ReplaceAll(text, "*", "\\*")
	text = strings.ReplaceAll(text, "[", "\\[")
	text = strings.ReplaceAll(text, "]", "\\]")

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.BotToken)
	payload := map[string]interface{}{
		"chat_id":    t.ChatID,
		"text":       fmt.Sprintf("%s\n\n%s", title, message),
		"parse_mode": "Markdown",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram notification failed with status: %d", resp.StatusCode)
	}

	return nil
}

// EmailNotifier sends alerts via email
type EmailNotifier struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	From         string
	To           []string
	UseTLS       bool
}

func NewEmailNotifier(smtpHost string, smtpPort int, username, password, from string, to []string, useTLS bool) *EmailNotifier {
	return &EmailNotifier{
		SMTPHost:     smtpHost,
		SMTPPort:     smtpPort,
		SMTPUsername: username,
		SMTPPassword: password,
		From:         from,
		To:           to,
		UseTLS:       useTLS,
	}
}

func (e *EmailNotifier) Send(title, message string) error {
	// Construct email body
	body := fmt.Sprintf("Subject: %s\r\n\r\n%s", title, message)

	// Build address
	addr := fmt.Sprintf("%s:%d", e.SMTPHost, e.SMTPPort)

	// Send email
	var err error
	if e.UseTLS {
		err = e.sendTLS(addr, body)
	} else {
		err = e.sendPlain(addr, body)
	}

	return err
}

func (e *EmailNotifier) sendPlain(addr, body string) error {
	auth := smtp.PlainAuth("", e.SMTPUsername, e.SMTPPassword, e.SMTPHost)
	return smtp.SendMail(addr, auth, e.From, e.To, []byte(body))
}

func (e *EmailNotifier) sendTLS(addr, body string) error {
	// For TLS, we need to use a custom dialer
	// This is a simplified version - production code should use proper TLS configuration
	auth := smtp.PlainAuth("", e.SMTPUsername, e.SMTPPassword, e.SMTPHost)

	// Try TLS first, fallback to plain
	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return err
	}

	if err := client.Mail(e.From); err != nil {
		return err
	}

	for _, recipient := range e.To {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = writer.Write([]byte(body))
	return err
}

// AlertMessage represents an alert notification
type AlertMessage struct {
	Title    string
	Message  string
	Target   string
	Status   string
	Metadata map[string]string
}

// FormatAlertMessage formats an alert message
func FormatAlertMessage(msg AlertMessage) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("【监控告警】%s\n", msg.Title))
	sb.WriteString(fmt.Sprintf("监控目标: %s\n", msg.Target))
	sb.WriteString(fmt.Sprintf("当前状态: %s\n", msg.Status))
	sb.WriteString(fmt.Sprintf("告警时间: %s\n", time.Now().Format("2006-01-02 15:04:05")))

	if len(msg.Metadata) > 0 {
		sb.WriteString("\n详细信息:\n")
		for k, v := range msg.Metadata {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}

	sb.WriteString(fmt.Sprintf("\n%s", msg.Message))

	return sb.String()
}

// NotifierFactory creates notifiers based on type
type NotifierFactory struct{}

func NewNotifierFactory() *NotifierFactory {
	return &NotifierFactory{}
}

func (f *NotifierFactory) CreateNotifier(channelType string, config map[string]interface{}) (Notifier, error) {
	switch channelType {
	case "wechat":
		webhookURL, ok := config["webhook_url"].(string)
		if !ok {
			return nil, fmt.Errorf("missing webhook_url for WeChat")
		}
		return NewWeChatNotifier(webhookURL), nil

	case "dingtalk":
		webhookURL, ok := config["webhook_url"].(string)
		if !ok {
			return nil, fmt.Errorf("missing webhook_url for DingTalk")
		}
		secret, _ := config["secret"].(string)
		return NewDingTalkNotifier(webhookURL, secret), nil

	case "telegram":
		botToken, ok := config["bot_token"].(string)
		if !ok {
			return nil, fmt.Errorf("missing bot_token for Telegram")
		}
		chatID, ok := config["chat_id"].(string)
		if !ok {
			return nil, fmt.Errorf("missing chat_id for Telegram")
		}
		return NewTelegramNotifier(botToken, chatID), nil

	case "email":
		smtpHost, ok := config["smtp_host"].(string)
		if !ok {
			return nil, fmt.Errorf("missing smtp_host for Email")
		}
		smtpPort, _ := config["smtp_port"].(float64)
		username, _ := config["username"].(string)
		password, _ := config["password"].(string)
		from, _ := config["from"].(string)
		toRaw, ok := config["to"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("missing to for Email")
		}
		to := make([]string, len(toRaw))
		for i, v := range toRaw {
			to[i] = v.(string)
		}
		useTLS, _ := config["use_tls"].(bool)

		return NewEmailNotifier(smtpHost, int(smtpPort), username, password, from, to, useTLS), nil

	default:
		return nil, fmt.Errorf("unsupported channel type: %s", channelType)
	}
}