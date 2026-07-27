package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ErenKarakus1/API-Gateway/internal/config"
)

func TestConfiguredRouteProxiesToUpstream(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/42" {
			t.Fatalf("expected upstream path /42, got %q", r.URL.Path)
		}

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
}
