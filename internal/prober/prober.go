package prober

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Prober struct {
	cfg             *Config
	client          *http.Client
	registry        *prometheus.Registry
	checks          *prometheus.CounterVec
	minutes         *prometheus.CounterVec
	success         prometheus.Gauge
	duration        prometheus.Gauge
	lastCheck       prometheus.Gauge
	minuteSuccess   prometheus.Gauge
	minuteTimestamp prometheus.Gauge
	minuteStart     time.Time
	minuteChecks    int
	minuteAllGood   bool
}

const (
	success         = "success"
	httpError       = "http_error"
	invalidResponse = "invalid_response"
	networkError    = "network_error"
	timeout         = "timeout"
	missed          = "missed"
	maxResponseSize = 1 << 20
)

func New(cfg *Config) *Prober {
	p := &Prober{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				Proxy:             http.ProxyFromEnvironment,
				DisableKeepAlives: true,
			},
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
		},
		registry: prometheus.NewRegistry(),
		checks: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wishlist_probe_checks_total", Help: "Probe checks by result, including missed scheduled checks.",
		}, []string{"result"}),
		minutes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wishlist_probe_minutes_total", Help: "Completed UTC minutes by result.",
		}, []string{"result"}),
		success: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "wishlist_probe_success", Help: "Whether the last check succeeded (1 or 0).",
		}),
		duration: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "wishlist_probe_duration_seconds", Help: "Duration of the last check, including reading the response.",
		}),
		lastCheck: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "wishlist_probe_last_check_timestamp_seconds", Help: "Unix timestamp of the last completed check.",
		}),
		minuteSuccess: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "wishlist_probe_last_minute_success", Help: "Whether all expected checks in the last completed UTC minute succeeded.",
		}),
		minuteTimestamp: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "wishlist_probe_last_minute_timestamp_seconds", Help: "Unix timestamp of the end of the last completed UTC minute.",
		}),
	}
	for _, result := range []string{success, httpError, invalidResponse, networkError, timeout, missed} {
		p.checks.WithLabelValues(result)
	}
	for _, result := range []string{"good", "bad"} {
		p.minutes.WithLabelValues(result)
	}
	p.registry.MustRegister(p.checks, p.minutes, p.success, p.duration, p.lastCheck, p.minuteSuccess, p.minuteTimestamp)
	return p
}

func (p *Prober) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{EnableOpenMetrics: true}))
	return mux
}

func (p *Prober) Run(ctx context.Context) {
	p.minuteStart = time.Now().UTC().Truncate(time.Minute)
	p.minuteAllGood = true
	next := time.Now().Truncate(p.cfg.Interval).Add(p.cfg.Interval)
	timer := time.NewTimer(time.Until(next))
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		p.finishMinutes(time.Now())
		result, status, duration := missed, 0, time.Duration(0)
		deadline := next.Add(p.cfg.Timeout)
		if time.Now().Before(deadline) {
			checkCtx, cancel := context.WithDeadline(ctx, deadline)
			start := time.Now()
			result, status = p.check(checkCtx)
			duration = time.Since(start)
			cancel()
		}
		if ctx.Err() != nil {
			return
		}
		p.record(result, status, duration)
		if next.Truncate(time.Minute).Equal(p.minuteStart) {
			p.minuteChecks++
			p.minuteAllGood = p.minuteAllGood && result == success
		}
		p.finishMinutes(time.Now())
		next = time.Now().Truncate(p.cfg.Interval).Add(p.cfg.Interval)
		timer.Reset(time.Until(next))
	}
}

func (p *Prober) check(ctx context.Context) (string, int) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.URL, nil)
	if err != nil {
		return networkError, 0
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return requestError(err), 0
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return httpError, resp.StatusCode
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return requestError(err), resp.StatusCode
	}
	var wishlist struct {
		ID uuid.UUID `json:"id"`
	}
	if len(body) > maxResponseSize || json.Unmarshal(body, &wishlist) != nil || wishlist.ID != p.cfg.ExpectedID {
		return invalidResponse, resp.StatusCode
	}
	if ctx.Err() != nil {
		return requestError(ctx.Err()), resp.StatusCode
	}
	return success, resp.StatusCode
}

func (p *Prober) record(result string, status int, duration time.Duration) {
	p.checks.WithLabelValues(result).Inc()
	p.success.Set(0)
	level := slog.LevelWarn
	if result == success {
		p.success.Set(1)
		level = slog.LevelInfo
	}
	p.duration.Set(duration.Seconds())
	p.lastCheck.Set(float64(time.Now().Unix()))
	slog.Log(context.Background(), level, "probe completed", "result", result, "status", status, "duration_seconds", duration.Seconds())
}

func (p *Prober) finishMinutes(now time.Time) {
	for end := p.minuteStart.Add(time.Minute); !end.After(now); end = p.minuteStart.Add(time.Minute) {
		good := p.minuteChecks == int(time.Minute/p.cfg.Interval) && p.minuteAllGood
		result := "bad"
		p.minuteSuccess.Set(0)
		if good {
			result = "good"
			p.minuteSuccess.Set(1)
		}
		p.minutes.WithLabelValues(result).Inc()
		p.minuteTimestamp.Set(float64(end.Unix()))
		slog.Info("probe minute completed", "minute_end", end.UTC(), "result", result, "checks", p.minuteChecks)
		p.minuteStart, p.minuteChecks, p.minuteAllGood = end, 0, true
	}
}

func requestError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return timeout
	}
	return networkError
}
