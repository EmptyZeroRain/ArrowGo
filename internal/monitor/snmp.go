package monitor

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gosnmp/gosnmp"
	"monitor/internal/models"
)

// SNMPChecker implements SNMP monitoring
type SNMPChecker struct {
	target *models.MonitorTarget
}

// NewSNMPChecker creates a new SNMP checker
func NewSNMPChecker(target *models.MonitorTarget) *SNMPChecker {
	return &SNMPChecker{target: target}
}

// Check performs an SNMP check
func (s *SNMPChecker) Check(ctx context.Context, target *MonitorTarget) (*CheckResult, error) {
	start := time.Now()

	// Set default values
	community := target.SNMPCommunity
	if community == "" {
		community = "public"
	}

	oid := target.SNMPOID
	if oid == "" {
		oid = "1.3.6.1.2.1.1.1.0" // sysDescr.0 - system description
	}

	// Parse SNMP version
	var version gosnmp.SnmpVersion
	switch target.SNMPVersion {
	case "v2c", "v2":
		version = gosnmp.Version2c
	case "v3":
		version = gosnmp.Version3
	default:
		version = gosnmp.Version1
	}

	// Create SNMP client
	client := gosnmp.GoSNMP{
		Target:    target.Address,
		Port:      uint16(target.Port),
		Community: community,
		Version:   version,
		Timeout:   time.Duration(target.PingTimeout) * time.Millisecond,
	}

	if client.Port == 0 {
		client.Port = 161 // Default SNMP port
	}

	// Perform SNMP GET
	oids := []string{oid}
	result, err := client.Get(oids)
	if err != nil {
		return &CheckResult{
			Status:  "down",
			Message: fmt.Sprintf("SNMP query failed: %v", err),
		}, err
	}

	// Check if we got results
	if len(result.Variables) == 0 {
		return &CheckResult{
			Status:  "down",
			Message: "No SNMP response received",
		}, fmt.Errorf("empty SNMP response")
	}

	// Get the value
	variable := result.Variables[0]
	var actualValue string

	// Extract value based on type
	switch variable.Type {
	case gosnmp.Integer:
		actualValue = fmt.Sprintf("%d", variable.Value.(uint))
	case gosnmp.OctetString:
		actualValue = string(variable.Value.([]byte))
	case gosnmp.ObjectIdentifier:
		actualValue = variable.Value.(string)
	default:
		actualValue = fmt.Sprintf("%v", variable.Value)
	}

	// Check against expected value if operator specified
	status := "up"
	message := fmt.Sprintf("SNMP query successful - OID: %s, Value: %s", oid, actualValue)

	if target.SNMPOperator != "" && target.SNMPExpectedValue != "" {
		expectedValue := target.SNMPExpectedValue
		operator := target.SNMPOperator

		// Try to parse expected value as number
		var expectedNum float64
		var actualNum float64
		expectedIsNum := false
		actualIsNum := false

		if num, err := strconv.ParseFloat(expectedValue, 64); err == nil {
			expectedNum = num
			expectedIsNum = true
		}
		if num, err := strconv.ParseFloat(actualValue, 64); err == nil {
			actualNum = num
			actualIsNum = true
		}

		// Perform comparison
		shouldAlert := false
		if expectedIsNum && actualIsNum {
			// Numeric comparison
			switch operator {
			case "eq":
				shouldAlert = actualNum != expectedNum
			case "ne":
				shouldAlert = actualNum == expectedNum
			case "gt":
				shouldAlert = actualNum <= expectedNum
			case "lt":
				shouldAlert = actualNum >= expectedNum
			case "ge":
				shouldAlert = actualNum < expectedNum
			case "le":
				shouldAlert = actualNum > expectedNum
			}
		} else {
			// String comparison
			switch operator {
			case "eq":
				shouldAlert = actualValue != expectedValue
			case "ne":
				shouldAlert = actualValue == expectedValue
			default:
				// For string, only eq and ne make sense
				shouldAlert = actualValue != expectedValue
			}
		}

		if shouldAlert {
			status = "down"
			message = fmt.Sprintf("SNMP value check failed - Expected: %s %s %s, Got: %s",
				expectedValue, operator, oid, actualValue)
		} else {
			message = fmt.Sprintf("SNMP value check passed - Expected: %s %s %s, Got: %s",
				expectedValue, operator, oid, actualValue)
		}
	}

	elapsed := time.Since(start)

	return &CheckResult{
		Status:       status,
		ResponseTime: int64(elapsed.Milliseconds()),
		Message:     message,
		Data: map[string]interface{}{
			"oid":        oid,
			"value":      actualValue,
			"type":       variable.Type.String(),
			"community":  community,
			"version":    target.SNMPVersion,
		},
	}, nil
}

// SNMPCheckerWrapper implements the Checker interface for SNMP monitoring
type SNMPCheckerWrapper struct{}

func (w *SNMPCheckerWrapper) Check(ctx context.Context, target *MonitorTarget) (*CheckResult, error) {
	modelTarget := &models.MonitorTarget{
		Address:       target.Address,
		Port:          target.Port,
		SNMPCommunity: target.SNMPCommunity,
		SNMPOID:       target.SNMPOID,
		SNMPVersion:   target.SNMPVersion,
		SNMPExpectedValue: target.SNMPExpectedValue,
		SNMPOperator:  target.SNMPOperator,
		PingTimeout:   target.PingTimeout,
	}

	checker := NewSNMPChecker(modelTarget)
	return checker.Check(ctx, target)
}
