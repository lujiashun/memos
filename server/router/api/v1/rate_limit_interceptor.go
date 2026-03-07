package v1

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/usememos/memos/server/auth"
)

// RateLimitConfig defines the rate limit configuration
type RateLimitConfig struct {
	// UserLimit is the maximum number of requests per user per window
	UserLimit int
	// IPRateLimit is the maximum number of requests per IP per window
	IPLimit int
	// WindowSize is the time window for rate limiting
	WindowSize time.Duration
}

// DefaultRateLimitConfig provides default rate limit settings
var DefaultRateLimitConfig = RateLimitConfig{
	UserLimit:  60,  // 60 requests per minute per user
	IPLimit:    100, // 100 requests per minute per IP
	WindowSize: time.Minute,
}

// RateLimiter tracks request counts for rate limiting
type RateLimiter struct {
	config     RateLimitConfig
	userCounts map[int32][]time.Time // userID -> request times
	ipCounts   map[string][]time.Time // IP -> request times
	mutex      sync.RWMutex
}

// NewRateLimiter creates a new rate limiter with the given config
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	limiter := &RateLimiter{
		config:     config,
		userCounts: make(map[int32][]time.Time),
		ipCounts:   make(map[string][]time.Time),
	}

	// Start a goroutine to cleanup old entries
	go limiter.cleanupLoop()

	return limiter
}

// cleanupLoop periodically cleans up old entries
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.config.WindowSize)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes entries older than the window size
func (rl *RateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.config.WindowSize)

	// Cleanup user counts
	for userID, times := range rl.userCounts {
		var validTimes []time.Time
		for _, t := range times {
			if t.After(cutoff) {
				validTimes = append(validTimes, t)
			}
		}
		if len(validTimes) == 0 {
			delete(rl.userCounts, userID)
		} else {
			rl.userCounts[userID] = validTimes
		}
	}

	// Cleanup IP counts
	for ip, times := range rl.ipCounts {
		var validTimes []time.Time
		for _, t := range times {
			if t.After(cutoff) {
				validTimes = append(validTimes, t)
			}
		}
		if len(validTimes) == 0 {
			delete(rl.ipCounts, ip)
		} else {
			rl.ipCounts[ip] = validTimes
		}
	}
}

// Allow checks if the request is allowed based on rate limits
func (rl *RateLimiter) Allow(userID int32, ip string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.config.WindowSize)

	// Check user limit
	if userID > 0 {
		times := rl.userCounts[userID]
		var validTimes []time.Time
		for _, t := range times {
			if t.After(cutoff) {
				validTimes = append(validTimes, t)
			}
		}
		if len(validTimes) >= rl.config.UserLimit {
			return false
		}
		rl.userCounts[userID] = append(validTimes, now)
	}

	// Check IP limit
	times := rl.ipCounts[ip]
	var validTimes []time.Time
	for _, t := range times {
		if t.After(cutoff) {
			validTimes = append(validTimes, t)
		}
	}
	if len(validTimes) >= rl.config.IPLimit {
		return false
	}
	rl.ipCounts[ip] = append(validTimes, now)

	return true
}

// RateLimitInterceptor handles rate limiting for Connect handlers
type RateLimitInterceptor struct {
	limiter *RateLimiter
}

// NewRateLimitInterceptor creates a new rate limit interceptor
func NewRateLimitInterceptor(config RateLimitConfig) *RateLimitInterceptor {
	return &RateLimitInterceptor{
		limiter: NewRateLimiter(config),
	}
}

func (in *RateLimitInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		// Get user ID from context
		userID, _ := ctx.Value(auth.UserIDContextKey).(int32)

		// Get client IP
		ip := getClientIP(req.Header())

		// Check rate limit for CreateMemo requests
		if req.Spec().Procedure == "/memos.api.v1.MemoService/CreateMemo" {
			if !in.limiter.Allow(userID, ip) {
				return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("rate limit exceeded: too many memo creation requests"))
			}
		}

		return next(ctx, req)
	}
}

func (in *RateLimitInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (in *RateLimitInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// getClientIP extracts the client IP from headers
func getClientIP(header http.Header) string {
	// Try X-Forwarded-For first
	if xff := header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Try X-Real-IP
	if xri := header.Get("X-Real-Ip"); xri != "" {
		return xri
	}
	// Fallback to remote address (not available in Connect header, but included for completeness)
	return "unknown"
}
