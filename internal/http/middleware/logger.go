package middleware

import (
    "encoding/json"
    "os"
    "time"

    "github.com/gofiber/fiber/v2"
)

// Logger is a middleware that logs each HTTP request in JSON format.
// Required fields:
// - request_id (taken from context locals set by RequestID middleware)
// - method
// - path
// - status
// - latency (in milliseconds, as float)
func Logger() fiber.Handler {
    // Prepare a JSON encoder that writes one JSON object per line to stdout.
    enc := json.NewEncoder(os.Stdout)

    return func(c *fiber.Ctx) error {
        start := time.Now()

        // Process request
        err := c.Next()

        // Collect fields after handler executed to capture final status
        rid, _ := c.Locals(RequestIDLocalKey).(string)
        method := c.Method()
        // Use only the path segment (no query string) to match requirement naming
        path := c.Path()
        status := c.Response().StatusCode()
        latency := float64(time.Since(start).Milliseconds())

        _ = enc.Encode(map[string]any{
            "request_id": rid,
            "method":     method,
            "path":       path,
            "status":     status,
            "latency":    latency,
        })

        return err
    }
}
