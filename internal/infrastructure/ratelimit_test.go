package infrastructure

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newTestRequest(remoteAddr string, headers map[string]string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Request.RemoteAddr = remoteAddr
	for k, v := range headers {
		c.Request.Header.Set(k, v)
	}
	return c
}

func TestIPRateLimiterClientIP(t *testing.T) {
	tests := []struct {
		name       string
		trustProxy bool
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{
			name:       "uses remote addr when not trusting proxy",
			trustProxy: false,
			remoteAddr: "203.0.113.9:1234",
			headers:    map[string]string{"X-Forwarded-For": "198.51.100.1, 10.0.0.2"},
			want:       "203.0.113.9",
		},
		{
			name:       "first X-Forwarded-For entry wins when trusting proxy",
			trustProxy: true,
			remoteAddr: "10.0.0.2:1234",
			headers:    map[string]string{"X-Forwarded-For": "198.51.100.1, 10.0.0.2"},
			want:       "198.51.100.1",
		},
		{
			name:       "X-Real-IP fallback when trusting proxy",
			trustProxy: true,
			remoteAddr: "10.0.0.2:1234",
			headers:    map[string]string{"X-Real-IP": "198.51.100.7"},
			want:       "198.51.100.7",
		},
		{
			name:       "ignores spoofed headers when not trusting proxy",
			trustProxy: false,
			remoteAddr: "203.0.113.9:1234",
			headers:    map[string]string{"X-Real-IP": "198.51.100.7"},
			want:       "203.0.113.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewIPRateLimiter(10, 20, tt.trustProxy)
			c := newTestRequest(tt.remoteAddr, tt.headers)
			if got := l.clientIP(c); got != tt.want {
				t.Errorf("clientIP = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIPRateLimiterPruneIdleEntries(t *testing.T) {
	l := NewIPRateLimiter(10, 20, false)
	l.idleTTL = time.Hour

	l.get("198.51.100.1")
	l.get("198.51.100.2")

	// Simulate the second IP going idle long ago.
	l.mu.Lock()
	l.entries["198.51.100.2"].lastSeen = time.Now().Add(-2 * time.Hour)
	l.mu.Unlock()

	l.prune()

	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.entries["198.51.100.1"]; !ok {
		t.Error("active entry was pruned")
	}
	if _, ok := l.entries["198.51.100.2"]; ok {
		t.Error("idle entry was not pruned")
	}
}

func TestIPRateLimiterRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	l := NewIPRateLimiter(1, 2, false)

	var rejected int
	for i := 0; i < 5; i++ {
		c := newTestRequest("203.0.113.5:1234", nil)
		l.Middleware(func(*gin.Context) bool { return false })(c)
		if c.IsAborted() {
			rejected++
		}
	}
	if rejected != 3 {
		t.Errorf("rejected = %d, want 3 (burst 2 then refill 1/s)", rejected)
	}
}
