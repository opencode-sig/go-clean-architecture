// Package infrastructure provides shared foundational components for all
// business modules: configuration loading, database connectivity, unit-of-work
// transactions, structured logging, Prometheus metrics, and HTTP routing.
package infrastructure

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	articleHandler "github.com/kun/zhisuo-server/internal/article/adapter/handler"
	commentHandler "github.com/kun/zhisuo-server/internal/comment/adapter/handler"
	"github.com/kun/zhisuo-server/internal/port"
	userHandler "github.com/kun/zhisuo-server/internal/user/adapter/handler"
	"github.com/kun/zhisuo-server/internal/web"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// Handlers aggregates all domain HTTP handlers for injection into the router.
type Handlers struct {
	User    *userHandler.UserHandler
	Article *articleHandler.ArticleHandler
	Comment *commentHandler.CommentHandler
}

type ginLogWriter struct{}

func (ginLogWriter) Write(p []byte) (int, error) {
	slog.Info(string(p[:len(p)-1]))
	return len(p), nil
}

// NewRouter builds the Gin engine with recovery, request ID, structured logging,
// metrics middleware, rate limiting, health endpoints, API routes, and static serving.
func NewRouter(h Handlers, db *gorm.DB, rateLimiter *IPRateLimiter) *gin.Engine {
	gin.DefaultWriter = ginLogWriter{}

	r := gin.New()
	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		slog.ErrorContext(c.Request.Context(), "panic recovered", "error", recovered)
		port.Error(c, port.CodeInternal, "internal server error")
	}))
	r.Use(requestID())
	r.Use(slogLogger())
	r.Use(MetricsMiddleware())
	r.Use(rateLimiter.Middleware(nil))
	r.Use(IdempotencyMiddleware(db))

	r.GET("/healthz", NewHealthHandler(db))
	r.GET("/readyz", NewHealthHandler(db))

	r.POST("/api/v1/users/list", h.User.List)
	r.POST("/api/v1/users/create", h.User.Create)
	r.POST("/api/v1/users/get", h.User.GetByID)
	r.POST("/api/v1/users/update", h.User.Update)
	r.POST("/api/v1/users/delete", h.User.Delete)

	r.POST("/api/v1/articles/list", h.Article.List)
	r.POST("/api/v1/articles/create", h.Article.Create)
	r.POST("/api/v1/articles/get", h.Article.GetByID)
	r.POST("/api/v1/articles/update", h.Article.Update)
	r.POST("/api/v1/articles/delete", h.Article.Delete)
	r.POST("/api/v1/articles/by-user", h.Article.ListByUser)

	r.POST("/api/v1/comments/list", h.Comment.ListByArticle)
	r.POST("/api/v1/comments/create", h.Comment.Create)
	r.POST("/api/v1/comments/delete", h.Comment.Delete)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/metrics", metricsHandler())

	staticFS, _ := fs.Sub(web.FS, "static")
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			port.Error(c, port.CodeNotFound, "route not found")
			return
		}
		c.FileFromFS(c.Request.URL.Path, http.FS(staticFS))
	})

	return r
}

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}

		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = id
		}

		reqCtx := NewSpanContext(c.Request.Context(), traceID, "")
		reqCtx = SetReqID(reqCtx, id)
		c.Request = c.Request.WithContext(reqCtx)
		c.Writer.Header().Set("X-Request-ID", id)
		c.Next()
	}
}

func slogLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" || strings.HasPrefix(c.Request.URL.Path, "/swagger/") {
			c.Next()
			return
		}

		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		slog.InfoContext(c.Request.Context(), "request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"bytes", c.Writer.Size(),
			"duration", time.Since(start).String(),
		)
	}
}
