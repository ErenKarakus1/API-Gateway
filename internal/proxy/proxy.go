package proxy

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/ErenKarakus1/API-Gateway/internal/config"
	"github.com/gin-gonic/gin"
)

type Route struct {
	ID     string
	Path   string
	Proxy  *httputil.ReverseProxy
	Target *url.URL
}

func NewRoute(cfg config.RouteConfig) (Route, error) {
	target, err := url.Parse(cfg.Upstream)
	if err != nil {
		return Route{}, err
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	if cfg.Timeout != "" {
		timeout, err := time.ParseDuration(cfg.Timeout)
		if err != nil {
			return Route{}, err
		}
		reverseProxy.Transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: timeout,
			}).DialContext,
			ResponseHeaderTimeout: timeout,
		}
	}
	reverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = w.Write([]byte(`{"error":"upstream timeout"}`))
	}

	originalDirector := reverseProxy.Director
	reverseProxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		req.URL.Path = joinPaths(target.Path, trimRoutePrefix(req.URL.Path, cfg.Path))
	}

	return Route{
		ID:     cfg.ID,
		Path:   cfg.Path,
		Proxy:  reverseProxy,
		Target: target,
	}, nil
}

func Handler(route Route) gin.HandlerFunc {
	return func(c *gin.Context) {
		route.Proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func trimRoutePrefix(path string, prefix string) string {
	trimmed := strings.TrimPrefix(path, prefix)
	if trimmed == "" {
		return "/"
	}
	return trimmed
}

func joinPaths(base string, requestPath string) string {
	if base == "" || base == "/" {
		return requestPath
	}

	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(requestPath, "/")
}
