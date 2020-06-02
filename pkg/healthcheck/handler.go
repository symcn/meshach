package healthcheck

import (
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/mesh-operator/pkg/router"
)

// Check is a health/readiness check.
type Check func() error

// Handler is an endpoints with additional methods that register health and
// readiness checks. It handles handle "/live" and "/ready" HTTP
// endpoints.
type Handler interface {
	Routes() []*router.Route
	AddLivenessCheck(name string, check Check)
	AddReadinessCheck(name string, check Check)
	LiveEndpoint(ctx *gin.Context)
	ReadyEndpoint(ctx *gin.Context)
	RemoveLivenessCheck(name string)
	RemoveReadinessCheck(name string)
}

// basicHandler is a basic Handler implementation.
type basicHandler struct {
	checksMutex     sync.RWMutex
	livenessChecks  map[string]Check
	readinessChecks map[string]Check
}

var (
	handler Handler
	mu      sync.Mutex
)

// GetHealthHandler ...
func GetHealthHandler() Handler {
	if handler != nil {
		return handler
	}

	mu.Lock()
	defer mu.Unlock()

	if handler != nil {
		return handler
	}

	h := &basicHandler{
		livenessChecks:  make(map[string]Check),
		readinessChecks: make(map[string]Check),
	}
	handler = h
	return handler
}

func (s *basicHandler) Routes() []*router.Route {
	var routes []*router.Route

	ctlRoutes := []*router.Route{
		&router.Route{
			Method:  "GET",
			Path:    "/live",
			Handler: s.LiveEndpoint,
		},
		&router.Route{
			Method:  "GET",
			Path:    "/ready",
			Handler: s.ReadyEndpoint,
		},
	}

	routes = append(routes, ctlRoutes...)
	return routes
}

func (s *basicHandler) LiveEndpoint(ctx *gin.Context) {
	s.handle(ctx, s.livenessChecks)
}

func (s *basicHandler) ReadyEndpoint(ctx *gin.Context) {
	s.handle(ctx, s.readinessChecks, s.livenessChecks)
}

func (s *basicHandler) AddLivenessCheck(name string, check Check) {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()
	s.livenessChecks[name] = check
}

func (s *basicHandler) AddReadinessCheck(name string, check Check) {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()
	s.readinessChecks[name] = check
}

func (s *basicHandler) RemoveLivenessCheck(name string) {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()

	for n := range s.livenessChecks {
		if strings.HasPrefix(n, name) {
			delete(s.livenessChecks, n)
		}
	}
}

func (s *basicHandler) RemoveReadinessCheck(name string) {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()

	for n := range s.readinessChecks {
		if strings.HasPrefix(n, name) {
			delete(s.readinessChecks, n)
		}
	}
}

func (s *basicHandler) collectChecks(checks map[string]Check, resultsOut map[string]string, statusOut *int) {
	s.checksMutex.RLock()
	defer s.checksMutex.RUnlock()
	for name, check := range checks {
		if err := check(); err != nil {
			*statusOut = http.StatusServiceUnavailable
			resultsOut[name] = err.Error()
		} else {
			resultsOut[name] = "OK"
		}
	}
}

func (s *basicHandler) handle(ctx *gin.Context, checks ...map[string]Check) {
	checkResults := make(map[string]string)
	status := http.StatusOK
	for _, checks := range checks {
		s.collectChecks(checks, checkResults, &status)
	}

	// unless ?full=true, return an empty body. Kubernetes only cares about the
	// HTTP status code, so we won't waste bytes on the full body.
	fullStr := ctx.DefaultQuery("full", "false")
	if fullStr == "false" {
		ctx.JSON(status, "OK")
		return
	}

	ctx.IndentedJSON(status, checkResults)
}
