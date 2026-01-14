package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"monitor/internal/database"
	"monitor/internal/models"
)

// Service manages alert notifications
type Service struct {
	factory *NotifierFactory
	mu      sync.RWMutex
}

// NewService creates a new alert service
func NewService() *Service {
	return &Service{
		factory: NewNotifierFactory(),
	}
}

// SendAlert sends an alert notification
func (s *Service) SendAlert(ctx context.Context, targetID uint32, status string, metadata map[string]string) error {
	db := database.GetDB()

	// Get alert rules for this target
	var rules []models.AlertRule
	if err := db.Where("target_id = ? AND enabled = ?", targetID, true).Find(&rules).Error; err != nil {
		return err
	}

	// Get target info
	var target models.MonitorTarget
	if err := db.First(&target, targetID).Error; err != nil {
		return err
	}

	// Send alerts for each matching rule
	for _, rule := range rules {
		if s.shouldTriggerAlert(rule, status, metadata) {
			// Get channel
			var channel models.AlertChannel
			if err := db.First(&channel, rule.ChannelID).Error; err != nil {
				log.Printf("Failed to get alert channel %d: %v", rule.ChannelID, err)
				continue
			}

			if !channel.Enabled {
				continue
			}

			// Parse channel config
			var config map[string]interface{}
			if err := json.Unmarshal([]byte(channel.Config), &config); err != nil {
				log.Printf("Failed to parse channel config: %v", err)
				continue
			}

			// Create notifier
			notifier, err := s.factory.CreateNotifier(channel.Type, config)
			if err != nil {
				log.Printf("Failed to create notifier: %v", err)
				continue
			}

			// Format and send alert
			msg := AlertMessage{
				Title:    fmt.Sprintf("监控告警: %s", target.Name),
				Message:  s.formatAlertMessage(status, metadata),
				Target:   target.Name,
				Status:   status,
				Metadata: metadata,
			}

			formattedMsg := FormatAlertMessage(msg)

			// Send notification asynchronously
			go func(n Notifier, title, message string) {
				if err := n.Send(title, message); err != nil {
					log.Printf("Failed to send alert: %v", err)
				}
			}(notifier, msg.Title, formattedMsg)
		}
	}

	return nil
}

// shouldTriggerAlert determines if an alert should be sent based on rules
func (s *Service) shouldTriggerAlert(rule models.AlertRule, status string, metadata map[string]string) bool {
	// Simple implementation: trigger on any "down" status
	if status == "down" {
		return true
	}

	// Check threshold-based rules
	if rule.ThresholdType == "response_time" {
		if _, ok := metadata["response_time"]; ok {
			// Compare response time with threshold
			// This is a simplified check
			return true
		}
	}

	return false
}

// formatAlertMessage formats alert message details
func (s *Service) formatAlertMessage(status string, metadata map[string]string) string {
	var msg string
	if status == "down" {
		msg = "监控目标已宕机，请及时处理！"
	} else if status == "degraded" {
		msg = "监控目标性能下降，请关注！"
	} else {
		msg = "监控目标状态异常"
	}

	if len(metadata) > 0 {
		msg += "\n\n详细信息:"
		for k, v := range metadata {
			msg += fmt.Sprintf("\n%s: %s", k, v)
		}
	}

	return msg
}

// CreateAlertChannel creates a new alert channel
func (s *Service) CreateAlertChannel(channel *models.AlertChannel) error {
	db := database.GetDB()
	return db.Create(channel).Error
}

// GetAlertChannel retrieves an alert channel
func (s *Service) GetAlertChannel(id uint) (*models.AlertChannel, error) {
	db := database.GetDB()
	var channel models.AlertChannel
	err := db.First(&channel, id).Error
	return &channel, err
}

// ListAlertChannels lists all alert channels
func (s *Service) ListAlertChannels() ([]models.AlertChannel, error) {
	db := database.GetDB()
	var channels []models.AlertChannel
	err := db.Find(&channels).Error
	return channels, err
}

// UpdateAlertChannel updates an alert channel
func (s *Service) UpdateAlertChannel(channel *models.AlertChannel) error {
	db := database.GetDB()
	return db.Save(channel).Error
}

// DeleteAlertChannel deletes an alert channel
func (s *Service) DeleteAlertChannel(id uint) error {
	db := database.GetDB()
	return db.Delete(&models.AlertChannel{}, id).Error
}

// CreateAlertRule creates a new alert rule
func (s *Service) CreateAlertRule(rule *models.AlertRule) error {
	db := database.GetDB()
	return db.Create(rule).Error
}

// GetAlertRule retrieves an alert rule
func (s *Service) GetAlertRule(id uint) (*models.AlertRule, error) {
	db := database.GetDB()
	var rule models.AlertRule
	err := db.First(&rule, id).Error
	return &rule, err
}

// ListAlertRules lists all alert rules
func (s *Service) ListAlertRules() ([]models.AlertRule, error) {
	db := database.GetDB()
	var rules []models.AlertRule
	err := db.Preload("Channel").Find(&rules).Error
	return rules, err
}

// ListAlertRulesByTarget lists alert rules for a specific target
func (s *Service) ListAlertRulesByTarget(targetID uint32) ([]models.AlertRule, error) {
	db := database.GetDB()
	var rules []models.AlertRule
	err := db.Where("target_id = ?", targetID).Preload("Channel").Find(&rules).Error
	return rules, err
}

// UpdateAlertRule updates an alert rule
func (s *Service) UpdateAlertRule(rule *models.AlertRule) error {
	db := database.GetDB()
	return db.Save(rule).Error
}

// DeleteAlertRule deletes an alert rule
func (s *Service) DeleteAlertRule(id uint) error {
	db := database.GetDB()
	return db.Delete(&models.AlertRule{}, id).Error
}

// TestAlertChannel tests an alert channel by sending a test message
func (s *Service) TestAlertChannel(id uint) error {
	db := database.GetDB()

	var channel models.AlertChannel
	if err := db.First(&channel, id).Error; err != nil {
		return err
	}

	// Parse channel config
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(channel.Config), &config); err != nil {
		return fmt.Errorf("failed to parse channel config: %w", err)
	}

	// Create notifier
	notifier, err := s.factory.CreateNotifier(channel.Type, config)
	if err != nil {
		return err
	}

	// Send test message
	msg := AlertMessage{
		Title:    "测试告警",
		Message:  "这是一条测试告警消息，如果您收到此消息，说明告警通道配置成功！",
		Target:   "测试目标",
		Status:   "up",
		Metadata: map[string]string{"test": "true"},
	}

	formattedMsg := FormatAlertMessage(msg)
	return notifier.Send(msg.Title, formattedMsg)
}