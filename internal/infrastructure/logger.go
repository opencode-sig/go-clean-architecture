// Package infrastructure provides shared foundational components for all
// business modules: configuration loading, database connectivity, unit-of-work
// transactions, structured logging, Prometheus metrics, and HTTP routing.
package infrastructure

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/natefinch/lumberjack"
)

type ctxKey string

const reqIDKey ctxKey = "request_id"

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

// TraceHandler is a slog.Handler wrapper that injects trace_id from context into every log record.
type TraceHandler struct {
	next slog.Handler
}

// Enabled delegates to the wrapped handler's Enabled check.
func (h *TraceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle adds the trace_id attribute from context to the record before delegating.
func (h *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := ctx.Value(reqIDKey).(string); ok && id != "" {
		r.AddAttrs(slog.String("trace_id", id))
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

// SetReqID stores the given request ID in context for later extraction by TraceHandler.
func SetReqID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, reqIDKey, id)
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
