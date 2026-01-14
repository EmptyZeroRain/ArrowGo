package monitor

import (
	"context"
	"fmt"
	"net"
	"time"
)

type UDPChecker struct{}

func (c *UDPChecker) Check(ctx context.Context, target *MonitorTarget) (*CheckResult, error) {
	start := time.Now()

	address := fmt.Sprintf("%s:%d", target.Address, target.Port)

	conn, err := net.DialTimeout("udp", address, 10*time.Second)
	if err != nil {
		return &CheckResult{
			Status:       "down",
			ResponseTime: time.Since(start).Milliseconds(),
			Message:      fmt.Sprintf("UDP connection failed: %v", err),
		}, nil
	}
	defer conn.Close()

	responseTime := time.Since(start).Milliseconds()

	return &CheckResult{
		Status:       "up",
		ResponseTime: responseTime,
		Message:      "UDP connection successful",
	}, nil
}