package httpadapter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	metricsadapter "github.com/filimonq/wishlist-service/internal/adapter/out/metrics"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestMetrics(t *testing.T) {
	t.Parallel()
	metrics := metricsadapter.New()
	router := NewServer(nil, nil, nil, "test-secret", nil).Router(RequestMetrics(metrics.ObserveHTTPRequest))
	router.GET("/test/:id", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.GET("/panic", func(_ *gin.Context) { panic("test panic") })

	for _, path := range []string{"/test/one", "/test/two", "/unknown-one", "/unknown-two", "/panic", "/metrics", "/api/v1/wishlists"} {
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}
	recorder := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	body := recorder.Body.String()
	assert.Contains(t, body, `wishlist_http_requests_total{method="GET",route="/test/:id",status="204"} 2`)
	assert.Contains(t, body, `wishlist_http_requests_total{method="GET",route="unmatched",status="404"} 3`)
	assert.Contains(t, body, `wishlist_http_requests_total{method="GET",route="/panic",status="500"} 1`)
	assert.Contains(t, body, `wishlist_http_requests_total{method="GET",route="/api/v1/wishlists",status="401"} 1`)
	assert.Contains(t, body, `wishlist_http_request_duration_seconds_count{method="GET",route="/test/:id",status="204"} 2`)
	assert.NotContains(t, body, "/test/one")
	assert.NotContains(t, body, "/unknown-one")
}
