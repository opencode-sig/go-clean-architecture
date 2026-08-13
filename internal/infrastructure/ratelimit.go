package infrastructure

import (
	"net"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/kun/zhisuo-server/internal/port"
	"golang.org/x/time/rate"
)

// IPRateLimiter gives each client IP a token bucket limiter.
type IPRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

// NewIPRateLimiter creates an IP keyed limiter with the given rate and burst.
// It uses a client IP that should be set by a trusted proxy header downstream.
func NewIPRateLimiter(rps float64, burst int) *IPRateLimiter {
	if rps <= 0 {
		rps = 1
	}
	if burst <= 0 {
		burst = 1
	}
	return &IPRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

// execute returns the per-IP limiter (creating it on first sight).
func (l *IPRateLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	lim, ok := l.limiters[ip]
	if !ok {
		lim = rate.NewLimiter(l.rps, l.burst)
		l.limiters[ip] = lim
	}
	return lim
}

// Middleware builds a Gin handler that rejects requests exceeding the limit
// with a 429-style business response (uniform envelope: code 1005).
func (l *IPRateLimiter) Middleware(skip func(*gin.Context) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if skip != nil && skip(c) {
			c.Next()
			return
		}

		ip := clientIP(c.Request.RemoteAddr)
		if !l.get(ip).Allow() {
			port.Error(c, port.CodeTooManyRequests, "rate limit exceeded")
			c.Abort()
			return
		}
		c.Next()
	}
}

func clientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
