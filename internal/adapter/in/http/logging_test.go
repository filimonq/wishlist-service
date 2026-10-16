package httpadapter

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRequestLogging(t *testing.T) {
	previousLogger := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	tests := []struct {
		name   string
		status int
		level  string
		panics bool
	}{
		{name: "success", status: http.StatusOK, level: "INFO"},
		{name: "no content", status: http.StatusNoContent, level: "INFO"},
		{name: "redirect", status: http.StatusFound, level: "INFO"},
		{name: "bad request", status: http.StatusBadRequest, level: "WARN"},
		{name: "unauthorized", status: http.StatusUnauthorized, level: "WARN"},
		{name: "not found", status: http.StatusNotFound, level: "WARN"},
		{name: "conflict", status: http.StatusConflict, level: "WARN"},
		{name: "rate limited", status: http.StatusTooManyRequests, level: "WARN"},
		{name: "server error", status: http.StatusInternalServerError, level: "ERROR"},
		{name: "unavailable", status: http.StatusServiceUnavailable, level: "ERROR"},
		{name: "panic", status: http.StatusInternalServerError, level: "ERROR", panics: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
			router := NewServer(nil, nil, nil, "").Router()
			router.GET("/test/:token", func(c *gin.Context) {
				if tt.panics {
					panic("test panic")
				}
				c.Status(tt.status)
			})

			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/test/private-token?secret=private-query", nil)
			request.Header.Set("Authorization", "Bearer private-authorization")
			router.ServeHTTP(response, request)

			requestID := response.Header().Get("X-Request-ID")
			_, err := uuid.Parse(requestID)
			require.NoError(t, err)
			wantRecords := 1
			if tt.panics {
				wantRecords = 2
			}
			require.Equal(t, tt.status, response.Code)
			lines := strings.Split(strings.TrimSpace(output.String()), "\n")
			require.Len(t, lines, wantRecords)
			for _, line := range lines {
				var record map[string]any
				require.NoError(t, json.Unmarshal([]byte(line), &record))
				require.Equal(t, requestID, record["request_id"])
				require.Equal(t, "/test/:token", record["route"])
				require.Equal(t, tt.level, record["level"])
				if record["msg"] == "http request" {
					duration, ok := record["duration_ms"].(float64)
					require.True(t, ok, "duration_ms must be a number")
					require.GreaterOrEqual(t, duration, float64(0))
					require.EqualValues(t, tt.status, record["status"])
				} else {
					require.Equal(t, "test panic", record["panic"])
					require.NotEmpty(t, record["stack"])
				}
			}
			for _, secret := range []string{"private-token", "private-query", "private-authorization"} {
				require.NotContains(t, output.String(), secret)
			}
		})
	}
}
