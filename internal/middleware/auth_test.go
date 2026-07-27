package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestJWTAuthRejectsMissingToken(t *testing.T) {
	t.Parallel()

	router := authTestRouter("secret")
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestJWTAuthAcceptsValidToken(t *testing.T) {
	t.Parallel()

	router := authTestRouter("secret")
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+signedToken(t, "secret", "user-123"))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	if rec.Header().Get("X-User-ID") != "" {
		t.Fatalf("user id should be propagated through the request, not response headers")
	}
}

func authTestRouter(secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JWTAuth(secret))
	router.GET("/secure", func(c *gin.Context) {
		if c.GetString(UserIDKey) != "user-123" {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Status(http.StatusNoContent)
	})
	return router
}

func signedToken(t *testing.T, secret string, subject string) string {
	t.Helper()

	claims := jwt.RegisteredClaims{
		Subject:   subject,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return token
}
