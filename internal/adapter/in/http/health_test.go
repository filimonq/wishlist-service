package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHealthChecks(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		ping       func(context.Context) error
		wantStatus int
		wantBody   string
	}{
		{
			name: "liveness does not depend on database",
			path: "/livez",
			ping: func(context.Context) error {
				t.Fatal("liveness must not call the database")
				return nil
			},
			wantStatus: http.StatusOK,
			wantBody:   `{"status":"ok"}`,
		},
		{
			name:       "database available",
			path:       "/readyz",
			ping:       func(context.Context) error { return nil },
			wantStatus: http.StatusOK,
			wantBody:   `{"status":"ready"}`,
		},
		{
			name: "database unavailable without exposing error",
			path: "/readyz",
			ping: func(context.Context) error {
				return errors.New("connection failed with private-password")
			},
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   `{"status":"not ready"}`,
		},
		{
			name:       "database check missing",
			path:       "/readyz",
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   `{"status":"not ready"}`,
		},
		{
			name: "database check respects timeout",
			path: "/readyz",
			ping: func(ctx context.Context) error {
				_, hasDeadline := ctx.Deadline()
				require.True(t, hasDeadline)
				<-ctx.Done()
				return ctx.Err()
			},
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   `{"status":"not ready"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewServer(nil, nil, nil, "", tt.ping).Router()
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tt.path, nil))
			require.Equal(t, tt.wantStatus, response.Code)
			require.JSONEq(t, tt.wantBody, response.Body.String())
		})
	}
}
