package monitor

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"
)

// Global HTTP client with connection pooling
var (
	globalHTTPClient *http.Client
	httpClientOnce   sync.Once
)

// GetHTTPClient returns a singleton HTTP client optimized for high concurrency
func GetHTTPClient() *http.Client {
	httpClientOnce.Do(func() {
		transport := &http.Transport{
			// Connection pool settings for high concurrency
			MaxIdleConns:        200,              // Maximum number of idle connections
			MaxIdleConnsPerHost: 100,              // Maximum idle connections per host
			MaxConnsPerHost:     0,                // 0 means no limit (use MaxIdleConnsPerHost)
			IdleConnTimeout:     90 * time.Second, // How long to keep idle connections

			// TLS settings
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
				MinVersion:         tls.VersionTLS12,
			},

			// Timeouts
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second, // Connection timeout
				KeepAlive: 30 * time.Second, // Keep alive timeout
			}).DialContext,

			// Response header timeout
			ResponseHeaderTimeout: 30 * time.Second,

			// Expect continue timeout (for 100-Continue)
			ExpectContinueTimeout: 1 * time.Second,

			// Force HTTP/2
			ForceAttemptHTTP2: true,
		}

		globalHTTPClient = &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		}
	})

	return globalHTTPClient
}