// Package middleware — per-IP token-bucket rate limiter.
package middleware

import (
	"net/http"
	"sync"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

// RateLimitConfig configures the rate limiter.
type RateLimitConfig struct {
	// RequestsPerMinute is the number of requests allowed per IP per minute.
	RequestsPerMinute int
	// Burst is the maximum number of tokens the bucket can hold.
	// Allows short traffic spikes above the sustained rate.
	// Default: same as RequestsPerMinute.
	Burst int
	// KeyFunc derives the limiting key from the request.
	// Default: client IP address.
	KeyFunc func(ctx Context) string
	// OnLimitReached is called when a request is rejected.
	// Default: respond with 429 Too Many Requests.
	OnLimitReached func(ctx Context)
}

// RateLimit returns a middleware that limits requests to n per minute per IP.
//
//	gofastapi.Use(middleware.RateLimit(1000)) // 1 000 req/min per IP
func RateLimit(requestsPerMinute int, configs ...RateLimitConfig) MiddlewareFunc {
	cfg := RateLimitConfig{
		RequestsPerMinute: requestsPerMinute,
		Burst:             requestsPerMinute,
		KeyFunc:           func(ctx Context) string { return ctx.ClientIP() },
		OnLimitReached: func(ctx Context) {
			ctx.JSON(http.StatusTooManyRequests, map[string]any{
				"success": false,
				"error": map[string]string{
					"code":    "TOO_MANY_REQUESTS",
					"message": "rate limit exceeded — please slow down",
				},
			})
			ctx.Abort()
		},
	}
	if len(configs) > 0 {
		c := configs[0]
		if c.Burst > 0 {
			cfg.Burst = c.Burst
		}
		if c.KeyFunc != nil {
			cfg.KeyFunc = c.KeyFunc
		}
		if c.OnLimitReached != nil {
			cfg.OnLimitReached = c.OnLimitReached
		}
	}

	// Store limiters per key in an expiring cache (entries evicted after 10 min idle).
	store := gocache.New(10*time.Minute, 5*time.Minute)
	var mu sync.Mutex
	eventsPerSecond := rate.Limit(float64(cfg.RequestsPerMinute) / 60.0)

	getLimiter := func(key string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()
		if v, ok := store.Get(key); ok {
			store.Set(key, v, gocache.DefaultExpiration) // refresh TTL
			return v.(*rate.Limiter)
		}
		l := rate.NewLimiter(eventsPerSecond, cfg.Burst)
		store.Set(key, l, gocache.DefaultExpiration)
		return l
	}

	return func(ctx Context) {
		key := cfg.KeyFunc(ctx)
		limiter := getLimiter(key)
		if !limiter.Allow() {
			cfg.OnLimitReached(ctx)
			return
		}
		ctx.Next()
	}
}

// RateLimitWithConfig is the full-config variant of RateLimit.
func RateLimitWithConfig(cfg RateLimitConfig) MiddlewareFunc {
	return RateLimit(cfg.RequestsPerMinute, cfg)
}
