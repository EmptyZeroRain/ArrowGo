package monitor

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"monitor/internal/models"
)

// SMTPChecker implements SMTP server monitoring
type SMTPChecker struct {
	target *models.MonitorTarget
}

// NewSMTPChecker creates a new SMTP checker
func NewSMTPChecker(target *models.MonitorTarget) *SMTPChecker {
	return &SMTPChecker{target: target}
}

// Check performs an SMTP check
func (s *SMTPChecker) Check() (*CheckResult, error) {
	start := time.Now()

	// Build address
	host := s.target.Address
	port := int(s.target.Port)
	if port == 0 {
		if s.target.SMTPUseTLS {
			port = 465 // Default SMTPS port
		} else {
			port = 25 // Default SMTP port
		}
	}
	address := fmt.Sprintf("%s:%d", host, port)

	// Check basic TCP connection first
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return &CheckResult{
			Status: "down",
			Message: fmt.Sprintf("Connection failed: %v", err),
		}, err
	}
	conn.Close()
	elapsed := time.Since(start)

	// Perform SMTP handshake
	var result *CheckResult
	if s.target.SMTPUseTLS {
		result, err = s.checkSMTPS(address, host)
	} else {
		result, err = s.checkSMTP(address, host)
	}

	if err != nil {
		return result, err
	}

	result.ResponseTime = int64(elapsed.Milliseconds())
	return result, nil
}

// checkSMTP performs plain SMTP check
func (s *SMTPChecker) checkSMTP(address, host string) (*CheckResult, error) {
	// Connect to SMTP server
	client, err := smtp.Dial(address)
	if err != nil {
		return &CheckResult{
			Status:  "down",
			Message: fmt.Sprintf("SMTP connection failed: %v", err),
		}, err
	}
	defer client.Close()

	// Check STARTTLS if required
	if s.target.SMTPCheckStartTLS {
		ok, _ := client.Extension("STARTTLS")
		if !ok {
			return &CheckResult{
				Status:  "degraded",
				Message: "STARTTLS not supported by server",
			}, fmt.Errorf("STARTTLS not supported")
		}

		// Upgrade to TLS
		if err := client.StartTLS(&tls.Config{
			InsecureSkipVerify: false,
			ServerName:         host,
		}); err != nil {
			return &CheckResult{
				Status:  "degraded",
				Message: fmt.Sprintf("STARTTLS upgrade failed: %v", err),
			}, err
		}
	}

	// Authenticate if credentials provided
	if s.target.SMTPUsername != "" && s.target.SMTPPassword != "" {
		auth := smtp.PlainAuth("", s.target.SMTPUsername, s.target.SMTPPassword, host)
		if err := client.Auth(auth); err != nil {
			return &CheckResult{
				Status:  "down",
				Message: fmt.Sprintf("SMTP authentication failed: %v", err),
			}, err
		}
	}

	// Set sender if provided
	mailFrom := s.target.SMTPMailFrom
	if mailFrom == "" {
		mailFrom = "test@example.com"
	}
	if err := client.Mail(mailFrom); err != nil {
		return &CheckResult{
			Status:  "down",
			Message: fmt.Sprintf("MAIL FROM command failed: %v", err),
		}, err
	}

	// Set recipient if provided
	mailTo := s.target.SMTPMailTo
	if mailTo != "" {
		if err := client.Rcpt(mailTo); err != nil {
			return &CheckResult{
				Status:  "degraded",
				Message: fmt.Sprintf("RCPT TO command failed: %v", err),
			}, err
		}
	}

	// Reset connection
	if err := client.Reset(); err != nil {
		return &CheckResult{
			Status:  "degraded",
			Message: fmt.Sprintf("RSET command failed: %v", err),
		}, err
	}

	// Quit gracefully
	if err := client.Quit(); err != nil {
		return &CheckResult{
			Status:  "degraded",
			Message: fmt.Sprintf("QUIT command failed: %v", err),
		}, err
	}

	message := "SMTP server is operational"
	if s.target.SMTPUsername != "" {
		message += " (authenticated)"
	}
	if s.target.SMTPCheckStartTLS {
		message += " (STARTTLS verified)"
	}

	return &CheckResult{
		Status:  "up",
		Message: message,
		Data: map[string]interface{}{
			"starttls_supported": s.target.SMTPCheckStartTLS,
			"authenticated":       s.target.SMTPUsername != "",
			"host":                host,
		},
	}, nil
}

