package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"time"

	"monitor/internal/logger"
)

// AlertType 告警类型
type AlertType string

const (
	AlertTypeEmail   AlertType = "email"
	AlertTypeWebhook AlertType = "webhook"
	AlertTypeDingTalk AlertType = "dingtalk"
	AlertTypeWeChat   AlertType = "wechat"
)

// AlertSeverity 告警级别
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityHigh     AlertSeverity = "high"
	SeverityMedium   AlertSeverity = "medium"
	SeverityLow      AlertSeverity = "low"
)

// AlertEvent 告警事件
type AlertEvent struct {
	TargetID     uint32        `json:"target_id"`
	TargetName   string        `json:"target_name"`
	TargetType   string        `json:"target_type"`
	Address      string        `json:"address"`
	Status       string        `json:"status"`       // up, down
	ResponseTime int64         `json:"response_time"`
	Message      string        `json:"message"`
	Timestamp    time.Time     `json:"timestamp"`
	Severity     AlertSeverity `json:"severity"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// AlertChannel 告警渠道配置
type AlertChannel struct {
	ID          uint32      `json:"id"`
	Name        string      `json:"name"`
	Type        AlertType   `json:"type"`
	Enabled     bool        `json:"enabled"`
	Config      interface{} `json:"config"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// EmailConfig 邮件配置
type EmailConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	From         string `json:"from"`
	To           string `json:"to"`
	Subject      string `json:"subject"`
}

// WebhookConfig Webhook配置
type WebhookConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

// DingTalkConfig 钉钉配置
type DingTalkConfig struct {
	WebhookURL string `json:"webhook_url"`
	Secret     string `json:"secret"`
}

// WeChatConfig 企业微信配置
type WeChatConfig struct {
	WebhookURL string `json:"webhook_url"`
}

// AlertRule 告警规则
type AlertRule struct {
	ID              uint32        `json:"id"`
	Name            string        `json:"name"`
	TargetID        *uint32       `json:"target_id,omitempty"` // nil 表示所有目标
	Enabled         bool          `json:"enabled"`
	Severity        AlertSeverity `json:"severity"`
	Conditions      AlertCondition `json:"conditions"`
	Channels        []uint32      `json:"channels"` // 告警渠道ID列表
	CooldownSeconds int           `json:"cooldown_seconds"` // 冷却时间
	LastAlertTime   time.Time     `json:"last_alert_time"`
}

// AlertCondition 告警条件
type AlertCondition struct {
	DownConsecutiveTimes int     `json:"down_consecutive_times"` // 连续失败次数
	SlowResponseThreshold int64  `json:"slow_response_threshold"` // 响应时间阈值（毫秒）
	StatusChanged        bool    `json:"status_changed"`           // 状态改变时告警
}

// Manager 告警管理器
type Manager struct {
	channels    map[uint32]*AlertChannel
	rules       map[uint32]*AlertRule
	eventBuffer map[uint32][]AlertEvent // 每个目标的事件缓冲
}

// NewManager 创建告警管理器
func NewManager() *Manager {
	return &Manager{
		channels:    make(map[uint32]*AlertChannel),
		rules:       make(map[uint32]*AlertRule),
		eventBuffer: make(map[uint32][]AlertEvent),
	}
}

// AddChannel 添加告警渠道
func (m *Manager) AddChannel(channel *AlertChannel) {
	m.channels[channel.ID] = channel
	logger.Log.Info(fmt.Sprintf("Alert channel added: %s (%s)", channel.Name, channel.Type))
}

// RemoveChannel 移除告警渠道
func (m *Manager) RemoveChannel(id uint32) {
	delete(m.channels, id)
	logger.Log.Info(fmt.Sprintf("Alert channel removed: %d", id))
}

// AddRule 添加告警规则
func (m *Manager) AddRule(rule *AlertRule) {
	m.rules[rule.ID] = rule
	logger.Log.Info(fmt.Sprintf("Alert rule added: %s", rule.Name))
}

