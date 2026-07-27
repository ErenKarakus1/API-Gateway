package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ErenKarakus1/API-Gateway/internal/config"
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
