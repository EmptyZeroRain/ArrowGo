package models

import "time"

// AlertChannel 告警渠道模型
type AlertChannel struct {
	ID        uint32    `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	Type      string    `gorm:"size:50;not null" json:"type"` // email, webhook, dingtalk, wechat
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	Config    string    `gorm:"type:text;not null" json:"config"` // JSON string
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (AlertChannel) TableName() string {
	return "alert_channels"
}

// AlertRule 告警规则模型
type AlertRule struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	TargetID       uint32 `gorm:"not null" json:"target_id"`           // Associated monitor target
	ChannelID      uint   `gorm:"not null" json:"channel_id"`           // Alert channel
	ThresholdType  string `gorm:"size:20" json:"threshold_type"`        // failure_count, response_time
	ThresholdValue int    `json:"threshold_value"`                      // Threshold value
	Enabled        bool   `gorm:"default:true" json:"enabled"`          // Is enabled
	// Advanced fields
	ConditionLogic string `gorm:"type:text" json:"condition_logic"` // JSON: complex conditions with operators
	CooldownSeconds int   `gorm:"default:300" json:"cooldown_seconds"` // Cooldown between alerts
	LastAlertTime   time.Time `json:"last_alert_time"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relationships for loading
	Conditions []AlertCondition `gorm:"foreignKey:RuleID" json:"conditions,omitempty"`
	Groups     []AlertRuleGroup  `gorm:"foreignKey:RuleID" json:"groups,omitempty"`
}

func (AlertRule) TableName() string {
	return "alert_rules"
}

// AlertHistory 告警历史记录
type AlertHistory struct {
	ID          uint32    `gorm:"primaryKey" json:"id"`
	RuleID      uint32    `json:"rule_id"`
	TargetID    uint32    `json:"target_id"`
	ChannelID   uint32    `json:"channel_id"`
	Severity    string    `gorm:"size:50" json:"severity"`
	Status      string    `gorm:"size:50" json:"status"`
	Message     string    `gorm:"type:text" json:"message"`
	SentAt      time.Time `json:"sent_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (AlertHistory) TableName() string {
	return "alert_history"
}