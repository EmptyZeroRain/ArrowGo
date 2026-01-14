package monitor

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"monitor/internal/logger"
	"go.uber.org/zap"
)

type SSLChecker struct{}

func (c *SSLChecker) Check(ctx context.Context, target *MonitorTarget) (*CheckResult, error) {
	start := time.Now()

	// Extract host from address
	host := target.Address

	// Remove protocol prefix if present
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "ssl://")

	// Remove path if present (e.g., "www.example.com/path" -> "www.example.com")
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}

	// Split host and port if present
	if strings.Contains(host, ":") {
		h, p, err := net.SplitHostPort(host)
		if err == nil {
			host = h
			// If port was specified in address, use it; otherwise use target.Port
			if target.Port == 0 {
				// Parse the port from the address
				var portNum int
				fmt.Sscanf(p, "%d", &portNum)
				if portNum > 0 {
					target.Port = int32(portNum)
				}
			}
		}
	}

	// Default to port 443 if not specified
	port := target.Port
	if port == 0 {
		port = 443
	}

	address := fmt.Sprintf("%s:%d", host, port)

	logger.Debug("SSL check address parsed",
		zap.String("target", target.Name),
		zap.String("original_address", target.Address),
		zap.String("parsed_host", host),
		zap.Int("port", int(port)),
		zap.String("final_address", address),
	)

	// Create TLS connection
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", address, &tls.Config{
		InsecureSkipVerify: false, // We want to verify the certificate
	})

	if err != nil {
		logger.Warn("SSL/TLS connection failed",
			zap.String("target", target.Name),
			zap.String("address", address),
			zap.Error(err),
		)
		responseTime := time.Since(start).Milliseconds()

		return &CheckResult{
			Status:       "down",
			ResponseTime: responseTime,
			Message:      fmt.Sprintf("SSL/TLS connection failed: %v", err),
			Error: &ErrorDetails{
				Type:    "ssl_error",
				Message: err.Error(),
			},
		}, nil
	}
	defer conn.Close()

	// Get peer certificates
	state := conn.ConnectionState()
	certs := state.PeerCertificates

	if len(certs) == 0 {
		return &CheckResult{
			Status:       "down",
			ResponseTime: time.Since(start).Milliseconds(),
			Message:      "No certificates presented",
		}, nil
	}

	// Use the leaf certificate (end-entity cert)
	leafCert := certs[0]

	// Calculate days until expiry
	daysUntilExpiry := int(time.Until(leafCert.NotAfter).Hours() / 24)

	// Determine status based on certificate expiry
	status := "up"
	message := fmt.Sprintf("Certificate expires in %d days", daysUntilExpiry)

	if daysUntilExpiry < 0 {
		status = "down"
		message = fmt.Sprintf("Certificate expired %d days ago", -daysUntilExpiry)
	} else if daysUntilExpiry <= target.SSLCriticalDays {
		status = "critical"
		message = fmt.Sprintf("Certificate expires in %d days (CRITICAL)", daysUntilExpiry)
	} else if daysUntilExpiry <= target.SSLWarnDays {
		status = "warning"
		message = fmt.Sprintf("Certificate expires in %d days (WARNING)", daysUntilExpiry)
	}

	// Build certificate chain information
	var chainInfo []map[string]interface{}
	if target.SSLGetChain && len(certs) > 0 {
		for i, cert := range certs {
			certInfo := map[string]interface{}{
				"index":            i,
				"subject_cn":       cert.Subject.CommonName,
				"issuer_cn":        cert.Issuer.CommonName,
				"serial":           formatSerial(cert.SerialNumber),
				"not_before":       cert.NotBefore.Format(time.RFC3339),
				"not_after":        cert.NotAfter.Format(time.RFC3339),
				"days_until_expiry": int(time.Until(cert.NotAfter).Hours() / 24),
				"is_ca":            cert.IsCA,
				"key_usage":        cert.KeyUsage,
				"ext_key_usage":    cert.ExtKeyUsage,
				"signature_algorithm": cert.SignatureAlgorithm.String(),
			}

			// Add organization info if available
			if len(cert.Subject.Organization) > 0 {
				certInfo["subject_org"] = cert.Subject.Organization[0]
			}
			if len(cert.Issuer.Organization) > 0 {
				certInfo["issuer_org"] = cert.Issuer.Organization[0]
			}

			// Add DNS names if available
			if len(cert.DNSNames) > 0 {
				certInfo["dns_names"] = cert.DNSNames
			}

			chainInfo = append(chainInfo, certInfo)
		}
	}

	// Build certificate chain summary
	chainSummary := fmt.Sprintf("证书链包含 %d 个证书", len(certs))
	if len(chainInfo) > 0 {
		chainSummary += ": "
		var roles []string
		for i, cert := range certs {
			role := "终端证书"
			if cert.IsCA {
				if i == len(certs)-1 {
					role = "根证书"
				} else {
					role = "中间证书"
				}
			}
			roles = append(roles, fmt.Sprintf("%d.%s(%s)", i+1, cert.Subject.CommonName, role))
		}
		chainSummary += strings.Join(roles, " -> ")
	}

	// Build detailed message
	details := []string{
		message,
		fmt.Sprintf("颁发机构: %s", leafCert.Issuer.CommonName),
		fmt.Sprintf("主题: %s", leafCert.Subject.CommonName),
		fmt.Sprintf("序列号: %s", formatSerial(leafCert.SerialNumber)),
		fmt.Sprintf("生效日期: %s", leafCert.NotBefore.Format("2006-01-02")),
		fmt.Sprintf("过期日期: %s", leafCert.NotAfter.Format("2006-01-02")),
		chainSummary,
	}

	responseTime := time.Since(start).Milliseconds()

	logger.Info("SSL check completed",
		zap.String("target", target.Name),
		zap.String("host", host),
		zap.Int("days_until_expiry", daysUntilExpiry),
		zap.Int("chain_length", len(certs)),
		zap.String("status", status),
	)

	// Prepare response headers with certificate info
	headers := map[string]string{
		"issuer":        leafCert.Issuer.CommonName,
		"subject":       leafCert.Subject.CommonName,
		"serial":        formatSerial(leafCert.SerialNumber),
		"not_before":    leafCert.NotBefore.Format(time.RFC3339),
		"not_after":     leafCert.NotAfter.Format(time.RFC3339),
		"chain_count":   fmt.Sprintf("%d", len(certs)),
		"chain_summary": chainSummary,
	}

	// Prepare additional data with certificate chain
	data := make(map[string]interface{})
	if len(chainInfo) > 0 {
		data["certificate_chain"] = chainInfo
	}

	return &CheckResult{
		Status:       status,
		ResponseTime: responseTime,
		Message:      strings.Join(details, "\n"),
		Request: RequestDetails{
			Method: "SSL",
			URL:    address,
		},
		Response: ResponseDetails{
			StatusCode: daysUntilExpiry,
			Headers:    headers,
		},
		Data: data,
	}, nil
}

// formatSerial formats a serial number as hex string
func formatSerial(serial *big.Int) string {
	if serial == nil {
		return "unknown"
	}
	return fmt.Sprintf("%X", serial)
}

// decodePEMToCertificate converts PEM data to x509 certificate
func decodePEMToCertificate(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	return x509.ParseCertificate(block.Bytes)
}