// checkSMTPS performs SMTP over TLS/SSL check
func (s *SMTPChecker) checkSMTPS(address, host string) (*CheckResult, error) {
	// Create TLS connection
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second},
		"tcp",
		address,
		&tls.Config{
			InsecureSkipVerify: false,
			ServerName:         host,
		},
	)
	if err != nil {
		return &CheckResult{
			Status:  "down",
			Message: fmt.Sprintf("SMTPS connection failed: %v", err),
		}, err
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return &CheckResult{
			Status:  "down",
			Message: fmt.Sprintf("SMTPS client creation failed: %v", err),
		}, err
	}
	defer client.Quit()

	// Authenticate if credentials provided
	if s.target.SMTPUsername != "" && s.target.SMTPPassword != "" {
		auth := smtp.PlainAuth("", s.target.SMTPUsername, s.target.SMTPPassword, host)
		if err := client.Auth(auth); err != nil {
			return &CheckResult{
				Status:  "down",
				Message: fmt.Sprintf("SMTPS authentication failed: %v", err),
			}, err
		}
	}

	// Set sender if provided
	mailFrom := s.target.SMTPMailFrom
	if mailFrom == "" {
		mailFrom = "test@example.com"
	}
	if err := client.Mail(mailFrom); err != nil {
		return &CheckResult{
			Status:  "down",
			Message: fmt.Sprintf("MAIL FROM command failed: %v", err),
		}, err
	}

	// Set recipient if provided
	mailTo := s.target.SMTPMailTo
	if mailTo != "" {
		if err := client.Rcpt(mailTo); err != nil {
			return &CheckResult{
				Status:  "degraded",
				Message: fmt.Sprintf("RCPT TO command failed: %v", err),
			}, err
		}
	}

	// Reset connection
	if err := client.Reset(); err != nil {
		return &CheckResult{
			Status:  "degraded",
			Message: fmt.Sprintf("RSET command failed: %v", err),
		}, err
	}

	message := "SMTPS server is operational"
	if s.target.SMTPUsername != "" {
		message += " (authenticated)"
	}

	return &CheckResult{
		Status:  "up",
		Message: message,
		Data: map[string]interface{}{
			"tls":           true,
			"authenticated": s.target.SMTPUsername != "",
			"host":          host,
		},
	}, nil
}

// GetSMTPCapabilities retrieves SMTP server capabilities
func GetSMTPCapabilities(address string) (map[string]bool, error) {
	client, err := smtp.Dial(address)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// Get EHLO response
	err = client.Hello("localhost")
	if err != nil {
		return nil, err
	}

	// Check supported extensions
	capabilities := make(map[string]bool)

	if ok, _ := client.Extension("STARTTLS"); ok {
		capabilities["starttls"] = true
	}

	if ok, _ := client.Extension("AUTH"); ok {
		capabilities["auth"] = true
	}

	if ok, _ := client.Extension("SIZE"); ok {
		capabilities["size"] = true
	}

	if ok, _ := client.Extension("8BITMIME"); ok {
		capabilities["8bitmime"] = true
	}

	if ok, _ := client.Extension("PIPELINING"); ok {
		capabilities["pipelining"] = true
	}

	return capabilities, nil
}

// ValidateSMTPAddress validates an email address format
func ValidateSMTPAddress(address string) bool {
	parts := strings.Split(address, "@")
	if len(parts) != 2 {
		return false
	}
	if parts[0] == "" || parts[1] == "" {
		return false
	}
	if !strings.Contains(parts[1], ".") {
		return false
	}
	return true
}

// SMTPCheckerWrapper implements the Checker interface for SMTP monitoring
type SMTPCheckerWrapper struct{}

func (w *SMTPCheckerWrapper) Check(ctx context.Context, target *MonitorTarget) (*CheckResult, error) {
	modelTarget := &models.MonitorTarget{
		Address:           target.Address,
		Port:              target.Port,
		SMTPUsername:      target.SMTPUsername,
		SMTPPassword:      target.SMTPPassword,
		SMTPUseTLS:        target.SMTPUseTLS,
		SMTPMailFrom:      target.SMTPMailFrom,
		SMTPMailTo:        target.SMTPMailTo,
		SMTPCheckStartTLS: target.SMTPCheckStartTLS,
	}

	checker := NewSMTPChecker(modelTarget)
	return checker.Check()
}