package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	dnsresolver "monitor/pkg/dns"

	"monitor/internal/logger"
	"go.uber.org/zap"
)

type DNSChecker struct{}

type DNSRecordInfo struct {
	Type  string   `json:"type"`
	Value []string `json:"value"`
	TTL   uint32   `json:"ttl,omitempty"`
}

func (c *DNSChecker) Check(ctx context.Context, target *MonitorTarget) (*CheckResult, error) {
	start := time.Now()

	// Default DNS server configuration
	dnsServer := target.DNSServer
	dnsServerType := target.DNSServerType

	// If no custom DNS server, use system default
	if dnsServer == "" {
		dnsServer = "8.8.8.8:53"
		dnsServerType = "udp"
	}

	// Default to UDP if type not specified
	if dnsServerType == "" {
		dnsServerType = "udp"
	}

	logger.Debug("DNS lookup starting",
		zap.String("target", target.Name),
		zap.String("address", target.Address),
		zap.String("dns_server", dnsServer),
		zap.String("dns_server_name", target.DNSServerName),
		zap.String("dns_server_type", dnsServerType),
	)

	// Create resolver
	resolver := dnsresolver.NewResolver(dnsServer, dnsresolver.DNSType(dnsServerType))

	// Perform lookup
	result, err := resolver.Lookup(ctx, target.Address)
	if err != nil {
		status := "down"
		message := fmt.Sprintf("DNS lookup failed: %v", err)

		logger.Warn("DNS lookup failed",
			zap.String("target", target.Name),
			zap.String("address", target.Address),
			zap.String("dns_server", dnsServer),
			zap.String("dns_server_type", dnsServerType),
			zap.Error(err),
		)

		return &CheckResult{
			Status:       status,
			ResponseTime: time.Since(start).Milliseconds(),
			Message:      message,
			Error: &ErrorDetails{
				Type:    "dns_error",
				Message: err.Error(),
			},
		}, nil
	}

	// Convert result to DNSRecordInfo format
	allRecords := make([]DNSRecordInfo, 0)

	if len(result.A) > 0 {
		allRecords = append(allRecords, DNSRecordInfo{
			Type:  "A",
			Value: result.A,
		})
	}
	if len(result.AAAA) > 0 {
		allRecords = append(allRecords, DNSRecordInfo{
			Type:  "AAAA",
			Value: result.AAAA,
		})
	}
	if len(result.CNAME) > 0 {
		allRecords = append(allRecords, DNSRecordInfo{
			Type:  "CNAME",
			Value: result.CNAME,
		})
	}
	if len(result.MX) > 0 {
		allRecords = append(allRecords, DNSRecordInfo{
			Type:  "MX",
			Value: result.MX,
		})
	}
	if len(result.TXT) > 0 {
		allRecords = append(allRecords, DNSRecordInfo{
			Type:  "TXT",
			Value: result.TXT,
		})
	}
	if len(result.NS) > 0 {
		allRecords = append(allRecords, DNSRecordInfo{
			Type:  "NS",
			Value: result.NS,
		})
	}

	responseTime := time.Since(start).Milliseconds()

	// Build detailed message
	var message strings.Builder
	totalRecords := 0
	for _, record := range allRecords {
		totalRecords += len(record.Value)
		message.WriteString(fmt.Sprintf("%s: %d record(s); ", record.Type, len(record.Value)))
	}

	// Add server info to message
	if target.DNSServerName != "" {
		message.WriteString(fmt.Sprintf("via %s (%s); ", target.DNSServerName, dnsServerType))
	} else {
		message.WriteString(fmt.Sprintf("via %s (%s); ", dnsServer, dnsServerType))
	}

	// Determine overall status
	status := "up"
	if len(result.A) == 0 && len(result.AAAA) == 0 && len(result.CNAME) == 0 {
		status = "warning"
	}

	logger.Info("DNS lookup completed",
		zap.String("target", target.Name),
		zap.String("address", target.Address),
		zap.String("dns_server", dnsServer),
		zap.String("dns_server_type", dnsServerType),
		zap.Int("total_records", totalRecords),
		zap.Int64("response_time", responseTime),
		zap.String("status", status),
	)

	// Convert records to JSON for storage
	recordsJSON, _ := json.Marshal(allRecords)

	return &CheckResult{
		Status:       status,
		ResponseTime: responseTime,
		Message:      message.String(),
		Request: RequestDetails{
			Method: "DNS",
			URL:    target.Address,
		},
		Response: ResponseDetails{
			Headers: map[string]string{
				"dns_server":      dnsServer,
				"dns_server_name": target.DNSServerName,
				"dns_server_type": dnsServerType,
				"a_count":         fmt.Sprintf("%d", len(result.A)),
				"aaaa_count":       fmt.Sprintf("%d", len(result.AAAA)),
				"total_types":      fmt.Sprintf("%d", len(allRecords)),
			},
			Body: string(recordsJSON),
		},
	}, nil
}

// Fallback to system DNS if no custom server specified
func (c *DNSChecker) lookupWithSystemDNS(ctx context.Context, domain string) (*dnsresolver.DNSQueryResult, error) {
	timeout := 10 * time.Second

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var resolver net.Resolver

	ips, err := resolver.LookupIPAddr(ctx, domain)
	if err != nil {
		return nil, err
	}

	result := &dnsresolver.DNSQueryResult{}

	// Separate IPv4 and IPv6
	for _, ip := range ips {
		if ip.IP.To4() != nil {
			result.A = append(result.A, ip.IP.String())
		} else {
			result.AAAA = append(result.AAAA, ip.IP.String())
		}
	}

	return result, nil
}
