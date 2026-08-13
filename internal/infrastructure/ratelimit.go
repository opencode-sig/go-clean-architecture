package infrastructure

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kun/zhisuo-server/internal/port"
	"golang.org/x/time/rate"
)

const (
	// limiterPruneEvery is how often idle per-IP limiters are swept.
	limiterPruneEvery = 5 * time.Minute
	// limiterIdleTTL is how long a limiter is kept after its last request.
	limiterIdleTTL = 15 * time.Minute
)

// ipEntry pairs a token-bucket limiter with the last time it was used, so idle
// limiters can be evicted instead of leaking forever.
type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter gives each client IP a token bucket limiter and periodically
// prunes limiters that have been idle for limiterIdleTTL.
type IPRateLimiter struct {
	mu         sync.Mutex
	entries    map[string]*ipEntry
	rps        rate.Limit
	burst      int
	trustProxy bool
	idleTTL    time.Duration
}

// NewIPRateLimiter creates an IP keyed limiter with the given rate and burst.
// When trustProxy is true, the client IP is resolved from the X-Forwarded-For /
// X-Real-IP headers — only enable it behind a proxy that overwrites those
// headers, otherwise clients can spoof their identity.
func NewIPRateLimiter(rps float64, burst int, trustProxy bool) *IPRateLimiter {
	if rps <= 0 {
		rps = 1
	}
	if burst <= 0 {
		burst = 1
	}
	l := &IPRateLimiter{
		entries:    make(map[string]*ipEntry),
		rps:        rate.Limit(rps),
		burst:      burst,
		trustProxy: trustProxy,
		idleTTL:    limiterIdleTTL,
	}
	go l.pruneLoop()
	return l
}

// get returns the per-IP limiter (creating it on first sight) and stamps the
// last-seen time.
func (l *IPRateLimiter) get(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	e, ok := l.entries[ip]
	if !ok {
		e = &ipEntry{limiter: rate.NewLimiter(l.rps, l.burst)}
		l.entries[ip] = e
	}
	e.lastSeen = time.Now()
	return e.limiter
}

// pruneLoop sweeps idle limiters on a fixed interval until the process exits.
func (l *IPRateLimiter) pruneLoop() {
	t := time.NewTicker(limiterPruneEvery)
	defer t.Stop()
	for range t.C {
		l.prune()
	}
}

// prune removes limiters that have not been used within l.idleTTL.
func (l *IPRateLimiter) prune() {
	cutoff := time.Now().Add(-l.idleTTL)
	l.mu.Lock()
	defer l.mu.Unlock()
	for ip, e := range l.entries {
		if e.lastSeen.Before(cutoff) {
			delete(l.entries, ip)
		}
	}
}

// Middleware builds a Gin handler that rejects requests exceeding the limit
// with a 429-style business response (uniform envelope: code 1005).
func (l *IPRateLimiter) Middleware(skip func(*gin.Context) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if skip != nil && skip(c) {
			c.Next()
			return
		}

		ip := l.clientIP(c)
		if !l.get(ip).Allow() {
			port.Error(c, port.CodeTooManyRequests, "rate limit exceeded")
			c.Abort()
			return
		}
		c.Next()
	}
}

// clientIP resolves the caller's IP, honoring proxy headers only when
// trustProxy is enabled; otherwise it falls back to the socket remote address.
func (l *IPRateLimiter) clientIP(c *gin.Context) string {
	if l.trustProxy {
		if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
			if first := strings.TrimSpace(strings.Split(xff, ",")[0]); first != "" {
				return first
			}
		}
		if rip := c.GetHeader("X-Real-IP"); rip != "" {
			return rip
		}
	}
	return remoteIP(c.Request.RemoteAddr)
}

// remoteIP extracts the host from a socket address.
func remoteIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
