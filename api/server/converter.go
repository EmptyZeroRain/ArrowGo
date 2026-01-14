package server

import (
	"encoding/json"
	"fmt"
	"strings"

	"monitor/internal/models"
	"monitor/internal/monitor"
)

// ConvertAddRequestToModel 将 AddMonitorRequest 转换为数据库模型
func ConvertAddRequestToModel(req AddMonitorRequest) (*models.MonitorTarget, error) {
	var metadata string
	if req.Metadata != nil {
		bytes, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, err
		}
		metadata = string(bytes)
	}

	var httpHeaders string
	if req.HTTPHeaders != nil {
		bytes, err := json.Marshal(req.HTTPHeaders)
		if err != nil {
			return nil, err
		}
		httpHeaders = string(bytes)
	}

	target := &models.MonitorTarget{
		Name:     req.Name,
		Type:     req.Type,
		Address:  req.Address,
		Port:     req.Port,
		Interval: req.Interval,
		Metadata: metadata,
		Enabled:  req.Enabled,
		// HTTP/HTTPS specific fields
		HTTPMethod:          req.HTTPMethod,
		HTTPHeaders:         httpHeaders,
		HTTPBody:            req.HTTPBody,
		ResolvedHost:        req.ResolvedHost,
		FollowRedirects:     req.FollowRedirects,
		MaxRedirects:        req.MaxRedirects,
		ExpectedStatusCodes: req.ExpectedStatusCodes,
		// DNS specific fields
		DNSServer:     req.DNSServer,
		DNSServerName: req.DNSServerName,
		DNSServerType: req.DNSServerType,
		// PING specific fields
		PingCount:   req.PingCount,
		PingSize:    req.PingSize,
		PingTimeout: req.PingTimeout,
		// SMTP specific fields
		SMTPUsername:      req.SMTPUsername,
		SMTPPassword:      req.SMTPPassword,
		SMTPUseTLS:        req.SMTPUseTLS,
		SMTPMailFrom:      req.SMTPMailFrom,
		SMTPMailTo:        req.SMTPMailTo,
		SMTPCheckStartTLS: req.SMTPCheckStartTLS,
		// SNMP specific fields
		SNMPCommunity:    req.SNMPCommunity,
		SNMPOID:          req.SNMPOID,
		SNMPVersion:      req.SNMPVersion,
		SNMPExpectedValue: req.SNMPExpectedValue,
		SNMPOperator:     req.SNMPOperator,
		// SSL/TLS specific fields
		SSLWarnDays:     req.SSLWarnDays,
		SSLCriticalDays: req.SSLCriticalDays,
		SSLCheck:       req.SSLCheck,
		SSLGetChain:    req.SSLGetChain,
	}

	return target, nil
}

// UpdateModelFromRequest 使用请求更新模型
func UpdateModelFromRequest(target *models.MonitorTarget, req AddMonitorRequest) error {
	target.Name = req.Name
	target.Type = req.Type
	target.Address = req.Address
	target.Port = req.Port
	target.Interval = req.Interval
	target.Enabled = req.Enabled

	var metadata string
	if req.Metadata != nil {
		bytes, err := json.Marshal(req.Metadata)
		if err != nil {
			return err
		}
		metadata = string(bytes)
	}
	target.Metadata = metadata

	var httpHeaders string
	if req.HTTPHeaders != nil {
		bytes, err := json.Marshal(req.HTTPHeaders)
		if err != nil {
			return err
		}
		httpHeaders = string(bytes)
	}
	target.HTTPHeaders = httpHeaders

	// HTTP/HTTPS specific fields
	target.HTTPMethod = req.HTTPMethod
	target.HTTPBody = req.HTTPBody
	target.ResolvedHost = req.ResolvedHost
	target.FollowRedirects = req.FollowRedirects
	target.MaxRedirects = req.MaxRedirects
	target.ExpectedStatusCodes = req.ExpectedStatusCodes
	// DNS specific fields
	target.DNSServer = req.DNSServer
	target.DNSServerName = req.DNSServerName
	target.DNSServerType = req.DNSServerType
	// PING specific fields
	target.PingCount = req.PingCount
	target.PingSize = req.PingSize
	target.PingTimeout = req.PingTimeout
	// SMTP specific fields
	target.SMTPUsername = req.SMTPUsername
	target.SMTPPassword = req.SMTPPassword
	target.SMTPUseTLS = req.SMTPUseTLS
	target.SMTPMailFrom = req.SMTPMailFrom
	target.SMTPMailTo = req.SMTPMailTo
	target.SMTPCheckStartTLS = req.SMTPCheckStartTLS
	// SNMP specific fields
	target.SNMPCommunity = req.SNMPCommunity
	target.SNMPOID = req.SNMPOID
	target.SNMPVersion = req.SNMPVersion
	target.SNMPExpectedValue = req.SNMPExpectedValue
	target.SNMPOperator = req.SNMPOperator
	// SSL/TLS specific fields
	target.SSLWarnDays = req.SSLWarnDays
	target.SSLCriticalDays = req.SSLCriticalDays
	target.SSLCheck = req.SSLCheck
	target.SSLGetChain = req.SSLGetChain

	return nil
}

