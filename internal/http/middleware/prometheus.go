package middleware

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMiddleware holds the prometheus metrics and registry.
type PrometheusMiddleware struct {
	requestCount *prometheus.CounterVec
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
	}

	if err := reg.Register(m.requestCount); err != nil {
		// Return error if already registered, or we can use MustRegister but better handle it.
		// If it's already registered and we want to reuse it, we might need a different approach.
		// But usually per registry it should be unique.
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

		// Process the request
		err := c.Next()

		// Get path pattern (e.g., /documents/:id instead of /documents/123)
		path := c.Route().Path
		if path == "" {
			path = c.Path() // Fallback to raw path if route not found (e.g. 404)
		}

		status := c.Response().StatusCode()
		if err != nil {
			if fiberErr, ok := err.(*fiber.Error); ok {
				status = fiberErr.Code
			} else {
				// Default to 500 if error is not a fiber.Error
				// This depends on how ErrorHandler is implemented
				status = fiber.StatusInternalServerError
			}
		}

		m.requestCount.WithLabelValues(
			c.Method(),
			path,
			strconv.Itoa(status),
		).Inc()

		return err
	}
}
