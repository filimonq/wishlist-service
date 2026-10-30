package httpadapter

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/filimonq/wishlist-service/internal/adapter/in/http/demo"
	"github.com/stretchr/testify/assert"
)

func TestDemoRouteScopeAndMetrics(t *testing.T) {
	t.Parallel()
	const delay = 10 * time.Millisecond
	server := NewServer(nil, nil, nil, "test-secret", nil, demo.Middleware(demo.Config{Delay: delay, ErrorRate: 1}))
	var observedStatus int
	var observedSeconds float64
	router := server.Router(RequestMetrics(func(_, route string, status int, seconds float64) {
		if route == "/api/v1/public/:token" {
			observedStatus, observedSeconds = status, seconds
		}
	}))
	for _, tt := range []struct {
		method string
		path   string
		code   int
	}{
		{method: http.MethodGet, path: "/api/v1/public/test", code: http.StatusServiceUnavailable},
		{method: http.MethodGet, path: "/livez", code: http.StatusOK},
		{method: http.MethodGet, path: "/api/v1/wishlists", code: http.StatusUnauthorized},
		{method: http.MethodPost, path: "/api/v1/public/invalid/items/invalid/reserve", code: http.StatusBadRequest},
		{method: http.MethodGet, path: "/unknown", code: http.StatusNotFound},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(tt.method, tt.path, nil))
		assert.Equal(t, tt.code, recorder.Code, tt.path)
	}
	assert.Equal(t, http.StatusServiceUnavailable, observedStatus)
	assert.GreaterOrEqual(t, observedSeconds, delay.Seconds())
}
