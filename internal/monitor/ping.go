package monitor

import (
	"context"
	"fmt"
	"math"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"monitor/internal/models"
)

// PingChecker implements ICMP ping monitoring
type PingChecker struct {
	target *models.MonitorTarget
}

// NewPingChecker creates a new ping checker
func NewPingChecker(target *models.MonitorTarget) *PingChecker {
	return &PingChecker{target: target}
}

// Check performs a ping check
func (p *PingChecker) Check() (*CheckResult, error) {
	// Get ping parameters
	count := p.target.PingCount
	if count <= 0 {
		count = 4
	}

	size := p.target.PingSize
	if size <= 0 {
		size = 32
	}

	timeout := time.Duration(p.target.PingTimeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	// Perform ping based on OS
	var packetLoss int
	var avgTime time.Duration
	var err error

	if runtime.GOOS == "windows" {
		packetLoss, avgTime, err = p.pingWindows(count, size, timeout)
	} else {
		packetLoss, avgTime, err = p.pingUnix(count, size, timeout)
	}

	if err != nil {
		return &CheckResult{
			Status: "down",
			Message: fmt.Sprintf("Ping failed: %v", err),
		}, err
	}

	status := "up"
	message := fmt.Sprintf("Ping successful - Packet loss: %d%%, Avg time: %dms", packetLoss, avgTime.Milliseconds())

	// Consider down if packet loss is 100%
	if packetLoss == 100 {
		status = "down"
		message = fmt.Sprintf("Ping failed - 100%% packet loss")
	} else if packetLoss > 50 {
		status = "degraded"
		message = fmt.Sprintf("Ping degraded - Packet loss: %d%%, Avg time: %dms", packetLoss, avgTime.Milliseconds())
	}

	return &CheckResult{
		Status:      status,
		ResponseTime: int64(avgTime.Milliseconds()),
		Message:     message,
		Data: map[string]interface{}{
			"packet_loss": packetLoss,
			"avg_time":    avgTime.Milliseconds(),
			"packets_sent": count,
			"packets_received": count - (count * packetLoss / 100),
		},
	}, nil
}

// pingWindows performs ping on Windows
func (p *PingChecker) pingWindows(count, size int, timeout time.Duration) (int, time.Duration, error) {
	args := []string{
		"-n", fmt.Sprintf("%d", count),
		"-l", fmt.Sprintf("%d", size),
		"-w", fmt.Sprintf("%d", timeout.Milliseconds()),
		p.target.Address,
	}

	cmd := exec.Command("ping", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 100, 0, fmt.Errorf("ping command failed: %w", err)
	}

	// Parse output
	return p.parsePingOutput(string(output))
}

// pingUnix performs ping on Unix-like systems (Linux, macOS)
func (p *PingChecker) pingUnix(count, size int, timeout time.Duration) (int, time.Duration, error) {
	args := []string{
		"-c", fmt.Sprintf("%d", count),
		"-s", fmt.Sprintf("%d", size),
		"-W", fmt.Sprintf("%d", int(timeout.Seconds())),
		p.target.Address,
	}

	cmd := exec.Command("ping", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 100, 0, fmt.Errorf("ping command failed: %w", err)
	}

	// Parse output
	return p.parsePingOutput(string(output))
}

// parsePingOutput parses ping command output
func (p *PingChecker) parsePingOutput(output string) (int, time.Duration, error) {
	lines := strings.Split(output, "\n")

	var packetsSent, packetsReceived int
	var total time.Duration

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse packet statistics
		// Windows: "Packets: Sent = 4, Received = 4, Lost = 0 (0% loss)"
		// Linux/Unix: "4 packets transmitted, 4 received, 0% packet loss"
		if strings.Contains(line, "packets") || strings.Contains(line, "Packets") {
			fields := strings.Fields(line)
			for i, field := range fields {
				// Extract sent/received
				if val, err := extractNumber(field); err == nil {
					if strings.Contains(strings.ToLower(line), "sent") || i == 0 {
						packetsSent = int(val)
					}
					if strings.Contains(strings.ToLower(line), "received") || i == 1 {
						packetsReceived = int(val)
					}
				}
			}
		}

		// Parse time values
		// Windows: "Average = 2ms" or "平均 = 2ms"
		// Linux/Unix: "rtt min/avg/max/mdev = 1.234/2.345/3.456/0.123 ms"
		if strings.Contains(strings.ToLower(line), "avg") || strings.Contains(line, "Average") || strings.Contains(line, "rtt") {
			if times := parseTimeValues(line); len(times) > 0 {
				// Get average (usually the second value in rtt min/avg/max)
				if len(times) >= 2 {
					total = times[1]
				} else if len(times) == 1 {
					total = times[0]
				}
			}
		}

		// Parse individual ping times (for Unix systems)
		// Example: "64 bytes from 192.168.1.1: icmp_seq=1 ttl=64 time=0.123 ms"
		if strings.Contains(line, "time=") {
			if idx := strings.Index(line, "time="); idx != -1 {
				timeStr := line[idx+5:]
				timeStr = strings.Fields(timeStr)[0]
				if dur, err := time.ParseDuration(timeStr + "ms"); err == nil {
					total += dur
				}
			}
		}
	}

	// Calculate packet loss
	packetLoss := 0
	if packetsSent > 0 {
		packetLoss = int(math.Ceil(float64(packetsSent-packetsReceived) * 100 / float64(packetsSent)))
	}

	avgTime := total
	if packetsReceived > 0 {
		avgTime = total / time.Duration(packetsReceived)
	}

	return packetLoss, avgTime, nil
}

