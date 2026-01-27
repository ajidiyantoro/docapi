package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the standard header name used to propagate request IDs.
	RequestIDHeader = "X-Request-ID"
	// RequestIDLocalKey is the key used to store the request ID in Fiber's context locals.
	RequestIDLocalKey = "request_id"
)

// RequestID is a reusable middleware that ensures every request has a request ID.
//
// Behavior:
// - Reads X-Request-ID from the incoming request header.
// - If missing, generates a new UUID.
// - Stores the value in Fiber context locals under RequestIDLocalKey.
// - Adds X-Request-ID to the response header with the same value.
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Get(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}

		// Store in context for downstream handlers/middlewares
		c.Locals(RequestIDLocalKey, id)

		// Ensure the response carries the request ID
		c.Set(RequestIDHeader, id)

		return c.Next()
	}
}
