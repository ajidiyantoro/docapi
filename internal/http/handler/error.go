package handler

import (
	"github.com/gofiber/fiber/v2"

	"docapi/internal/http/middleware"
)

// errorPayload defines the standardized error response body.
type errorPayload struct {
	RequestID string        `json:"request_id"`
	Error     errorEnvelope `json:"error"`
}

type errorEnvelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// requestIDFromCtx extracts request_id previously stored by middleware.RequestID.
func requestIDFromCtx(c *fiber.Ctx) string {
	if v := c.Locals(middleware.RequestIDLocalKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// writeError writes a standardized JSON error response without leaking internal errors.
//
// Parameters:
// - status: HTTP status code to return
// - code: machine-readable short error code (e.g., "INVALID_ID", "NOT_FOUND", "INTERNAL_ERROR")
// - message: human-readable safe message (no internal details)
func writeError(c *fiber.Ctx, status int, code, message string) error {
	res := errorPayload{
		RequestID: requestIDFromCtx(c),
		Error: errorEnvelope{
			Code:    code,
			Message: message,
		},
	}
	return c.Status(status).JSON(res)
}

// ErrorHandler returns a Fiber global error handler that standardizes error responses.
func ErrorHandler() fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		status := fiber.StatusInternalServerError
		if e, ok := err.(*fiber.Error); ok {
			status = e.Code
		}

		switch status {
		case fiber.StatusBadRequest:
			return writeError(c, status, "BAD_REQUEST", "bad request")
		case fiber.StatusNotFound:
			return writeError(c, status, "NOT_FOUND", "resource not found")
		case fiber.StatusMethodNotAllowed:
			return writeError(c, status, "METHOD_NOT_ALLOWED", "method not allowed")
		default:
			return writeError(c, status, "INTERNAL_ERROR", "internal server error")
		}
	}
}
