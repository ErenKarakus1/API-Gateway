package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	path := writeConfig(t, `
server:
  port: "8081"
routes:
  - id: users
    path: /api/users
    upstream: http://localhost:9001
    methods:
      - GET
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("expected config to load: %v", err)
	}

	if cfg.Server.Port != "8081" {
		t.Fatalf("expected port 8081, got %q", cfg.Server.Port)
	}

	if len(cfg.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(cfg.Routes))
	}

	route := cfg.Routes[0]
	if route.ID != "users" || route.Path != "/api/users" || route.Upstream != "http://localhost:9001" {
		t.Fatalf("loaded unexpected route: %+v", route)
	}
}

func TestLoadRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	path := writeConfig(t, `
server:
  port: ""
routes: []
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected invalid config to fail")
	}
}

func TestLoadRejectsRolesWithoutAuth(t *testing.T) {
	t.Parallel()

	path := writeConfig(t, `
server:
  port: "8080"
routes:
  - id: users
    path: /api/users
    upstream: http://localhost:9001
    roles:
      - admin
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected roles without auth to fail")
	}
}

func TestLoadRejectsInvalidTimeout(t *testing.T) {
	t.Parallel()

	path := writeConfig(t, `
server:
  port: "8080"
routes:
  - id: users
    path: /api/users
    upstream: http://localhost:9001
    timeout: sometimes
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected invalid timeout to fail")
	}
}

func TestLoadRejectsNegativeRetries(t *testing.T) {
	t.Parallel()

	path := writeConfig(t, `
server:
  port: "8080"
routes:
  - id: users
    path: /api/users
    upstream: http://localhost:9001
    retries: -1
`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected negative retries to fail")
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "gateway.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	return path
}
