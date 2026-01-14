package monitor

import (
	"context"
	"fmt"
	"net"
	"time"
)

type TCPChecker struct{}

func (c *TCPChecker) Check(ctx context.Context, target *MonitorTarget) (*CheckResult, error) {
	start := time.Now()

	address := fmt.Sprintf("%s:%d", target.Address, target.Port)

	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return &CheckResult{
			Status:       "down",
			ResponseTime: time.Since(start).Milliseconds(),
			Message:      fmt.Sprintf("TCP connection failed: %v", err),
		}, nil
	}
	defer conn.Close()

	responseTime := time.Since(start).Milliseconds()

	return &CheckResult{
		Status:       "up",
		ResponseTime: responseTime,
		Message:      "TCP connection successful",
	}, nil
}