// ConvertModelToMonitorTarget 将数据库模型转换为监控目标
func ConvertModelToMonitorTarget(target models.MonitorTarget) (*monitor.MonitorTarget, error) {
	var metadata map[string]string
	if target.Metadata != "" {
		if err := json.Unmarshal([]byte(target.Metadata), &metadata); err != nil {
			return nil, err
		}
	}

	// Parse HTTP headers
	var httpHeaders map[string]string
	if target.HTTPHeaders != "" {
		if err := json.Unmarshal([]byte(target.HTTPHeaders), &httpHeaders); err != nil {
			return nil, err
		}
	}

	// Parse expected status codes
	var expectedStatusCodes []int
	if target.ExpectedStatusCodes != "" {
		codesStr := strings.Split(target.ExpectedStatusCodes, ",")
		for _, codeStr := range codesStr {
			codeStr = strings.TrimSpace(codeStr)
			if codeStr != "" {
				var code int
				if _, err := fmt.Sscanf(codeStr, "%d", &code); err == nil {
					expectedStatusCodes = append(expectedStatusCodes, code)
				}
			}
		}
	}

	monitorTarget := &monitor.MonitorTarget{
		ID:       target.ID,
		Name:     target.Name,
		Type:     target.Type,
		Address:  target.Address,
		Port:     target.Port,
		Interval: target.Interval,
		Metadata: metadata,
		Enabled:  target.Enabled,
		// HTTP/HTTPS specific fields
		HTTPMethod:          target.HTTPMethod,
		HTTPHeaders:         httpHeaders,
		HTTPBody:            target.HTTPBody,
		ResolvedHost:        target.ResolvedHost,
		FollowRedirects:     target.FollowRedirects,
		MaxRedirects:        target.MaxRedirects,
		ExpectedStatusCodes: expectedStatusCodes,
		// DNS specific fields
		DNSServer:     target.DNSServer,
		DNSServerName: target.DNSServerName,
		DNSServerType: target.DNSServerType,
		// PING specific fields
		PingCount:   target.PingCount,
		PingSize:    target.PingSize,
		PingTimeout: target.PingTimeout,
		// SMTP specific fields
		SMTPUsername:      target.SMTPUsername,
		SMTPPassword:      target.SMTPPassword,
		SMTPUseTLS:        target.SMTPUseTLS,
		SMTPMailFrom:      target.SMTPMailFrom,
		SMTPMailTo:        target.SMTPMailTo,
		SMTPCheckStartTLS: target.SMTPCheckStartTLS,
		// SNMP specific fields
		SNMPCommunity:    target.SNMPCommunity,
		SNMPOID:          target.SNMPOID,
		SNMPVersion:      target.SNMPVersion,
		SNMPExpectedValue: target.SNMPExpectedValue,
		SNMPOperator:     target.SNMPOperator,
		// SSL/TLS specific fields
		SSLWarnDays:     target.SSLWarnDays,
		SSLCriticalDays: target.SSLCriticalDays,
		SSLCheck:       target.SSLCheck,
		SSLGetChain:    target.SSLGetChain,
	}

	return monitorTarget, nil
}
