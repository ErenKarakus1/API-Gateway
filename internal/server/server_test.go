package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ErenKarakus1/API-Gateway/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

func TestConfiguredRouteProxiesToUpstream(t *testing.T) {
	t.Parallel()

	var upstreamRequestID string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/42" {
			t.Fatalf("expected upstream path /42, got %q", r.URL.Path)
		}
		upstreamRequestID = r.Header.Get("X-Request-ID")

		w.Header().Set("X-Upstream", "users")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"id":"42"}`))
	}))
	t.Cleanup(upstream.Close)

	router, err := New(config.Config{
		Server: config.ServerConfig{Port: "8080"},
		Routes: []config.RouteConfig{
			{
				ID:       "users",
				Path:     "/api/users",
				Upstream: upstream.URL,
				Methods:  []string{http.MethodGet},
			},
		},
	})
	if err != nil {
		t.Fatalf("create router: %v", err)
	}

	gateway := httptest.NewServer(router)
	t.Cleanup(gateway.Close)

	res, err := http.Get(gateway.URL + "/api/users/42")
	if err != nil {
		t.Fatalf("send gateway request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, res.StatusCode)
	}

	if res.Header.Get("X-Upstream") != "users" {
		t.Fatalf("expected upstream header to be proxied")
	}

	if res.Header.Get("X-Request-ID") == "" {
		t.Fatalf("expected gateway response to include request id")
	}

	if upstreamRequestID == "" {
		t.Fatalf("expected request id to be propagated upstream")
	}
}

func TestProtectedRouteRequiresValidJWT(t *testing.T) {
	t.Parallel()

	var upstreamCalled bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		if r.Header.Get("X-User-ID") != "user-123" {
			t.Fatalf("expected user id to be propagated upstream")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(upstream.Close)

	router, err := New(config.Config{
		Server: config.ServerConfig{Port: "8080"},
		Auth:   config.AuthConfig{JWTSecret: "secret"},
		Routes: []config.RouteConfig{
			{
				ID:           "secure-users",
				Path:         "/api/secure-users",
				Upstream:     upstream.URL,
				Methods:      []string{http.MethodGet},
				AuthRequired: true,
			},
		},
	})
	if err != nil {
		t.Fatalf("create router: %v", err)
	}

	gateway := httptest.NewServer(router)
	t.Cleanup(gateway.Close)

	unauthorizedRes, err := http.Get(gateway.URL + "/api/secure-users")
	if err != nil {
		t.Fatalf("send unauthorized gateway request: %v", err)
	}
	defer unauthorizedRes.Body.Close()

	if unauthorizedRes.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, unauthorizedRes.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, gateway.URL+"/api/secure-users", nil)
	if err != nil {
		t.Fatalf("build authorized request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+serverTestToken(t, "secret", "user-123"))

	authorizedRes, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send authorized gateway request: %v", err)
	}
	defer authorizedRes.Body.Close()

	if authorizedRes.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, authorizedRes.StatusCode)
	}

	if !upstreamCalled {
		t.Fatalf("expected authorized request to reach upstream")
	}
}

func TestRequestIDMiddlewarePreservesIncomingID(t *testing.T) {
	t.Parallel()

	router, err := New(config.Config{
		Server: config.ServerConfig{Port: "8080"},
	})
	if err != nil {
		t.Fatalf("create router: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "portfolio-request-1")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") != "portfolio-request-1" {
		t.Fatalf("expected incoming request id to be preserved")
	}
}

func serverTestToken(t *testing.T, secret string, subject string) string {
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
