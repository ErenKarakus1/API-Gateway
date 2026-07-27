package middleware

import (
	"encoding/json"
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

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.Error.Code != "missing_bearer_token" {
		t.Fatalf("expected missing_bearer_token, got %q", body.Error.Code)
	}
}

func TestJWTAuthAcceptsValidToken(t *testing.T) {
	t.Parallel()

	router := authTestRouter("secret")
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+signedToken(t, "secret", "user-123", []string{"user"}))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	if rec.Header().Get("X-User-ID") != "" {
		t.Fatalf("user id should be propagated through the request, not response headers")
	}
}

func TestRequireRolesAllowsMatchingRole(t *testing.T) {
	t.Parallel()

	router := roleTestRouter("secret", []string{"admin"})
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+signedToken(t, "secret", "user-123", []string{"admin"}))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestRequireRolesRejectsMissingRole(t *testing.T) {
	t.Parallel()

	router := roleTestRouter("secret", []string{"admin"})
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+signedToken(t, "secret", "user-123", []string{"user"}))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
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

func roleTestRouter(secret string, roles []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JWTAuth(secret), RequireRoles(roles))
	router.GET("/secure", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	return router
}

func signedToken(t *testing.T, secret string, subject string, roles []string) string {
	t.Helper()

	claims := Claims{
		Roles: roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return token
}
