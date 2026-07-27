package server

import (
	"fmt"
	"net/http"

	"github.com/ErenKarakus1/API-Gateway/internal/config"
	"github.com/ErenKarakus1/API-Gateway/internal/middleware"
	"github.com/ErenKarakus1/API-Gateway/internal/proxy"
	"github.com/gin-gonic/gin"
)

func New(cfg config.Config) (*gin.Engine, error) {
	router := gin.New()

	router.Use(gin.Logger(), gin.Recovery(), middleware.RequestID())

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

	for _, routeCfg := range cfg.Routes {
		route, err := proxy.NewRoute(routeCfg)
		if err != nil {
			return nil, fmt.Errorf("create route %q: %w", routeCfg.ID, err)
		}

		methods := routeCfg.Methods
		if len(methods) == 0 {
			methods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
		}

		for _, method := range methods {
			router.Handle(method, routeCfg.Path, proxy.Handler(route))
			router.Handle(method, routeCfg.Path+"/*path", proxy.Handler(route))
		}
	}

	return router, nil
}
