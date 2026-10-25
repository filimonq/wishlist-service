package metricsadapter_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	metrics "github.com/filimonq/wishlist-service/internal/adapter/out/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsEndpoint(t *testing.T) {
	t.Parallel()
	m := metrics.New()
	handler := m.Handler()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "go_goroutines ")
	assert.Contains(t, recorder.Body.String(), "process_cpu_seconds_total ")
	for _, name := range []string{
		"wishlist_user_registrations_total", "wishlist_user_logins_total",
		"wishlist_wishlists_created_total", "wishlist_items_reserved_total",
	} {
		assert.Contains(t, recorder.Body.String(), name+" 0\n")
	}

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	request.Header.Set("Accept", "application/openmetrics-text; version=1.0.0")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	assert.Contains(t, recorder.Header().Get("Content-Type"), "application/openmetrics-text")
	assert.Contains(t, recorder.Body.String(), "# EOF")

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}
