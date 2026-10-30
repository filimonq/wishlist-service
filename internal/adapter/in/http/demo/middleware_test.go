package demo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		cfg  Config
		code int
	}{
		{name: "disabled", code: http.StatusOK},
		{name: "delayed", cfg: Config{Delay: 10 * time.Millisecond}, code: http.StatusOK},
		{name: "unavailable", cfg: Config{ErrorRate: 1}, code: http.StatusServiceUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			router := gin.New()
			called := false
			router.GET("/test", Middleware(tt.cfg), func(c *gin.Context) {
				called = true
				c.Status(http.StatusOK)
			})
			recorder := httptest.NewRecorder()
			start := time.Now()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))
			assert.Equal(t, tt.code, recorder.Code)
			assert.Equal(t, tt.code == http.StatusOK, called)
			assert.GreaterOrEqual(t, time.Since(start), tt.cfg.Delay)
		})
	}
}

func TestCancelledDelaySkipsHandler(t *testing.T) {
	t.Parallel()
	router := gin.New()
	called := false
	router.GET("/test", Middleware(Config{Delay: time.Hour}), func(_ *gin.Context) { called = true })
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/test", nil)
	router.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusRequestTimeout, recorder.Code)
	assert.False(t, called)
}

func TestFromEnv(t *testing.T) {
	for _, tt := range []struct {
		name      string
		delay     string
		rate      string
		wantError string
	}{
		{name: "defaults"},
		{name: "configured", delay: "1s", rate: "0.3"},
		{name: "invalid delay", delay: "abc", wantError: "DEMO_DELAY"},
		{name: "negative delay", delay: "-1s", wantError: "DEMO_DELAY"},
		{name: "invalid rate", rate: "abc", wantError: "DEMO_ERROR_RATE"},
		{name: "negative rate", rate: "-0.1", wantError: "DEMO_ERROR_RATE"},
		{name: "rate above one", rate: "1.1", wantError: "DEMO_ERROR_RATE"},
		{name: "NaN", rate: "NaN", wantError: "DEMO_ERROR_RATE"},
		{name: "infinity", rate: "+Inf", wantError: "DEMO_ERROR_RATE"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DEMO_DELAY", tt.delay)
			t.Setenv("DEMO_ERROR_RATE", tt.rate)
			cfg, err := FromEnv()
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			if tt.name == "configured" {
				assert.Equal(t, time.Second, cfg.Delay)
				assert.InDelta(t, 0.3, cfg.ErrorRate, 0.000001)
			} else {
				assert.Zero(t, cfg.Delay)
				assert.Zero(t, cfg.ErrorRate)
			}
		})
	}
}