// CheckICMP performs a raw ICMP ping (alternative method)
func CheckICMP(host string, timeout time.Duration) (time.Duration, error) {
	// Resolve IP address
	ipAddr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve address: %w", err)
	}

	start := time.Now()

	// Create a connection
	conn, err := net.DialIP("ip:icmp", nil, ipAddr)
	if err != nil {
		return 0, fmt.Errorf("failed to create ICMP connection: %w", err)
	}
	defer conn.Close()

	// Set deadline
	conn.SetDeadline(time.Now().Add(timeout))

	// Send ICMP packet (simple echo request)
	packet := make([]byte, 8)
	packet[0] = 8 // Echo request
	packet[1] = 0 // Code
	packet[2] = 0 // Checksum (will be filled)
	packet[3] = 0
	packet[4] = 0 // Identifier
	packet[5] = 0
	packet[6] = 0 // Sequence
	packet[7] = 0

	_, err = conn.Write(packet)
	if err != nil {
		return 0, fmt.Errorf("failed to send ICMP packet: %w", err)
	}

	// Read response
	reply := make([]byte, 1024)
	_, err = conn.Read(reply)
	if err != nil {
		return 0, fmt.Errorf("failed to read ICMP response: %w", err)
	}

	elapsed := time.Since(start)
	return elapsed, nil
}

// Helper functions

func extractNumber(s string) (int, error) {
	s = strings.Trim(s, ",=:;()")
	s = strings.TrimSpace(s)
	var num int
	_, err := fmt.Sscanf(s, "%d", &num)
	return num, err
}

func parseTimeValues(line string) []time.Duration {
	var times []time.Duration

	fields := strings.Fields(line)
	for _, field := range fields {
		// Look for patterns like "2.345ms" or "2ms"
		field = strings.Trim(field, ",=:;()/")
		if strings.HasSuffix(field, "ms") {
			msStr := strings.TrimSuffix(field, "ms")
			var ms float64
			if _, err := fmt.Sscanf(msStr, "%f", &ms); err == nil {
				times = append(times, time.Duration(ms*float64(time.Millisecond)))
			}
		}
	}

	return times
}

// PingCheckerWrapper implements the Checker interface for PING monitoring
type PingCheckerWrapper struct{}

func (w *PingCheckerWrapper) Check(ctx context.Context, target *MonitorTarget) (*CheckResult, error) {
	modelTarget := &models.MonitorTarget{
		PingCount:   target.PingCount,
		PingSize:    target.PingSize,
		PingTimeout: target.PingTimeout,
		Address:     target.Address,
	}

	checker := NewPingChecker(modelTarget)
	return checker.Check()
}