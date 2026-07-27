package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/ErenKarakus1/API-Gateway/internal/response"
	"github.com/gin-gonic/gin"
)

type CircuitBreaker struct {
	mu     sync.Mutex
	routes map[string]circuitState
	now    func() time.Time
}

type circuitState struct {
	failures int
	openedAt time.Time
}

func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		routes: make(map[string]circuitState),
		now:    time.Now,
	}
}

func (breaker *CircuitBreaker) Protect(routeID string, failureThreshold int, resetTimeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if failureThreshold <= 0 || resetTimeout <= 0 {
			c.Next()
			return
		}

		if !breaker.allow(routeID, resetTimeout) {
			response.Error(c, http.StatusServiceUnavailable, "circuit_open", "upstream circuit is open")
			return
		}

		c.Next()

		if c.Writer.Status() >= http.StatusInternalServerError {
			breaker.recordFailure(routeID, failureThreshold)
			return
		}

		breaker.recordSuccess(routeID)
	}
}

func (breaker *CircuitBreaker) allow(routeID string, resetTimeout time.Duration) bool {
	breaker.mu.Lock()
	defer breaker.mu.Unlock()

	state := breaker.routes[routeID]
	if state.openedAt.IsZero() {
		return true
	}

	if breaker.now().Sub(state.openedAt) >= resetTimeout {
		delete(breaker.routes, routeID)
		return true
	}

	return false
}

func (breaker *CircuitBreaker) recordFailure(routeID string, failureThreshold int) {
	breaker.mu.Lock()
	defer breaker.mu.Unlock()

	state := breaker.routes[routeID]
	state.failures++
	if state.failures >= failureThreshold && state.openedAt.IsZero() {
		state.openedAt = breaker.now()
	}
	breaker.routes[routeID] = state
}

func (breaker *CircuitBreaker) recordSuccess(routeID string) {
	breaker.mu.Lock()
	defer breaker.mu.Unlock()

	delete(breaker.routes, routeID)
}
