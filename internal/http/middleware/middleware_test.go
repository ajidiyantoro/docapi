package middleware

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestRequestID(t *testing.T) {
	app := fiber.New()
	app.Use(RequestID())

	app.Get("/test", func(c *fiber.Ctx) error {
		rid := c.Locals(RequestIDLocalKey)
		return c.SendString(rid.(string))
	})

	t.Run("should generate new request id if not present", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		ridHeader := resp.Header.Get(RequestIDHeader)
		assert.NotEmpty(t, ridHeader)

		// Check if it's readable in handler (from response body)
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		assert.Equal(t, ridHeader, buf.String())
	})

	t.Run("should preserve existing request id", func(t *testing.T) {
		existingID := "test-id-123"
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(RequestIDHeader, existingID)

		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		assert.Equal(t, existingID, resp.Header.Get(RequestIDHeader))

		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		assert.Equal(t, existingID, buf.String())
	})
}

func TestNoop(t *testing.T) {
	app := fiber.New()
	app.Use(Noop())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	assert.Equal(t, "ok", buf.String())
}

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	app := fiber.New()
	loc := time.UTC

	// Logger usually depends on RequestID for request_id field
	app.Use(RequestID())
	app.Use(LoggerWithWriter(&buf, loc))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusAccepted)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	assert.Equal(t, fiber.StatusAccepted, resp.StatusCode)

	// Verify log output
	var logData map[string]any
	err := json.Unmarshal(buf.Bytes(), &logData)
	assert.NoError(t, err)

	assert.NotEmpty(t, logData["request_id"])
	assert.Equal(t, "GET", logData["method"])
	assert.Equal(t, "/test", logData["path"])
	assert.Equal(t, float64(fiber.StatusAccepted), logData["status"])
	assert.NotNil(t, logData["latency"])
	assert.NotEmpty(t, logData["ts"])
}
