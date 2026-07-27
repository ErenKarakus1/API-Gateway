package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/ErenKarakus1/API-Gateway/internal/config"
	"github.com/ErenKarakus1/API-Gateway/internal/middleware"
	"github.com/ErenKarakus1/API-Gateway/internal/response"
	"github.com/gin-gonic/gin"
)

type Route struct {
	ID      string
	Path    string
	Proxy   *httputil.ReverseProxy
	Target  *url.URL
	Retries int
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
		response.WriteError(w, r.Header.Get(middleware.RequestIDHeader), http.StatusGatewayTimeout, "upstream_timeout", "upstream timeout")
	}

	originalDirector := reverseProxy.Director
	reverseProxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		req.URL.Path = joinPaths(target.Path, trimRoutePrefix(req.URL.Path, cfg.Path))
	}

	return Route{
		ID:      cfg.ID,
		Path:    cfg.Path,
		Proxy:   reverseProxy,
		Target:  target,
		Retries: cfg.Retries,
	}, nil
}

func Handler(route Route) gin.HandlerFunc {
	return func(c *gin.Context) {
		if route.Retries > 0 && canRetry(c.Request.Method) {
			if retry(c, route) {
				return
			}
		}

		route.Proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func retry(c *gin.Context, route Route) bool {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Error(c, http.StatusBadGateway, "request_body_read_failed", "read request body")
		return true
	}
	c.Request.Body.Close()

	attempts := route.Retries + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		req := c.Request.Clone(c.Request.Context())
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.URL.Scheme = route.Target.Scheme
		req.URL.Host = route.Target.Host
		req.URL.Path = joinPaths(route.Target.Path, trimRoutePrefix(c.Request.URL.Path, route.Path))
		req.Host = route.Target.Host
		req.RequestURI = ""

		transport := route.Proxy.Transport
		if transport == nil {
			transport = http.DefaultTransport
		}
		res, err := transport.RoundTrip(req)
		if err != nil {
			if attempt == attempts {
				response.Error(c, http.StatusGatewayTimeout, "upstream_timeout", "upstream timeout")
				return true
			}
			continue
		}

		defer res.Body.Close()
		if retryableStatus(res.StatusCode) && attempt < attempts {
			_, _ = io.Copy(io.Discard, res.Body)
			continue
		}

		copyHeader(c.Writer.Header(), res.Header)
		c.Writer.Header().Set("X-Retry-Attempts", fmt.Sprintf("%d", attempt-1))
		c.Status(res.StatusCode)
		_, _ = io.Copy(c.Writer, res.Body)
		return true
	}

	return false
}

func canRetry(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func retryableStatus(status int) bool {
	return status == http.StatusBadGateway ||
		status == http.StatusServiceUnavailable ||
		status == http.StatusGatewayTimeout
}

func copyHeader(dst http.Header, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
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