// RemoveRule 移除告警规则
func (m *Manager) RemoveRule(id uint32) {
	delete(m.rules, id)
	logger.Log.Info(fmt.Sprintf("Alert rule removed: %d", id))
}

// ProcessEvent 处理监控事件并检查是否需要告警
func (m *Manager) ProcessEvent(event AlertEvent) {
	// 添加到事件缓冲
	if _, ok := m.eventBuffer[event.TargetID]; !ok {
		m.eventBuffer[event.TargetID] = []AlertEvent{}
	}
	m.eventBuffer[event.TargetID] = append(m.eventBuffer[event.TargetID], event)

	// 保留最近100个事件
	if len(m.eventBuffer[event.TargetID]) > 100 {
		m.eventBuffer[event.TargetID] = m.eventBuffer[event.TargetID][1:]
	}

	// 检查所有规则
	for _, rule := range m.rules {
		if !rule.Enabled {
			continue
		}

		// 检查规则是否适用于该目标
		if rule.TargetID != nil && *rule.TargetID != event.TargetID {
			continue
		}

		// 检查冷却时间
		if rule.CooldownSeconds > 0 && !rule.LastAlertTime.IsZero() {
			if time.Since(rule.LastAlertTime) < time.Duration(rule.CooldownSeconds)*time.Second {
				continue
			}
		}

		// 检查告警条件
		if m.shouldAlert(event, rule) {
			m.sendAlert(event, rule)
			rule.LastAlertTime = time.Now()
		}
	}
}

// shouldAlert 检查是否应该触发告警
func (m *Manager) shouldAlert(event AlertEvent, rule *AlertRule) bool {
	conditions := rule.Conditions

	// 检查连续失败次数
	if conditions.DownConsecutiveTimes > 0 && event.Status == "down" {
		events := m.eventBuffer[event.TargetID]
		consecutiveDown := 0

		for i := len(events) - 1; i >= 0; i-- {
			if events[i].Status == "down" {
				consecutiveDown++
			} else {
				break
			}
		}

		if consecutiveDown >= conditions.DownConsecutiveTimes {
			return true
		}
	}

	// 检查响应时间阈值
	if conditions.SlowResponseThreshold > 0 && event.ResponseTime > conditions.SlowResponseThreshold {
		return true
	}

	// 检查状态改变
	if conditions.StatusChanged {
		events := m.eventBuffer[event.TargetID]
		if len(events) >= 2 {
			lastEvent := events[len(events)-2]
			if lastEvent.Status != event.Status {
				return true
			}
		}
	}

	return false
}

// sendAlert 发送告警
func (m *Manager) sendAlert(event AlertEvent, rule *AlertRule) {
	for _, channelID := range rule.Channels {
		channel, ok := m.channels[channelID]
		if !ok || !channel.Enabled {
			continue
		}

		switch channel.Type {
		case AlertTypeEmail:
			m.sendEmailAlert(event, channel)
		case AlertTypeWebhook:
			m.sendWebhookAlert(event, channel)
		case AlertTypeDingTalk:
			m.sendDingTalkAlert(event, channel)
		case AlertTypeWeChat:
			m.sendWeChatAlert(event, channel)
		}
	}

	logger.Log.Warn(fmt.Sprintf("ALERT: [%s] %s - %s is %s",
		rule.Severity, event.TargetName, event.Address, event.Status))
}

// sendEmailAlert 发送邮件告警
func (m *Manager) sendEmailAlert(event AlertEvent, channel *AlertChannel) {
	config, ok := channel.Config.(*EmailConfig)
	if !ok {
		logger.Log.Error("Invalid email config")
		return
	}

	subject := fmt.Sprintf("[%s] 监控告警: %s", event.Severity, event.TargetName)
	body := fmt.Sprintf(`
监控告警通知

目标名称: %s
目标类型: %s
目标地址: %s
当前状态: %s
响应时间: %dms
告警级别: %s
告警消息: %s
时间: %s
`,
		event.TargetName,
		event.TargetType,
		event.Address,
		event.Status,
		event.ResponseTime,
		event.Severity,
		event.Message,
		event.Timestamp.Format("2006-01-02 15:04:05"),
	)

	// 发送邮件
	err := smtp.SendMail(
		fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort),
		smtp.PlainAuth("", config.Username, config.Password, config.SMTPHost),
		config.From,
		[]string{config.To},
		[]byte(fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, body)),
	)

	if err != nil {
		logger.Log.Error(fmt.Sprintf("Failed to send email: %v", err))
	} else {
		logger.Log.Info("Email alert sent successfully")
	}
}

