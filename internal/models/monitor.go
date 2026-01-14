package models

import (
	"time"
)

type MonitorTarget struct {
	ID        uint32 `gorm:"primaryKey" json:"id"`
	Name      string `gorm:"size:255;not null" json:"name"`
	Type      string `gorm:"size:50;not null" json:"type"` // http, https, tcp, udp, dns
	Address   string `gorm:"size:500;not null" json:"address"`
	Port      int32  `json:"port"`
	Interval  int64  `gorm:"default:60" json:"interval"` // seconds
	Metadata  string `gorm:"type:text" json:"metadata"`  // JSON string
	Enabled   bool   `gorm:"default:true" json:"enabled"`

	// HTTP/HTTPS specific fields
	HTTPMethod         string `gorm:"size:10" json:"http_method"`          // GET, POST, PUT, DELETE, etc.
	HTTPHeaders        string `gorm:"type:text" json:"http_headers"`       // JSON string
	HTTPBody           string `gorm:"type:text" json:"http_body"`
	ResolvedHost       string `gorm:"size:255" json:"resolved_host"`       // Custom host resolution
	FollowRedirects    bool   `gorm:"default:true" json:"follow_redirects"` // Follow 301/302 redirects
	MaxRedirects       int    `gorm:"default:10" json:"max_redirects"`      // Maximum redirect depth
	ExpectedStatusCodes string `gorm:"type:text" json:"expected_status_codes"` // Comma-separated status codes (e.g., "200,201,301,302")

	// DNS specific fields
	DNSServer      string `gorm:"size:255" json:"dns_server"`       // DNS server address (e.g., 8.8.8.8:53)
	DNSServerName  string `gorm:"size:255" json:"dns_server_name"`   // DNS server name (e.g., "Google DNS")
	DNSServerType  string `gorm:"size:10" json:"dns_server_type"`   // DNS protocol: udp, tcp, doh, dot

	// PING specific fields
	PingCount  int    `gorm:"default:4" json:"ping_count"`   // Number of ping packets to send
	PingSize   int    `gorm:"default:32" json:"ping_size"`   // Size of ping packet in bytes
	PingTimeout int   `gorm:"default:5000" json:"ping_timeout"` // Timeout in milliseconds

	// SMTP specific fields
	SMTPUsername      string `gorm:"size:255" json:"smtp_username"`       // SMTP username for authentication
	SMTPPassword      string `gorm:"size:255" json:"smtp_password"`       // SMTP password for authentication
	SMTPUseTLS        bool   `gorm:"default:false" json:"smtp_use_tls"`   // Use TLS/SSL
	SMTPMailFrom      string `gorm:"size:255" json:"smtp_mail_from"`      // From address for test email
	SMTPMailTo        string `gorm:"size:255" json:"smtp_mail_to"`        // To address for test email
	SMTPCheckStartTLS bool   `gorm:"default:true" json:"smtp_check_starttls"` // Check STARTTLS support

	// SNMP specific fields
	SNMPCommunity    string `gorm:"size:255" json:"snmp_community"`    // SNMP community string (default: public)
	SNMPOID          string `gorm:"size:500" json:"snmp_oid"`           // SNMP OID to query
	SNMPVersion      string `gorm:"size:10" json:"snmp_version"`        // SNMP version: v1, v2c, v3
	SNMPExpectedValue string `gorm:"size:255" json:"snmp_expected_value"` // Expected value for comparison
	SNMPOperator     string `gorm:"size:10" json:"snmp_operator"`       // eq, ne, gt, lt, ge, le

	// SSL/TLS certificate specific fields
	SSLWarnDays    int    `gorm:"default:30" json:"ssl_warn_days"`    // Days before expiration to warn
	SSLCriticalDays int   `gorm:"default:7" json:"ssl_critical_days"`  // Days before expiration to mark as critical
	SSLGetChain    bool   `gorm:"default:true" json:"ssl_get_chain"`   // Get certificate chain information
	SSLCheck       bool   `gorm:"default:false" json:"ssl_check"`     // Enable SSL/TLS certificate monitoring for HTTPS

	// Alert channels association
	AlertChannelIDs string `gorm:"type:text" json:"alert_channel_ids"` // JSON array of alert channel IDs

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (MonitorTarget) TableName() string {
	return "monitor_targets"
}

type MonitorStatus struct {
	ID             uint32 `gorm:"primaryKey" json:"id"`
	TargetID       uint32 `gorm:"not null;index" json:"target_id"`
	Status         string `gorm:"size:50;not null" json:"status"` // up, down, unknown
	ResponseTime   int64  `json:"response_time"`                  // milliseconds
	Message        string `gorm:"type:text" json:"message"`
	CheckedAt      time.Time `gorm:"index" json:"checked_at"`
	UptimePercentage int32  `gorm:"default:0" json:"uptime_percentage"`

	// SSL Certificate info
	SSLDaysUntilExpiry *int    `gorm:"column:ssl_days_until_expiry" json:"ssl_days_until_expiry,omitempty"`
	SSLIssuer          *string `gorm:"column:ssl_issuer;size:255" json:"ssl_issuer,omitempty"`
	SSLSubject         *string `gorm:"column:ssl_subject;size:255" json:"ssl_subject,omitempty"`
	SSLEserial         *string `gorm:"column:ssl_serial;size:128" json:"ssl_serial,omitempty"`

	// DNS info
	DNSRecords *string `gorm:"column:dns_records;type:text" json:"dns_records,omitempty"` // JSON string of DNS records
	ResolvedIP *string `gorm:"column:resolved_ip;size:64" json:"resolved_ip,omitempty"`  // Resolved IP address

	// Additional check data (JSON string)
	Data *string `gorm:"column:data;type:text" json:"data,omitempty"` // Full check result data including certificate chain, etc.

	Target MonitorTarget `gorm:"foreignKey:TargetID" json:"target,omitempty"`
}

func (MonitorStatus) TableName() string {
	return "monitor_status"
}

type MonitorHistory struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	TargetID   uint32 `gorm:"not null;index" json:"target_id"`
	Status     string `gorm:"size:50;not null" json:"status"`
	ResponseTime int64 `json:"response_time"`
	Message    string `gorm:"type:text" json:"message"`
	CheckedAt  time.Time `gorm:"index" json:"checked_at"`
}

func (MonitorHistory) TableName() string {
	return "monitor_history"
}

type IPGeoCache struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	IP        string `gorm:"size:45;uniqueIndex;not null" json:"ip"`
	Country   string `gorm:"size:100" json:"country"`
	Region    string `gorm:"size:100" json:"region"`
	City      string `gorm:"size:100" json:"city"`
	ISP       string `gorm:"size:255" json:"isp"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (IPGeoCache) TableName() string {
	return "ip_geo_cache"
}

type DNSProvider struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Name      string `gorm:"size:255;not null" json:"name"`         // Provider name (e.g., "Google DNS")
	Server    string `gorm:"size:500;not null" json:"server"`       // DNS server address (e.g., 8.8.8.8:53)
	ServerType string `gorm:"size:10;not null" json:"server_type"`   // DNS protocol: udp, tcp, doh, dot
	IsDefault bool   `gorm:"default:false" json:"is_default"`       // Mark as default provider
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DNSProvider) TableName() string {
	return "dns_providers"
}