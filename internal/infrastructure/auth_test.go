package infrastructure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kun/zhisuo-server/internal/port"
)

func newTestJWTAuth(t *testing.T) *JWTAuth {
	t.Helper()
	auth, err := NewJWTAuth(&Config{
		AuthEnabled: true,
		AuthSecret:  "test-secret-that-is-long-enough",
		AuthIssuer:  "test-issuer",
		AuthExpires: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewJWTAuth: %v", err)
	}
	return auth
}

func TestNewJWTAuthMissingSecret(t *testing.T) {
	if _, err := NewJWTAuth(&Config{AuthEnabled: true, AuthSecret: "", AuthIssuer: "i", AuthExpires: time.Hour}); err == nil {
		t.Fatal("expected error when auth enabled without secret")
	}
}

func TestJWTAuthSignParseRoundTrip(t *testing.T) {
	auth := newTestJWTAuth(t)

	token, expires, err := auth.Sign(42)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if !expires.After(time.Now()) {
		t.Fatalf("expected future expiry, got %v", expires)
	}

	userID, err := auth.Parse(token)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if userID != 42 {
		t.Fatalf("Parse userID = %d, want 42", userID)
	}
}

func TestJWTAuthParseRejectsTamperedToken(t *testing.T) {
	auth := newTestJWTAuth(t)

	token, _, err := auth.Sign(1)
	if err != nil {
		t.Fatal(err)
	}

	tampered := token[:len(token)-4] + "xxxx"
	if _, err := auth.Parse(tampered); err == nil {
		t.Fatal("expected error parsing tampered token")
	}
}

func TestJWTAuthParseRejectsExpiredToken(t *testing.T) {
	auth := newTestJWTAuth(t)
	auth.ttl = -time.Minute

	token, _, err := auth.Sign(1)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := auth.Parse(token); err == nil {
		t.Fatal("expected error parsing expired token")
	}
}

func TestJWTAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auth := newTestJWTAuth(t)
	token, _, err := auth.Sign(7)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid token injects user id", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		auth.AuthMiddleware()(c)

		if c.IsAborted() {
			t.Fatalf("unexpected abort, status %d", w.Code)
		}
		if got, ok := UserIDFromContext(c.Request.Context()); !ok || got != 7 {
			t.Fatalf("UserIDFromContext = %d, %v; want 7, true", got, ok)
		}
	})

	t.Run("missing token is rejected", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", nil)

		auth.AuthMiddleware()(c)

		if !c.IsAborted() {
			t.Fatal("expected request to be aborted")
		}
		var resp port.Response
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.Code != port.CodeUnauthorized {
			t.Fatalf("code = %d, want %d", resp.Code, port.CodeUnauthorized)
		}
	})

	t.Run("invalid token is rejected", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
		c.Request.Header.Set("Authorization", "Bearer not-a-token")

		auth.AuthMiddleware()(c)

		if !c.IsAborted() {
			t.Fatal("expected request to be aborted")
		}
		var resp port.Response
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.Code != port.CodeUnauthorized {
			t.Fatalf("code = %d, want %d", resp.Code, port.CodeUnauthorized)
		}
	})
}
