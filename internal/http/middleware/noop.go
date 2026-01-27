package middleware

import "github.com/gofiber/fiber/v2"

// Noop is a minimal middleware that simply calls the next handler.
// It exists to demonstrate middleware wiring in the project structure.
func Noop() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}
