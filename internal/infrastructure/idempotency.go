package infrastructure

import (
	"bytes"
	"net/http"

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
type idempotencyEntry struct {
	Key   string           `gorm:"primaryKey;size:128"`
	State idempotencyState `gorm:"size:16;not null"`
	Code  int              `gorm:"not null"`
	Body  string           `gorm:"type:text;not null"`
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

// IdempotencyMiddleware makes POST requests carrying an Idempotency-Key
// exactly-once. The first request claims the key (INSERT in_flight), runs the
// handler, and stores the response. Concurrent or retried requests with the same
// key see the stored response. Empty in-flight claims are returned as 202 so
// clients can poll; a completed entry replays its stored response verbatim.
func IdempotencyMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader(IdempotencyKeyHeader)
		if key == "" || c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		if len(key) > 128 {
			key = key[:128]
		}

		claim := &idempotencyEntry{Key: key, State: stateInFlight}
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
					"state": stateComplete,
					"code":  captured.ResponseWriter.Status(),
					"body":  captured.body.String(),
				})
		}
	}
}
