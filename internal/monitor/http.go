package monitor

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"
	"time"

	"monitor/internal/logger"
	"go.uber.org/zap"
)

type HTTPChecker struct{}

func (c *HTTPChecker) Check(ctx context.Context, target *MonitorTarget) (*CheckResult, error) {
	start := time.Now()

	// 构建URL - 支持完整URL（包含路径）
	url := target.Address
	// 如果地址不包含协议前缀，添加协议
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		if target.Type == "http" {
			url = "http://" + url
		} else if target.Type == "https" {
			url = "https://" + url
		}
	}

	// 确定HTTP方法
	method := target.HTTPMethod
	if method == "" {
		method = "GET"
	}

	// 准备请求体
	var bodyReader *bytes.Reader
	if target.HTTPBody != "" {
		bodyReader = bytes.NewReader([]byte(target.HTTPBody))
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		logger.Error("Failed to create HTTP request",
			zap.String("target", target.Name),
			zap.Error(err),
		)
		return &CheckResult{
			Status:       "down",
			ResponseTime: time.Since(start).Milliseconds(),
			Message:      fmt.Sprintf("Failed to create request: %v", err),
		}, nil
	}

	// 设置自定义Headers
	if target.HTTPHeaders != nil {
		for key, value := range target.HTTPHeaders {
			req.Header.Set(key, value)
		}
	}

	// 添加基础请求头（如果没有设置）
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}
	if req.Header.Get("Accept-Encoding") == "" {
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	}
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	}
	if req.Header.Get("Connection") == "" {
		req.Header.Set("Connection", "keep-alive")
	}

	// 设置自定义Host
	if target.ResolvedHost != "" {
		req.Host = target.ResolvedHost
	}

	// Get global HTTP client with connection pooling
	client := GetHTTPClient()

	// Configure redirect policy
	if !target.FollowRedirects {
		// Create a new client that doesn't follow redirects
		client = &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // Don't follow redirects
			},
		}
	} else if target.MaxRedirects > 0 {
		// Custom redirect limit
		client = &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= target.MaxRedirects {
					return fmt.Errorf("stopped after %d redirects", target.MaxRedirects)
				}
				return nil
			},
		}
	}

	// Configure custom DNS resolver if needed
	if target.DNSServer != "" {
		// Create custom transport with DNS resolver
		transport := client.Transport.(*http.Transport)

		dialer := &net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}

		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			// 解析DNS服务器地址
			dnsServer := target.DNSServer
			if !strings.Contains(dnsServer, ":") {
				dnsServer = dnsServer + ":53"
			}

			// 使用自定义DNS解析器
			resolver := &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					return dialer.DialContext(ctx, "udp", dnsServer)
				},
			}

			// 从地址中提取主机名
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				host = addr
			}

			// 使用自定义DNS解析
			ips, err := resolver.LookupHost(ctx, host)
			if err != nil {
				logger.Error("DNS resolution failed",
					zap.String("host", host),
					zap.String("dns_server", target.DNSServer),
					zap.Error(err),
				)
				return nil, err
			}

			// 使用第一个IP地址
			if len(ips) > 0 {
				ip := ips[0]
				if _, port, err := net.SplitHostPort(addr); err == nil {
					addr = net.JoinHostPort(ip, port)
				} else {
					addr = net.JoinHostPort(ip, "443")
				}
			}

			return dialer.DialContext(ctx, network, addr)
		}

		// Create client with custom transport
		client = &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		}
	}

	// 执行请求
	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("HTTP request failed",
			zap.String("target", target.Name),
			zap.String("url", url),
			zap.Error(err),
		)
		responseTime := time.Since(start).Milliseconds()

		// 保存请求信息到结果
		result := &CheckResult{
			Status:       "down",
			ResponseTime: responseTime,
			Message:      fmt.Sprintf("Request failed: %v", err),
		}

		// 保存请求详情
		result.Request = RequestDetails{
			Method:      method,
			URL:         url,
			Headers:     cloneHeaders(req.Header),
			Body:        target.HTTPBody,
		}

		result.Error = &ErrorDetails{
			Type:    "network_error",
			Message: err.Error(),
		}

		return result, nil
	}
	defer resp.Body.Close()

	responseTime := time.Since(start).Milliseconds()

	logger.Debug("HTTP check completed",
		zap.String("target", target.Name),
		zap.Int("status_code", resp.StatusCode),
		zap.Int64("response_time", responseTime),
	)

	// Get actual resolved IP from DNS lookup
	resolvedIP := ""
	host := target.Address
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		parsedURL, err := neturl.Parse(host)
		if err == nil {
			host = parsedURL.Hostname()
		}
	}

	// Remove port if present
	if strings.Contains(host, ":") {
		h, _, err := net.SplitHostPort(host)
		if err == nil {
			host = h
		}
	}

	// Do DNS lookup to get the real IP address
	ips, err := net.LookupIP(host)
	if err == nil && len(ips) > 0 {
		// Prefer IPv4 addresses
		for _, ip := range ips {
			if ip.To4() != nil {
				resolvedIP = ip.String()
				logger.Debug("DNS lookup resolved to IPv4",
					zap.String("host", host),
					zap.String("ip", resolvedIP),
				)
				break
			}
		}
		// Fall back to IPv6 if no IPv4
		if resolvedIP == "" && len(ips) > 0 {
			resolvedIP = ips[0].String()
			logger.Debug("DNS lookup resolved to IPv6",
				zap.String("host", host),
				zap.String("ip", resolvedIP),
			)
		}
	} else {
		logger.Warn("DNS lookup failed",
			zap.String("host", host),
			zap.Error(err),
		)
	}

	// Fallback: use hostname if DNS lookup fails
	if resolvedIP == "" {
		resolvedIP = resp.Request.URL.Hostname()
		logger.Debug("Using hostname as fallback for resolved IP",
			zap.String("hostname", resolvedIP),
		)
	}

	// 读取响应体
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Warn("Failed to read response body",
			zap.String("target", target.Name),
			zap.Error(err),
		)
		responseBody = []byte(fmt.Sprintf("Failed to read response body: %v", err))
	}

	// 解压gzip响应体（用于提取Title）
	var htmlBody []byte
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		reader, err := gzip.NewReader(bytes.NewReader(responseBody))
		if err == nil {
			htmlBody, err = io.ReadAll(reader)
			if err != nil {
				htmlBody = responseBody // 如果解压失败，使用原始数据
			}
		} else {
			htmlBody = responseBody
		}
	} else {
		htmlBody = responseBody
	}

	// 限制响应体大小（避免存储过大的响应）
	maxBodySize := 1024 * 100 // 100KB
	if len(responseBody) > maxBodySize {
		responseBody = append(responseBody[:maxBodySize], []byte("... (truncated)")...)
	}

	// Determine status based on expected status codes
	status := determineStatus(resp.StatusCode, target.ExpectedStatusCodes)

	// 构建结果
	result := &CheckResult{
		Status:       status,
		ResponseTime: responseTime,
		Message:      fmt.Sprintf("HTTP %d %s", resp.StatusCode, resp.Status),
	}

	// 保存请求详情
	result.Request = RequestDetails{
		Method:      method,
		URL:         url,
		Headers:     cloneHeaders(req.Header),
		Body:        target.HTTPBody,
	}

	// 保存响应详情，包含解析的IP
	result.Response = ResponseDetails{
		StatusCode:    resp.StatusCode,
		Headers:       cloneHeaders(resp.Header),
		Body:          string(responseBody),
		ContentLength: resp.ContentLength,
	}

	// Add resolved IP to headers for storage
	if result.Response.Headers == nil {
		result.Response.Headers = make(map[string]string)
	}
	result.Response.Headers["resolved_ip"] = resolvedIP

	// Extract title from HTML response if content-type is HTML
	if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		if title := extractTitle(htmlBody); title != "" {
			result.Response.Headers["title"] = title
		}
	}

	return result, nil
}

