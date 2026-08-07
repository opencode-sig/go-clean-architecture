package infrastructure

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NewHealthHandler returns liveness (/healthz) and readiness (/readyz) handlers.
// Liveness always 200. Readiness pings the DB so load balancers only route
// traffic when the service can actually serve requests.
func NewHealthHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.URL.Path {
		case "/healthz":
			c.JSON(http.StatusOK, gin.H{"status": "up"})
		case "/readyz":
			ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
			defer cancel()
			sqlDB, err := db.DB()
			if err != nil || sqlDB.PingContext(ctx) != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "up"})
		}
	}
}
