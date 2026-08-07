// Package infrastructure provides shared foundational components for all
// business modules: configuration loading, database connectivity, unit-of-work
// transactions, structured logging, Prometheus metrics, and HTTP routing.
package infrastructure

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/natefinch/lumberjack"
)

type ctxKey string

const requestIDKey ctxKey = "request_id"

// InitLogger configures the global slog.Logger with JSON output to stdout and a rotated file.
func InitLogger(cfg *Config) {
	lumber := &lumberjack.Logger{
		Filename:   cfg.LogFile,
		MaxSize:    cfg.LogMaxSize,
		MaxAge:     cfg.LogMaxAge,
		MaxBackups: cfg.LogMaxBK,
		Compress:   true,
	}

	writer := io.MultiWriter(os.Stdout, lumber)

	base := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level:     parseLogLevel(cfg.LogLevel),
		AddSource: true,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if s, ok := a.Value.Any().(*slog.Source); ok {
					if i := strings.Index(s.File, "zhisuo-server/"); i >= 0 {
						s.File = s.File[i+len("zhisuo-server/"):]
					}
					if i := strings.Index(s.Function, "zhisuo-server/"); i >= 0 {
						s.Function = s.Function[i+len("zhisuo-server/"):]
					}
				}
			}
			return a
		},
	})

	logger := slog.New(&TraceHandler{next: base})
	slog.SetDefault(logger)

	go rotateDaily(lumber)
}

type traceCtxKey string

const traceIDKey traceCtxKey = "trace_id"
const spanIDKey traceCtxKey = "span_id"
const parentSpanIDKey traceCtxKey = "parent_span_id"

// NewSpanContext seeds a fresh log-only trace context for a request that has
// no incoming x-trace-id: the trace id serves as the root span id.
func NewSpanContext(parent context.Context, traceID, parentID string) context.Context {
	if traceID == "" {
		traceID = randomID()
	}
	spanID := randomID()
	parent = context.WithValue(parent, traceIDKey, traceID)
	parent = context.WithValue(parent, spanIDKey, spanID)
	parent = context.WithValue(parent, parentSpanIDKey, parentID)
	return parent
}

// AddChildSpan decorates a trace-carrying context with a new child span id.
// It returns the new span id along with the context.
func AddChildSpan(ctx context.Context) (context.Context, string) {
	spanID := randomID()
	parentID, _ := currentSpanID(ctx)
	out := context.WithValue(ctx, spanIDKey, spanID)
	out = context.WithValue(out, parentSpanIDKey, parentID)
	return out, spanID
}

func currentTraceID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(traceIDKey).(string)
	return id, ok && id != ""
}

func currentSpanID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(spanIDKey).(string)
	return id, ok && id != ""
}

func currentParentSpanID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(parentSpanIDKey).(string)
	return id, ok && id != ""
}

// TraceHandler is a slog.Handler wrapper that injects trace_id, span_id and
// parent_span_id from context into every log record.
type TraceHandler struct {
	next slog.Handler
}

// Enabled delegates to the wrapped handler's Enabled check.
func (h *TraceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle adds trace context attributes from ctx to the record before delegating.
func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := ctx.Value(requestIDKey).(string); ok && id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	if id, ok := currentTraceID(ctx); ok {
		r.AddAttrs(slog.String("trace_id", id))
	}
	if id, ok := currentSpanID(ctx); ok {
		r.AddAttrs(slog.String("span_id", id))
	}
	if id, ok := currentParentSpanID(ctx); ok && id != "" {
		r.AddAttrs(slog.String("parent_span_id", id))
	}

	return h.next.Handle(ctx, r)
}

// WithAttrs returns a new TraceHandler whose wrapped handler includes the given attributes.
func (h *TraceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TraceHandler{next: h.next.WithAttrs(attrs)}
}

// WithGroup returns a new TraceHandler whose wrapped handler starts the given group.
func (h *TraceHandler) WithGroup(name string) slog.Handler {
	return &TraceHandler{next: h.next.WithGroup(name)}
}

func randomID() string {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(time.Now().UnixNano())+uint64(rand.Uint32()))
	return hex.EncodeToString(b)
}

// SetReqID stores the incoming request ID in context for TraceHandler output.
func SetReqID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func rotateDaily(l *lumberjack.Logger) {
	for {
		now := time.Now()
		next := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
		time.Sleep(next.Sub(now))
		l.Rotate()
	}
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
