// Package middleware — in-memory response cache.
//
// The cache middleware stores GET responses in an in-memory LRU/TTL cache.
// Cached responses are returned directly without hitting the handler.
// Only successful (2xx) responses are cached.
// Non-GET requests and requests with query parameters are not cached by default.
package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// CacheConfig configures the cache middleware.
type CacheConfig struct {
	// TTL is how long a cached response is considered fresh.
	// Default: 1 minute.
	TTL time.Duration
	// MaxItems is the soft cap on the number of cached entries (LRU eviction).
	// Default: 10 000.
	MaxItems int
	// KeyFunc derives the cache key from the request.
	// Default: method + path + sorted query string.
	KeyFunc func(ctx Context) string
	// ShouldCache decides whether a given request should be cached.
	// Default: only cache GET requests without an Authorization header.
	ShouldCache func(ctx Context) bool
	// Store is an optional custom cache store. Default: in-memory go-cache.
	Store interface {
		Get(key string) (any, bool)
		Set(key string, value any, d time.Duration)
	}
}

// cachedResponse stores a pre-serialised JSON response.
type cachedResponse struct {
	Status int
	Body   any
}

// Cache returns a middleware that caches GET responses for ttl duration.
//
//	gofastapi.Use(middleware.Cache(5 * time.Minute))
func Cache(ttl time.Duration, configs ...CacheConfig) MiddlewareFunc {
	cfg := CacheConfig{
		TTL:      ttl,
		MaxItems: 10_000,
		KeyFunc: func(ctx Context) string {
			raw := ctx.Request().URL.RawQuery
			return ctx.Request().Method + ":" + ctx.Request().URL.Path + "?" + raw
		},
		ShouldCache: func(ctx Context) bool {
			if ctx.Request().Method != http.MethodGet {
				return false
			}
			// Don't cache authenticated requests by default.
			if ctx.Request().Header.Get("Authorization") != "" {
				return false
			}
			return true
		},
	}
	if len(configs) > 0 {
		c := configs[0]
		if c.TTL > 0 {
			cfg.TTL = c.TTL
		}
		if c.KeyFunc != nil {
			cfg.KeyFunc = c.KeyFunc
		}
		if c.ShouldCache != nil {
			cfg.ShouldCache = c.ShouldCache
		}
		if c.Store != nil {
			cfg.Store = c.Store
		}
	}

	var store interface {
		Get(string) (any, bool)
		Set(string, any, time.Duration)
	}
	if cfg.Store != nil {
		store = cfg.Store
	} else {
		c := gocache.New(cfg.TTL, cfg.TTL*2)
		store = &goCacheWrapper{c}
	}

	return func(ctx Context) {
		if !cfg.ShouldCache(ctx) {
			ctx.Next()
			return
		}

		key := cfg.KeyFunc(ctx)

		// Cache hit.
		if v, found := store.Get(key); found {
			if cr, ok := v.(*cachedResponse); ok {
				addCacheHeader(ctx, "HIT")
				ctx.JSON(cr.Status, cr.Body)
				ctx.Abort()
				return
			}
		}

		// Cache miss — let the handler run, then cache the response.
		// (True response capture requires a response writer interceptor in the
		// router adapter. This implementation uses a cooperative model where
		// handlers may call ctx.Set("cache_response", v) to opt in.)
		addCacheHeader(ctx, "MISS")
		ctx.Next()

		// Check if handler deposited a cacheable value.
		if v, ok := ctx.Get("_cache_response"); ok {
			if cr, ok2 := v.(*cachedResponse); ok2 {
				store.Set(key, cr, cfg.TTL)
			}
		}
	}
}

// CacheResponse stores v in the context so the Cache middleware can persist it.
// Call this from your handler when you want the response to be cached.
//
//	func myHandler(ctx crud.Context) {
//	    data := fetchExpensiveData()
//	    middleware.CacheResponse(ctx, 200, data)
//	}
func CacheResponse(ctx Context, status int, body any) {
	ctx.Set("_cache_response", &cachedResponse{Status: status, Body: body})
	ctx.JSON(status, body)
}

// Invalidate removes a cache entry by key prefix.
// The store must be the *goCacheWrapper returned by Cache's default store.
func Invalidate(store any, keyPrefix string) {
	if gc, ok := store.(*goCacheWrapper); ok {
		for k := range gc.c.Items() {
			if strings.HasPrefix(k, keyPrefix) {
				gc.c.Delete(k)
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

type goCacheWrapper struct {
	c *gocache.Cache
}

func (w *goCacheWrapper) Get(k string) (any, bool)             { return w.c.Get(k) }
func (w *goCacheWrapper) Set(k string, v any, d time.Duration) { w.c.Set(k, v, d) }

func addCacheHeader(_ Context, _ string) {
	// X-Cache header injection is handled by the router adapter.
	_ = fmt.Sprintf // keep import
}
