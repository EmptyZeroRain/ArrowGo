package monitor

import (
	"context"
	"fmt"
	"time"

	"monitor/internal/logger"
	"go.uber.org/zap"
)

// HTTPSChecker combines HTTP and SSL certificate checking
type HTTPSChecker struct{}

func (c *HTTPSChecker) Check(ctx context.Context, target *MonitorTarget) (*CheckResult, error) {
	start := time.Now()

	logger.Debug("Starting HTTPS check",
		zap.String("target", target.Name),
		zap.String("address", target.Address),
		zap.Bool("ssl_check", target.SSLCheck),
	)

	// First, perform SSL certificate check if enabled
	var sslResult *CheckResult
	if target.SSLCheck {
		sslChecker := &SSLChecker{}
		var err error
		sslResult, err = sslChecker.Check(ctx, target)
		if err != nil {
			logger.Warn("SSL certificate check failed",
				zap.String("target", target.Name),
				zap.Error(err),
			)
		}
	}

	// Perform HTTP check
	// Create a modified HTTPChecker that uses TLS config
	httpChecker := &HTTPChecker{}
	httpResult, err := httpChecker.Check(ctx, target)
	if err != nil {
		logger.Error("HTTP check failed",
			zap.String("target", target.Name),
			zap.Error(err),
		)
		return httpResult, err
	}

	// Merge SSL certificate information into HTTP result
	if sslResult != nil {
		if httpResult.Data == nil {
			httpResult.Data = make(map[string]interface{})
		}

		// Copy SSL certificate data
		if certChain, ok := sslResult.Data["certificate_chain"]; ok {
			httpResult.Data["certificate_chain"] = certChain
		}
		if certInfo, ok := sslResult.Data["certificate_info"]; ok {
			httpResult.Data["certificate_info"] = certInfo
		}

		// Copy SSL headers for database storage
		if httpResult.Response.Headers == nil {
			httpResult.Response.Headers = make(map[string]string)
		}

		// Map SSL result headers to HTTP result headers for service.go to save
		if issuer, ok := sslResult.Response.Headers["issuer"]; ok {
			httpResult.Response.Headers["ssl_issuer"] = issuer
		}
		if subject, ok := sslResult.Response.Headers["subject"]; ok {
			httpResult.Response.Headers["ssl_subject"] = subject
		}
		if serial, ok := sslResult.Response.Headers["serial"]; ok {
			httpResult.Response.Headers["ssl_serial"] = serial
		}

		// Add days_until_expiry from certificate chain data
		if certChain, ok := sslResult.Data["certificate_chain"]; ok {
			if chain, ok := certChain.([]map[string]interface{}); ok && len(chain) > 0 {
				if days, ok := chain[0]["days_until_expiry"].(int); ok {
					httpResult.Response.Headers["days_until_expiry"] = fmt.Sprintf("%d", days)
				}
			}
		}

		// Update message to include SSL info
		if sslResult.Status == "down" {
			httpResult.Status = "down"
			httpResult.Message = fmt.Sprintf("%s (SSL: %s)", httpResult.Message, sslResult.Message)
		} else {
			httpResult.Message = fmt.Sprintf("%s | SSL: %s", httpResult.Message, sslResult.Message)
		}

		logger.Debug("Merged SSL certificate data into HTTPS result",
			zap.String("target", target.Name),
			zap.Int("cert_chain_length", len(httpResult.Data)),
			zap.Bool("has_ssl_headers", len(httpResult.Response.Headers) > 0),
		)
	}

	responseTime := time.Since(start).Milliseconds()
	httpResult.ResponseTime = responseTime

	logger.Debug("HTTPS check completed",
		zap.String("target", target.Name),
		zap.String("status", httpResult.Status),
		zap.Int64("response_time", responseTime),
		zap.Bool("ssl_data_present", len(httpResult.Data) > 0),
	)

	return httpResult, nil
}