// sendWebhookAlert 发送 Webhook 告警
func (m *Manager) sendWebhookAlert(event AlertEvent, channel *AlertChannel) {
	config, ok := channel.Config.(*WebhookConfig)
	if !ok {
		logger.Log.Error("Invalid webhook config")
		return
	}

	payload := map[string]interface{}{
		"target_id":     event.TargetID,
		"target_name":   event.TargetName,
		"target_type":   event.TargetType,
		"address":       event.Address,
		"status":        event.Status,
		"response_time": event.ResponseTime,
		"message":       event.Message,
		"severity":      event.Severity,
		"timestamp":     event.Timestamp.Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("Failed to marshal webhook payload: %v", err))
		return
	}

	req, err := http.NewRequest("POST", config.URL, bytes.NewReader(body))
	if err != nil {
		logger.Log.Error(fmt.Sprintf("Failed to create webhook request: %v", err))
		return
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("Failed to send webhook: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logger.Log.Info("Webhook alert sent successfully")
	} else {
		logger.Log.Error(fmt.Sprintf("Webhook returned status: %d", resp.StatusCode))
	}
}

// sendDingTalkAlert 发送钉钉告警
func (m *Manager) sendDingTalkAlert(event AlertEvent, channel *AlertChannel) {
	config, ok := channel.Config.(*DingTalkConfig)
	if !ok {
		logger.Log.Error("Invalid dingtalk config")
		return
	}

	// 钉钉消息格式
	message := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"title": fmt.Sprintf("[%s] 监控告警", event.Severity),
			"text": fmt.Sprintf(`### %s 监控告警

> 目标名称: %s
> 目标地址: %s
> 当前状态: %s
> 响应时间: %dms
> 告警级别: %s
> 告警消息: %s
> 时间: %s`,
				event.Severity,
				event.TargetName,
				event.Address,
				event.Status,
				event.ResponseTime,
				event.Severity,
				event.Message,
				event.Timestamp.Format("2006-01-02 15:04:05"),
			),
		},
	}

	body, err := json.Marshal(message)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("Failed to marshal dingtalk message: %v", err))
		return
	}

	resp, err := http.Post(config.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		logger.Log.Error(fmt.Sprintf("Failed to send dingtalk alert: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		logger.Log.Info("DingTalk alert sent successfully")
	} else {
		logger.Log.Error(fmt.Sprintf("DingTalk returned status: %d", resp.StatusCode))
	}
}

// sendWeChatAlert 发送企业微信告警
func (m *Manager) sendWeChatAlert(event AlertEvent, channel *AlertChannel) {
	config, ok := channel.Config.(*WeChatConfig)
	if !ok {
		logger.Log.Error("Invalid wechat config")
		return
	}

	// 企业微信消息格式
	message := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"content": fmt.Sprintf(`### %s 监控告警

> 目标名称: %s
> 目标地址: %s
> 当前状态: %s
> 响应时间: %dms
> 告警级别: %s
> 告警消息: %s
> 时间: %s`,
				event.Severity,
				event.TargetName,
				event.Address,
				event.Status,
				event.ResponseTime,
				event.Severity,
				event.Message,
				event.Timestamp.Format("2006-01-02 15:04:05"),
			),
		},
	}

	body, err := json.Marshal(message)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("Failed to marshal wechat message: %v", err))
		return
	}

	resp, err := http.Post(config.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		logger.Log.Error(fmt.Sprintf("Failed to send wechat alert: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		logger.Log.Info("WeChat alert sent successfully")
	} else {
		logger.Log.Error(fmt.Sprintf("WeChat returned status: %d", resp.StatusCode))
	}
}