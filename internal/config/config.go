package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultPath = "config/gateway.yaml"

type Config struct {
	Server ServerConfig  `yaml:"server"`
	Auth   AuthConfig    `yaml:"auth"`
	Routes []RouteConfig `yaml:"routes"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type RouteConfig struct {
	ID             string               `yaml:"id"`
	Path           string               `yaml:"path"`
	Upstream       string               `yaml:"upstream"`
	Methods        []string             `yaml:"methods"`
	AuthRequired   bool                 `yaml:"auth_required"`
	Roles          []string             `yaml:"roles"`
	RateLimit      RateLimitConfig      `yaml:"rate_limit"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
	Timeout        string               `yaml:"timeout"`
	Retries        int                  `yaml:"retries"`
}

type AuthConfig struct {
	JWTSecret string `yaml:"jwt_secret"`
}

type RateLimitConfig struct {
	Requests int    `yaml:"requests"`
	Window   string `yaml:"window"`
}

type CircuitBreakerConfig struct {
	FailureThreshold int    `yaml:"failure_threshold"`
	ResetTimeout     string `yaml:"reset_timeout"`
}

func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (cfg Config) Validate() error {
	if cfg.Server.Port == "" {
		return errors.New("server.port is required")
	}

	seenRoutes := make(map[string]struct{}, len(cfg.Routes))
	for i, route := range cfg.Routes {
		if route.ID == "" {
			return fmt.Errorf("routes[%d].id is required", i)
		}
		if route.Path == "" {
			return fmt.Errorf("routes[%d].path is required", i)
		}
		if route.Upstream == "" {
			return fmt.Errorf("routes[%d].upstream is required", i)
		}
		if route.AuthRequired && cfg.Auth.JWTSecret == "" {
			return fmt.Errorf("auth.jwt_secret is required when route %q requires auth", route.ID)
		}
		if len(route.Roles) > 0 && !route.AuthRequired {
			return fmt.Errorf("route %q must require auth when roles are configured", route.ID)
		}
		if route.RateLimit.Requests < 0 {
			return fmt.Errorf("routes[%d].rate_limit.requests must be greater than zero", i)
		}
		if route.RateLimit.Requests > 0 && route.RateLimit.Window == "" {
			return fmt.Errorf("routes[%d].rate_limit.window is required", i)
		}
		if route.CircuitBreaker.FailureThreshold < 0 {
			return fmt.Errorf("routes[%d].circuit_breaker.failure_threshold must be greater than zero", i)
		}
		if route.CircuitBreaker.FailureThreshold > 0 {
			if route.CircuitBreaker.ResetTimeout == "" {
				return fmt.Errorf("routes[%d].circuit_breaker.reset_timeout is required", i)
			}
			if _, err := time.ParseDuration(route.CircuitBreaker.ResetTimeout); err != nil {
				return fmt.Errorf("routes[%d].circuit_breaker.reset_timeout is invalid: %w", i, err)
			}
		}
		if route.Timeout != "" {
			if _, err := time.ParseDuration(route.Timeout); err != nil {
				return fmt.Errorf("routes[%d].timeout is invalid: %w", i, err)
			}
		}
		if route.Retries < 0 {
			return fmt.Errorf("routes[%d].retries must be greater than or equal to zero", i)
		}
		if _, ok := seenRoutes[route.ID]; ok {
			return fmt.Errorf("route id %q is duplicated", route.ID)
		}
		seenRoutes[route.ID] = struct{}{}
	}

	return nil
}
