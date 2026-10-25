package metricsadapter

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry      *prometheus.Registry
	registrations prometheus.Counter
	logins        prometheus.Counter
	wishlists     prometheus.Counter
	reservations  prometheus.Counter
	requests      *prometheus.CounterVec
	duration      *prometheus.HistogramVec
}

func New() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		registrations: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wishlist_user_registrations_total", Help: "Successful user registrations.",
		}),
		logins: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wishlist_user_logins_total", Help: "Successful user logins.",
		}),
		wishlists: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wishlist_wishlists_created_total", Help: "Successfully created wishlists.",
		}),
		reservations: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wishlist_items_reserved_total", Help: "Successful gift reservations.",
		}),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wishlist_http_requests_total", Help: "Completed HTTP requests.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "wishlist_http_request_duration_seconds", Help: "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route", "status"}),
	}
	m.registry.MustRegister(
		collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.registrations, m.logins, m.wishlists, m.reservations, m.requests, m.duration,
	)
	return m
}

func (m *Metrics) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{EnableOpenMetrics: true}))
	return mux
}

func (m *Metrics) UserRegistered()  { m.registrations.Inc() }
func (m *Metrics) UserLoggedIn()    { m.logins.Inc() }
func (m *Metrics) WishlistCreated() { m.wishlists.Inc() }
func (m *Metrics) ItemReserved()    { m.reservations.Inc() }

func (m *Metrics) ObserveHTTPRequest(method, route string, status int, seconds float64) {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodConnect, http.MethodTrace:
	default:
		method = "OTHER"
	}
	m.requests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	m.duration.WithLabelValues(method, route, strconv.Itoa(status)).Observe(seconds)
}
