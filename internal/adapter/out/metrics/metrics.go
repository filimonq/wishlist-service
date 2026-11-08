package metricsadapter

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/filimonq/wishlist-service/internal/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry            *prometheus.Registry
	registrations       prometheus.Counter
	logins              prometheus.Counter
	loginFailures       *prometheus.CounterVec
	wishlists           prometheus.Counter
	wishlistFailures    *prometheus.CounterVec
	items               prometheus.Counter
	itemFailures        *prometheus.CounterVec
	reservations        prometheus.Counter
	reservationFailures *prometheus.CounterVec
	requests            *prometheus.CounterVec
	duration            *prometheus.HistogramVec
}

const internalErrorReason = "internal_error"

func New() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		registrations: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wishlist_user_registrations_total", Help: "Successful user registrations.",
		}),
		logins: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wishlist_user_logins_total", Help: "Successful user logins.",
		}),
		loginFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wishlist_user_login_failures_total", Help: "Failed login operations by reason.",
		}, []string{"reason"}),
		wishlists: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wishlist_wishlists_created_total", Help: "Successfully created wishlists.",
		}),
		wishlistFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wishlist_wishlist_creation_failures_total", Help: "Failed wishlist creation operations by reason.",
		}, []string{"reason"}),
		items: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wishlist_items_created_total", Help: "Successfully created gifts.",
		}),
		itemFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wishlist_item_creation_failures_total", Help: "Failed gift creation operations by reason.",
		}, []string{"reason"}),
		reservations: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wishlist_items_reserved_total", Help: "Successful gift reservations.",
		}),
		reservationFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wishlist_item_reservation_failures_total", Help: "Failed gift reservation operations by reason.",
		}, []string{"reason"}),
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
		m.registrations, m.logins, m.loginFailures, m.wishlists, m.reservations,
		m.reservationFailures, m.items, m.itemFailures, m.wishlistFailures, m.requests, m.duration,
	)
	for _, reason := range []string{"invalid_credentials", internalErrorReason} {
		m.loginFailures.WithLabelValues(reason)
	}
	for _, reason := range []string{"wishlist_not_found", "item_not_found", "item_not_in_wishlist", "already_reserved", internalErrorReason} {
		m.reservationFailures.WithLabelValues(reason)
	}
	for _, reason := range []string{"invalid_event_date", internalErrorReason} {
		m.wishlistFailures.WithLabelValues(reason)
	}
	for _, reason := range []string{"invalid_priority", "wishlist_not_found", "access_denied", internalErrorReason} {
		m.itemFailures.WithLabelValues(reason)
	}
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
func (m *Metrics) ItemCreated()     { m.items.Inc() }
func (m *Metrics) ItemReserved()    { m.reservations.Inc() }

func (m *Metrics) UserLoginFailed(err error) {
	reason := internalErrorReason
	if errors.Is(err, domain.ErrInvalidCredentials) {
		reason = "invalid_credentials"
	}
	m.loginFailures.WithLabelValues(reason).Inc()
}

func (m *Metrics) ItemReservationFailed(err error) {
	reason := internalErrorReason
	switch {
	case errors.Is(err, domain.ErrWishlistNotFound):
		reason = "wishlist_not_found"
	case errors.Is(err, domain.ErrItemNotFound):
		reason = "item_not_found"
	case errors.Is(err, domain.ErrItemNotInWishlist):
		reason = "item_not_in_wishlist"
	case errors.Is(err, domain.ErrItemAlreadyReserved):
		reason = "already_reserved"
	}
	m.reservationFailures.WithLabelValues(reason).Inc()
}

func (m *Metrics) WishlistCreationFailed(err error) {
	reason := internalErrorReason
	if errors.Is(err, domain.ErrEventDateInPast) {
		reason = "invalid_event_date"
	}
	m.wishlistFailures.WithLabelValues(reason).Inc()
}

func (m *Metrics) ItemCreationFailed(err error) {
	reason := internalErrorReason
	switch {
	case errors.Is(err, domain.ErrInvalidPriority):
		reason = "invalid_priority"
	case errors.Is(err, domain.ErrWishlistNotFound):
		reason = "wishlist_not_found"
	case errors.Is(err, domain.ErrWishlistAccessDenied):
		reason = "access_denied"
	}
	m.itemFailures.WithLabelValues(reason).Inc()
}

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
