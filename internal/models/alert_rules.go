package models

import "time"

// AlertCondition represents a single condition in an alert rule
type AlertCondition struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	RuleID      uint   `gorm:"not null;index" json:"rule_id"`         // Parent rule
	FieldType   string `gorm:"size:50;not null" json:"field_type"`      // status, response_time, uptime, etc.
	Operator    string `gorm:"size:10;not null" json:"operator"`        // eq, ne, gt, lt, ge, le, contains
	Value       string `gorm:"type:text" json:"value"`                  // Threshold value
	LogicalOp   string `gorm:"size:5" json:"logical_op"`                // and, or (for next condition)
	Order       int    `gorm:"default:0" json:"order"`                  // Evaluation order
	CreatedAt   time.Time `json:"created_at"`
}

func (AlertCondition) TableName() string {
	return "alert_conditions"
}

// AlertRuleGroup represents a group of conditions with AND logic
type AlertRuleGroup struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	RuleID      uint   `gorm:"not null;index" json:"rule_id"`         // Parent rule
	Name        string `gorm:"size:255" json:"name"`                   // Group name
	LogicalOp   string `gorm:"size:5" json:"logical_op"`                // and, or (for next group)
	Order       int    `gorm:"default:0" json:"order"`                  // Evaluation order
	CreatedAt   time.Time `json:"created_at"`
}

func (AlertRuleGroup) TableName() string {
	return "alert_rule_groups"
}
