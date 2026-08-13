// Package infrastructure provides shared foundational components for all
// business modules: configuration loading, database connectivity, unit-of-work
// transactions, structured logging, Prometheus metrics, and HTTP routing.
package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kun/zhisuo-server/internal/port"
)

const userIDKey ctxKey = "user_id"

// JWTAuth signs and verifies stateless HMAC bearer tokens and injects the
// authenticated user id into the request context. It is the authentication
// skeleton: wire it onto the /api/v1 route group and read the user id from the
// context in protected handlers. Token issuance must be gated by a real
// credential check (login) in production.
type JWTAuth struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

// NewJWTAuth builds a JWT manager. It fails when no signing secret is set —
// auth enabled without a secret is a configuration error caught at startup.
func NewJWTAuth(cfg *Config) (*JWTAuth, error) {
	if cfg.AuthSecret == "" {
		return nil, errors.New("jwt secret must not be empty when auth is enabled")
	}
	return &JWTAuth{secret: []byte(cfg.AuthSecret), issuer: cfg.AuthIssuer, ttl: cfg.AuthExpires}, nil
}

// Sign mints a token for the given user id and returns it with its expiry.
func (a *JWTAuth) Sign(userID int64) (string, time.Time, error) {
	expires := time.Now().Add(a.ttl)
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		Issuer:    a.issuer,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(expires),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(a.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expires, nil
}

// Parse validates a token's signature and expiry and returns the subject user id.
func (a *JWTAuth) Parse(tokenString string) (int64, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return a.secret, nil
		},
		jwt.WithIssuer(a.issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return 0, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid || claims.Subject == "" {
		return 0, errors.New("invalid token claims")
	}
	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid token subject: %w", err)
	}
	return userID, nil
}

// AuthMiddleware rejects requests without a valid Bearer token and injects the
// authenticated user id into the request context.
func (a *JWTAuth) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || strings.TrimSpace(token) == "" {
			port.Error(c, port.CodeUnauthorized, "missing bearer token")
			c.Abort()
			return
		}

		userID, err := a.Parse(token)
		if err != nil {
			slog.DebugContext(c.Request.Context(), "invalid bearer token", "error", err)
			port.Error(c, port.CodeUnauthorized, "invalid token")
			c.Abort()
			return
		}

		c.Request = c.Request.WithContext(SetUserID(c.Request.Context(), userID))
		c.Next()
	}
}

// SetUserID stores the authenticated user id in the request context.
func SetUserID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserIDFromContext returns the authenticated user id, if any.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}

// TokenRequest is the JSON body for the dev-only token endpoint.
type TokenRequest struct {
	UserID int64 `json:"user_id" binding:"required" example:"1" minimum:"1"`
}

// NewAuthTokenHandler godoc
// @Summary      Issue a dev JWT token
// @Description  Mints a token for a user id without any credential check. Development only — register via auth.dev_token_endpoint.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body TokenRequest true "User ID"
// @Success      200  {object}  port.Response{data=object}
// @Router       /auth/token [post]
func NewAuthTokenHandler(auth *JWTAuth) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req TokenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			port.Error(c, port.CodeBadRequest, err.Error())
			return
		}

		token, expires, err := auth.Sign(req.UserID)
		if err != nil {
			slog.ErrorContext(c.Request.Context(), "sign token failed", "error", err)
			port.ErrorInternal(c, "internal error")
			return
		}

		port.Success(c, gin.H{"token": token, "expires_at": expires})
	}
}
