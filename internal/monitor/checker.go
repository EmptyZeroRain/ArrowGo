package monitor

import (
	"context"
	"fmt"
)

type CheckResult struct {
	Status       string
	ResponseTime int64
	Message      string
	Data         map[string]interface{} // Additional data

	// 请求详情
	Request RequestDetails
	// 响应详情
	Response ResponseDetails
	// 错误详情
	Error *ErrorDetails
}

// RequestDetails 请求详情
type RequestDetails struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// ResponseDetails 响应详情
type ResponseDetails struct {
	StatusCode    int               `json:"status_code,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	Body          string            `json:"body,omitempty"`
	ContentLength int64             `json:"content_length,omitempty"`
}

// ErrorDetails 错误详情
type ErrorDetails struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
}

type MonitorTarget struct {
	ID       uint32
	Name     string
	Type     string
	Address  string
	Port     int32
	Interval int64
	Metadata map[string]string
	Enabled  bool

	// HTTP/HTTPS specific fields
	HTTPMethod          string            // GET, POST, PUT, DELETE, etc.
	HTTPHeaders         map[string]string // Custom headers
	HTTPBody            string            // Request body
	ResolvedHost        string            // Custom host resolution
	FollowRedirects     bool              // Follow 301/302 redirects
	MaxRedirects        int               // Maximum redirect depth
	ExpectedStatusCodes []int             // Expected status codes (e.g., [200, 201, 301, 302])

	// DNS specific fields
	DNSServer     string // Custom DNS server (e.g., 8.8.8.8:53)
	DNSServerName string // DNS server name
	DNSServerType string // DNS protocol type

	// PING specific fields
	PingCount   int // Number of ping packets
	PingSize    int // Size of ping packet
	PingTimeout int // Timeout in milliseconds

	// SMTP specific fields
	SMTPUsername      string // SMTP authentication username
	SMTPPassword      string // SMTP authentication password
	SMTPUseTLS        bool   // Use TLS/SSL
	SMTPMailFrom      string // From address for test
	SMTPMailTo        string // To address for test
	SMTPCheckStartTLS bool   // Check STARTTLS support

	// SNMP specific fields
	SNMPCommunity    string // SNMP community string
	SNMPOID          string // SNMP OID to query
	SNMPVersion      string // SNMP version: v1, v2c, v3
	SNMPExpectedValue string // Expected value for comparison
	SNMPOperator     string // Comparison operator: eq, ne, gt, lt, ge, le

	// SSL/TLS specific fields
	SSLWarnDays    int  // Days before expiration to warn
	SSLCriticalDays int  // Days before expiration to mark as critical
	SSLCheck       bool // Enable SSL/TLS certificate monitoring
	SSLGetChain    bool // Get certificate chain information
}

type Checker interface {
	Check(ctx context.Context, target *MonitorTarget) (*CheckResult, error)
}

func NewChecker(typ string) (Checker, error) {
	switch typ {
	case "http":
		return &HTTPChecker{}, nil
	case "https":
		return &HTTPSChecker{}, nil
	case "tcp":
		return &TCPChecker{}, nil
	case "udp":
		return &UDPChecker{}, nil
	case "dns":
		return &DNSChecker{}, nil
	case "ping", "icmp":
		return &PingCheckerWrapper{}, nil
	case "smtp", "smtps":
		return &SMTPCheckerWrapper{}, nil
	case "snmp":
		return &SNMPCheckerWrapper{}, nil
	case "ssl", "tls":
		return &SSLChecker{}, nil
	default:
		return nil, fmt.Errorf("unsupported monitor type: %s", typ)
	}
}