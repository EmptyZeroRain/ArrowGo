package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiterConfig holds rate limiter configuration
type RateLimiterConfig struct {
	RequestsPerSecond float64       // Rate limit (requests per second)
	BurstSize         int           // Burst size
	CleanupInterval   time.Duration // How often to cleanup unused limiters
}

// IPRateLimiter manages rate limiters per IP
type IPRateLimiter struct {
	limiters map[string]*limiterEntry
	mu       sync.RWMutex
	config   RateLimiterConfig
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewIPRateLimiter creates a new IP-based rate limiter
func NewIPRateLimiter(config RateLimiterConfig) *IPRateLimiter {
	rl := &IPRateLimiter{
		limiters: make(map[string]*limiterEntry),
		config:   config,
	}

	// Start cleanup goroutine
	go rl.cleanupStaleLimiters()

	return rl
}

// getLimiter returns the rate limiter for the given IP
func (rl *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[ip]
	if !exists {
		limiter := rate.NewLimiter(rate.Limit(rl.config.RequestsPerSecond), rl.config.BurstSize)
		rl.limiters[ip] = &limiterEntry{
			limiter:  limiter,
			lastSeen: time.Now(),
		}
		return limiter
	}

	entry.lastSeen = time.Now()
	return entry.limiter
}

// cleanupStaleLimiters removes limiters that haven't been used recently
func (rl *IPRateLimiter) cleanupStaleLimiters() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, entry := range rl.limiters {
			if now.Sub(entry.lastSeen) > rl.config.CleanupInterval {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware returns a Gin middleware for rate limiting
func (rl *IPRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": fmt.Sprintf("Rate limit exceeded. Please try again later."),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Default rate limiter (100 requests per second with burst of 200)
var DefaultRateLimiter = NewIPRateLimiter(RateLimiterConfig{
	RequestsPerSecond: 100,
	BurstSize:         200,
	CleanupInterval:   5 * time.Minute,
})

// RateLimit is a middleware function using the default rate limiter
func RateLimit() gin.HandlerFunc {
	return DefaultRateLimiter.Middleware()
}