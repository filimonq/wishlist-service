package prober

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

type contextReader struct{ ctx context.Context }

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func (r contextReader) Read(_ []byte) (int, error) {
	<-r.ctx.Done()
	return 0, r.ctx.Err()
}

func TestCheck(t *testing.T) {
	t.Parallel()
	const id = "00000000-0000-4000-8000-000000000001"
	tests := []struct {
		name    string
		body    string
		status  int
		err     error
		outcome string
	}{
		{name: "matching wishlist", status: 200, body: `{"id":"` + id + `","title":"Birthday"}`, outcome: success},
		{name: "wrong wishlist", status: 200, body: `{"id":"00000000-0000-4000-8000-000000000002"}`, outcome: invalidResponse},
		{name: "missing id", status: 200, body: `{}`, outcome: invalidResponse},
		{name: "invalid JSON", status: 200, body: `<html>ok</html>`, outcome: invalidResponse},
		{name: "trailing garbage", status: 200, body: `{"id":"` + id + `"} invalid`, outcome: invalidResponse},
		{name: "oversized body", status: 200, body: strings.Repeat(" ", maxResponseSize+1), outcome: invalidResponse},
		{name: "HTTP error", status: 500, outcome: httpError},
		{name: "client error", status: 404, outcome: httpError},
		{name: "redirect is not success", status: 302, outcome: httpError},
		{name: "network error", err: io.ErrUnexpectedEOF, outcome: networkError},
		{name: "timeout", err: context.DeadlineExceeded, outcome: timeout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := testProber()
			calls := 0
			p.client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, http.MethodGet, req.Method)
				if tt.err != nil {
					return nil, tt.err
				}
				return &http.Response{
					StatusCode: tt.status, Body: io.NopCloser(strings.NewReader(tt.body)),
					Header: http.Header{"Location": []string{"/redirected"}},
				}, nil
			})
			result, status := p.check(t.Context())
			assert.Equal(t, tt.outcome, result)
			assert.Equal(t, tt.status, status)
			assert.Equal(t, 1, calls)
		})
	}
}

func TestCheckTimeoutIncludesBody(t *testing.T) {
	t.Parallel()
	p := testProber()
	p.client.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(contextReader{ctx: req.Context()})}, nil
	})
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	result, status := p.check(ctx)
	assert.Equal(t, timeout, result)
	assert.Equal(t, http.StatusOK, status)
}

func TestCompletedMinute(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name   string
		checks int
		allOK  bool
		good   string
		bad    string
	}{
		{name: "all twelve succeeded", checks: 12, allOK: true, good: "1", bad: "0"},
		{name: "failed check", checks: 12, allOK: false, good: "0", bad: "1"},
		{name: "missed check", checks: 11, allOK: true, good: "0", bad: "1"},
		{name: "empty minute", checks: 0, allOK: true, good: "0", bad: "1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := testProber()
			p.minuteStart = time.Date(2026, 10, 31, 10, 0, 0, 0, time.UTC)
			p.minuteChecks, p.minuteAllGood = tt.checks, tt.allOK
			end := p.minuteStart.Add(time.Minute)
			p.finishMinutes(end)
			p.finishMinutes(end)
			body := metricsBody(p)
			assert.Contains(t, body, `wishlist_probe_minutes_total{result="good"} `+tt.good+"\n")
			assert.Contains(t, body, `wishlist_probe_minutes_total{result="bad"} `+tt.bad+"\n")
			assert.Contains(t, body, "wishlist_probe_last_minute_success "+tt.good+"\n")
			assert.Equal(t, end, p.minuteStart)
		})
	}
}

func TestMissingMinutesAreBad(t *testing.T) {
	t.Parallel()
	p := testProber()
	p.minuteStart = time.Date(2026, 10, 31, 10, 0, 0, 0, time.UTC)
	p.minuteChecks, p.minuteAllGood = 12, true
	p.finishMinutes(p.minuteStart.Add(3 * time.Minute))
	body := metricsBody(p)
	assert.Contains(t, body, `wishlist_probe_minutes_total{result="good"} 1`)
	assert.Contains(t, body, `wishlist_probe_minutes_total{result="bad"} 2`)
	assert.Contains(t, body, "wishlist_probe_last_minute_success 0\n")
}

func TestRunCancellation(t *testing.T) {
	t.Parallel()
	p := testProber()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	p.Run(ctx)
	assert.Contains(t, metricsBody(p), "wishlist_probe_last_check_timestamp_seconds 0\n")
}

func testProber() *Prober {
	return New(&Config{
		URL: "http://example.test/api/v1/public/test", ExpectedID: uuid.MustParse("00000000-0000-4000-8000-000000000001"),
		Interval: 5 * time.Second, Timeout: 2 * time.Second,
	})
}

func metricsBody(p *Prober) string {
	recorder := httptest.NewRecorder()
	p.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	return recorder.Body.String()
}
