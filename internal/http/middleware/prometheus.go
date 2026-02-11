package middleware

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	requestCount    *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	once            sync.Once
)

// PrometheusMiddleware holds the prometheus metrics and registry.
type PrometheusMiddleware struct {
	requestCount    *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

// NewPrometheusMiddleware creates a new PrometheusMiddleware.
func NewPrometheusMiddleware(reg prometheus.Registerer) (*PrometheusMiddleware, error) {
	var errCount, errDuration error

	once.Do(func() {
		requestCount = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests processed.",
			},
			[]string{"method", "path", "status"},
		)

		requestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route", "status"},
		)

		errCount = reg.Register(requestCount)
		errDuration = reg.Register(requestDuration)
	})

	if errCount != nil {
		var are prometheus.AlreadyRegisteredError
		if !errors.As(errCount, &are) {
			return nil, errCount
		}
		requestCount = are.ExistingCollector.(*prometheus.CounterVec)
	}

	if errDuration != nil {
		var are prometheus.AlreadyRegisteredError
		if !errors.As(errDuration, &are) {
			return nil, errDuration
		}
		requestDuration = are.ExistingCollector.(*prometheus.HistogramVec)
	}

	return &PrometheusMiddleware{
		requestCount:    requestCount,
		requestDuration: requestDuration,
	}, nil
}

// Handler returns the fiber middleware handler.
func (m *PrometheusMiddleware) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Exclude /metrics from being counted
		if c.Path() == "/metrics" {
			return c.Next()
		}

		start := time.Now()

		// Process the request
		err := c.Next()

		duration := time.Since(start).Seconds()

		// Get path pattern (e.g., /documents/:id instead of /documents/123)
		path := c.Route().Path
		if path == "" {
			path = c.Path() // Fallback to raw path if route not found (e.g. 404)
		}

		route := "UNMATCHED"
		if c.Route() != nil && c.Route().Path != "" {
			route = c.Route().Path
		}

		status := c.Response().StatusCode()
		if err != nil {
			if fiberErr, ok := err.(*fiber.Error); ok {
				status = fiberErr.Code
			} else if status == 0 || status == fiber.StatusOK {
				// Default to 500 if error is not a fiber.Error and status is not set or 200
				status = fiber.StatusInternalServerError
			}
		}

		statusStr := strconv.Itoa(status)
		// Fix: Clone the method string to avoid buffer reuse issues
		method := strings.Clone(c.Method())

		m.requestCount.WithLabelValues(
			method,
			path,
			statusStr,
		).Inc()

		m.requestDuration.WithLabelValues(
			method,
			route,
			statusStr,
		).Observe(duration)

		return err
	}
}
