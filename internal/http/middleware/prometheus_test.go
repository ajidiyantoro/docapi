package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestPrometheusMiddleware(t *testing.T) {
	// Use a fresh registry for each test to avoid "duplicate registration" panic
	reg := prometheus.NewRegistry()
	promMiddleware, err := NewPrometheusMiddleware(reg)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	app := fiber.New()
	app.Use(promMiddleware.Handler())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "bad request")
	})

	// Test 1: request ke endpoint normal menambah http_requests_total
	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Verify metric
	count := testutil.ToFloat64(promMiddleware.requestCount.WithLabelValues("GET", "/test", "200"))
	if count != 1 {
		t.Errorf("expected count 1, got %f", count)
	}

	// Test Error endpoint
	reqErr := httptest.NewRequest("GET", "/error", nil)
	app.Test(reqErr)

	countErr := testutil.ToFloat64(promMiddleware.requestCount.WithLabelValues("GET", "/error", "400"))
	if countErr != 1 {
		t.Errorf("expected count 1 for error, got %f", countErr)
	}
}

func TestPrometheusMiddleware_ExcludeMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	promMiddleware, err := NewPrometheusMiddleware(reg)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	app := fiber.New()
	app.Use(promMiddleware.Handler())

	app.Get("/metrics", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// Test 2: /metrics tidak menambah counter
	req := httptest.NewRequest("GET", "/metrics", nil)
	app.Test(req)

	// We can't easily check that NO metrics were added without knowing all possible labels
	// but we can check the total count of the collector if we use testutil.CollectAndCount
	count := testutil.CollectAndCount(promMiddleware.requestCount)
	if count != 0 {
		t.Errorf("expected 0 metrics collected for /metrics, got %d", count)
	}
}

func TestPrometheusMiddleware_PathPattern(t *testing.T) {
	reg := prometheus.NewRegistry()
	promMiddleware, err := NewPrometheusMiddleware(reg)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	app := fiber.New()
	app.Use(promMiddleware.Handler())

	app.Get("/documents/:id", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// Request with actual ID
	req := httptest.NewRequest("GET", "/documents/123", nil)
	app.Test(req)

	// Should use /documents/:id as label, not /documents/123
	count := testutil.ToFloat64(promMiddleware.requestCount.WithLabelValues("GET", "/documents/:id", "200"))
	if count != 1 {
		t.Errorf("expected count 1 for pattern /documents/:id, got %f", count)
	}

	// Verify Histogram also recorded
	countDur := testutil.CollectAndCount(promMiddleware.requestDuration)
	if countDur == 0 {
		t.Error("expected histogram metrics to be collected, got 0")
	}
}
