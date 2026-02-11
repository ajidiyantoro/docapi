package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMiddleware holds the prometheus metrics and registry.
type PrometheusMiddleware struct {
	requestCount    *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

// NewPrometheusMiddleware creates a new PrometheusMiddleware.
func NewPrometheusMiddleware(reg prometheus.Registerer) (*PrometheusMiddleware, error) {
	m := &PrometheusMiddleware{
		requestCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests processed.",
			},
			[]string{"method", "path", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route", "status"},
		),
	}

	if err := reg.Register(m.requestCount); err != nil {
		return nil, err
	}

	if err := reg.Register(m.requestDuration); err != nil {
		return nil, err
	}

	return m, nil
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
		method := c.Method()

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