// extractTitle extracts the title from HTML content
func extractTitle(body []byte) string {
	// Use regex to find <title> tag content
	re := regexp.MustCompile(`(?i)<title[^>]*>(.*?)</title>`)
	matches := re.FindSubmatch(body)
	if len(matches) > 1 {
		title := strings.TrimSpace(string(matches[1]))
		// Decode HTML entities (basic)
		title = strings.ReplaceAll(title, "&nbsp;", " ")
		title = strings.ReplaceAll(title, "&lt;", "<")
		title = strings.ReplaceAll(title, "&gt;", ">")
		title = strings.ReplaceAll(title, "&amp;", "&")
		title = strings.ReplaceAll(title, "&quot;", "\"")
		return title
	}
	return ""
}

// determineStatus determines if the status code is expected
func determineStatus(statusCode int, expectedCodes []int) string {
	// Default: 2xx is up, 3xx/4xx/5xx is down
	if len(expectedCodes) == 0 {
		if statusCode >= 200 && statusCode < 300 {
			return "up"
		}
		return "down"
	}

	// Check against expected codes
	for _, code := range expectedCodes {
		if statusCode == code {
			return "up"
		}
	}

	return "down"
}

// cloneHeaders 克隆 HTTP headers
func cloneHeaders(headers http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range headers {
		if len(values) > 0 {
			result[key] = strings.Join(values, ", ")
		}
	}
	return result
}