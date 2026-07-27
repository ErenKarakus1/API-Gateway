package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ErenKarakus1/API-Gateway/internal/config"
	"github.com/ErenKarakus1/API-Gateway/internal/metrics"
	"github.com/ErenKarakus1/API-Gateway/internal/middleware"
	"github.com/ErenKarakus1/API-Gateway/internal/proxy"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func New(cfg config.Config) (*gin.Engine, error) {
	return NewWithLogger(cfg, zap.NewNop())
}

func NewWithLogger(cfg config.Config, logger *zap.Logger) (*gin.Engine, error) {
	router := gin.New()
	rateLimiter := middleware.NewRateLimiter()
	appMetrics := metrics.New()

	router.Use(gin.Recovery(), middleware.RequestID(), appMetrics.Middleware(), middleware.Logger(logger))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	})
	router.GET("/metrics", appMetrics.Handler())

	for _, routeCfg := range cfg.Routes {
		route, err := proxy.NewRoute(routeCfg)
		if err != nil {
			return nil, fmt.Errorf("create route %q: %w", routeCfg.ID, err)
		}

		methods := routeCfg.Methods
		if len(methods) == 0 {
			methods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
		}

		handlers := []gin.HandlerFunc{}
		if routeCfg.AuthRequired {
			handlers = append(handlers, middleware.JWTAuth(cfg.Auth.JWTSecret))
		}
		if len(routeCfg.Roles) > 0 {
			handlers = append(handlers, middleware.RequireRoles(routeCfg.Roles))
		}
		if routeCfg.RateLimit.Requests > 0 {
			window, err := time.ParseDuration(routeCfg.RateLimit.Window)
			if err != nil {
				return nil, fmt.Errorf("parse rate limit window for route %q: %w", routeCfg.ID, err)
			}
			handlers = append(handlers, rateLimiter.Limit(routeCfg.ID, routeCfg.RateLimit.Requests, window))
		}
		handlers = append(handlers, proxy.Handler(route))

		for _, method := range methods {
			router.Handle(method, routeCfg.Path, handlers...)
			router.Handle(method, routeCfg.Path+"/*path", handlers...)
		}
	}

	return router, nil
}
