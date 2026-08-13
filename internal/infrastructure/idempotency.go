package infrastructure

import (
	"bytes"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// IdempotencyKeyHeader is the request header carrying the client idempotency key.
const IdempotencyKeyHeader = "Idempotency-Key"

// idempotencyState distinguishes in-flight claims from completed responses.
type idempotencyState string

const (
	stateInFlight idempotencyState = "in_flight"
	stateComplete idempotencyState = "complete"
)

// idempotencyEntry stores an idempotency claim plus its completed response.
// ExpiresAt bounds how long a key is retained; the janitor deletes expired rows.
type idempotencyEntry struct {
	Key       string           `gorm:"primaryKey;size:128"`
	State     idempotencyState `gorm:"size:16;not null"`
	Code      int              `gorm:"not null"`
	Body      string           `gorm:"type:text;not null"`
	ExpiresAt time.Time        `gorm:"index"`
}

// TableName keeps GORM from pluralizing this entity into a conflicting name.
func (idempotencyEntry) TableName() string { return "idempotency_keys" }

// bodyCapturer snapshots the response that gin writes so it can be replayed.
type bodyCapturer struct {
	gin.ResponseWriter
	body bytes.Buffer
	used bool
}

func (w *bodyCapturer) Write(p []byte) (int, error) {
	w.used = true
	w.body.Write(p)
	return w.ResponseWriter.Write(p)
}

func (w *bodyCapturer) WriteString(s string) (int, error) {
	w.used = true
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// startIdempotencyJanitor launches a single background sweep that deletes
// expired idempotency keys on a fixed interval.
func startIdempotencyJanitor(db *gorm.DB, interval time.Duration) {
	idemJanitorOnce.Do(func() {
		go func() {
			t := time.NewTicker(interval)
			defer t.Stop()
			for range t.C {
				// expires_at IS NULL covers rows created before the column existed.
				if err := db.Exec("DELETE FROM idempotency_keys WHERE expires_at IS NULL OR expires_at < NOW()").Error; err != nil {
					slog.Error("idempotency janitor sweep failed", "error", err)
				}
			}
		}()
	})
}

var idemJanitorOnce sync.Once

// IdempotencyMiddleware makes POST requests carrying an Idempotency-Key
// exactly-once. The first request claims the key (INSERT in_flight), runs the
// handler, and stores the response with an expiry (cfg.IdempotencyTTL). Concurrent
// or retried requests with the same key see the stored response; empty in-flight
// claims are returned as 202 so clients can poll. Expired rows are swept by a
// background janitor started once per process.
func IdempotencyMiddleware(db *gorm.DB, ttl, cleanupInterval time.Duration) gin.HandlerFunc {
	if cleanupInterval > 0 {
		startIdempotencyJanitor(db, cleanupInterval)
	}

	return func(c *gin.Context) {
		key := c.GetHeader(IdempotencyKeyHeader)
		if key == "" || c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		if len(key) > 128 {
			key = key[:128]
		}

		claim := &idempotencyEntry{
			Key:       key,
			State:     stateInFlight,
			ExpiresAt: time.Now().Add(ttl),
		}
		claimed := db.WithContext(c.Request.Context()).Create(claim).Error == nil

		if !claimed {
			// Key already exists: replay completed response or signal in-flight.
			var entry idempotencyEntry
			if err := db.WithContext(c.Request.Context()).Where("`key` = ?", key).First(&entry).Error; err == nil {
				switch entry.State {
				case stateComplete:
					c.Data(entry.Code, "application/json; charset=utf-8", []byte(entry.Body))
				default:
					c.Data(http.StatusNotImplemented, "application/json; charset=utf-8",
						[]byte(`{"code":1006,"message":"request in flight","data":null}`))
				}
				c.Abort()
				return
			}
			// Race: claim lost but entry vanished — run the handler this time.
			c.Next()
			return
		}

		captured := &bodyCapturer{ResponseWriter: c.Writer}
		c.Writer = captured
		c.Next()

		if captured.used && captured.body.Len() > 0 {
			_ = db.WithContext(c.Request.Context()).
				Model(&idempotencyEntry{}).
				Where("`key` = ?", claim.Key).
				UpdateColumns(map[string]any{
					"state":      stateComplete,
					"code":       captured.ResponseWriter.Status(),
					"body":       captured.body.String(),
					"expires_at": time.Now().Add(ttl),
				})
		}
	}
}
