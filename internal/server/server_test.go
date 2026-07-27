package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
				Roles:        []string{"user"},
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
	req.Header.Set("Authorization", "Bearer "+serverTestToken(t, "secret", "user-123", []string{"user"}))

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

func TestRouteRateLimitBlocksExcessRequests(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(upstream.Close)

	router, err := New(config.Config{
		Server: config.ServerConfig{Port: "8080"},
		Routes: []config.RouteConfig{
			{
				ID:       "limited-users",
				Path:     "/api/limited-users",
				Upstream: upstream.URL,
				Methods:  []string{http.MethodGet},
				RateLimit: config.RateLimitConfig{
					Requests: 1,
					Window:   "1m",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("create router: %v", err)
	}

	gateway := httptest.NewServer(router)
	t.Cleanup(gateway.Close)

	firstRes, err := http.Get(gateway.URL + "/api/limited-users")
	if err != nil {
		t.Fatalf("send first gateway request: %v", err)
	}
	defer firstRes.Body.Close()

	if firstRes.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, firstRes.StatusCode)
	}

	secondRes, err := http.Get(gateway.URL + "/api/limited-users")
	if err != nil {
		t.Fatalf("send second gateway request: %v", err)
	}
	defer secondRes.Body.Close()

	if secondRes.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, secondRes.StatusCode)
	}
}

func TestRouteTimeoutReturnsGatewayTimeout(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(upstream.Close)

	router, err := New(config.Config{
		Server: config.ServerConfig{Port: "8080"},
		Routes: []config.RouteConfig{
			{
				ID:       "slow-users",
				Path:     "/api/slow-users",
				Upstream: upstream.URL,
				Methods:  []string{http.MethodGet},
				Timeout:  "10ms",
			},
		},
	})
	if err != nil {
		t.Fatalf("create router: %v", err)
	}

	gateway := httptest.NewServer(router)
	t.Cleanup(gateway.Close)

	res, err := http.Get(gateway.URL + "/api/slow-users")
	if err != nil {
		t.Fatalf("send gateway request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("expected status %d, got %d", http.StatusGatewayTimeout, res.StatusCode)
	}
}

func TestRouteRetriesTransientUpstreamFailure(t *testing.T) {
	t.Parallel()

	attempts := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	t.Cleanup(upstream.Close)

	router, err := New(config.Config{
		Server: config.ServerConfig{Port: "8080"},
		Routes: []config.RouteConfig{
			{
				ID:       "flaky-users",
				Path:     "/api/flaky-users",
				Upstream: upstream.URL,
				Methods:  []string{http.MethodGet},
				Retries:  1,
			},
		},
	})
	if err != nil {
		t.Fatalf("create router: %v", err)
	}

	gateway := httptest.NewServer(router)
	t.Cleanup(gateway.Close)

	res, err := http.Get(gateway.URL + "/api/flaky-users")
	if err != nil {
		t.Fatalf("send gateway request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}

	if res.Header.Get("X-Retry-Attempts") != "1" {
		t.Fatalf("expected retry attempts header to be 1, got %q", res.Header.Get("X-Retry-Attempts"))
	}

	if attempts != 2 {
		t.Fatalf("expected 2 upstream attempts, got %d", attempts)
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

func TestMetricsEndpointExposesGatewayMetrics(t *testing.T) {
	t.Parallel()

	router, err := New(config.Config{
		Server: config.ServerConfig{Port: "8080"},
	})
	if err != nil {
		t.Fatalf("create router: %v", err)
	}

	gateway := httptest.NewServer(router)
	t.Cleanup(gateway.Close)

	healthRes, err := http.Get(gateway.URL + "/health")
	if err != nil {
		t.Fatalf("send health request: %v", err)
	}
	defer healthRes.Body.Close()

	metricsRes, err := http.Get(gateway.URL + "/metrics")
	if err != nil {
		t.Fatalf("send metrics request: %v", err)
	}
	defer metricsRes.Body.Close()

	body, err := io.ReadAll(metricsRes.Body)
	if err != nil {
		t.Fatalf("read metrics response: %v", err)
	}

	if !strings.Contains(string(body), "gateway_requests_total") {
		t.Fatalf("expected gateway metrics, got %s", string(body))
	}
}

func serverTestToken(t *testing.T, secret string, subject string, roles []string) string {
	t.Helper()

	claims := struct {
		Roles []string `json:"roles"`
		jwt.RegisteredClaims
	}{
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
