package handler

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"docapi/internal/service"
)

// RegisterRoutes attaches HTTP routes to the provided Fiber app.
// Keep handlers minimal and free of business logic in this skeleton.
func RegisterRoutes(app *fiber.App, db *sql.DB, docSvc service.DocumentService) {
	// Serve OpenAPI spec and Swagger UI
	app.Get("/openapi.yaml", func(c *fiber.Ctx) error {
		c.Type("yaml")
		return c.SendFile("openapi.yaml")
	})
	app.Get("/docs", func(c *fiber.Ctx) error {
		html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: '/openapi.yaml',
      dom_id: '#swagger-ui',
      presets: [SwaggerUIBundle.presets.apis],
      layout: 'BaseLayout'
    });
  </script>
</body>
</html>`
		return c.Type("html").SendString(html)
	})

	// New health endpoint: checks DB connectivity only
	app.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			return writeError(c, fiber.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "dependency unavailable")
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "healthy"})
	})

	// Backward-compatible simple liveness probe
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// List documents endpoint with limit & offset
	app.Get("/documents", func(c *fiber.Ctx) error {
		limitStr := c.Query("limit", "10")
		offsetStr := c.Query("offset", "0")
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return writeError(c, fiber.StatusBadRequest, "INVALID_LIMIT", "invalid limit")
		}
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			return writeError(c, fiber.StatusBadRequest, "INVALID_OFFSET", "invalid offset")
		}

		res, err := docSvc.List(c.UserContext(), limit, offset)
		if err != nil {
			return writeError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		}
		return c.JSON(res)
	})

	// Upload document endpoint (multipart/form-data, field name: file)
	app.Post("/documents", func(c *fiber.Ctx) error {
		fh, err := c.FormFile("file")
		if err != nil {
			return writeError(c, fiber.StatusBadRequest, "FILE_REQUIRED", "file is required")
		}

		f, err := fh.Open()
		if err != nil {
			return writeError(c, fiber.StatusBadRequest, "FILE_OPEN_ERROR", "cannot open uploaded file")
		}
		defer f.Close()

		ct := fh.Header.Get("Content-Type")
		if ct == "" {
			ct = "application/octet-stream"
		}

		doc, err := docSvc.Upload(c.UserContext(), f, fh.Filename, ct, fh.Size)
		if err != nil {
			return writeError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		}
		return c.Status(fiber.StatusCreated).JSON(doc)
	})

	// Get document by ID
	app.Get("/documents/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		if _, err := uuid.Parse(id); err != nil {
			return writeError(c, fiber.StatusBadRequest, "INVALID_ID", "invalid id format")
		}
		doc, err := docSvc.Get(c.UserContext(), id)
		if err != nil {
			// Translate not found
			if errors.Is(err, sql.ErrNoRows) {
				return writeError(c, fiber.StatusNotFound, "NOT_FOUND", "document not found")
			}
			return writeError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		}
		return c.JSON(doc)
	})

	// Delete document by ID
	app.Delete("/documents/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		if _, err := uuid.Parse(id); err != nil {
			return writeError(c, fiber.StatusBadRequest, "INVALID_ID", "invalid id format")
		}
		if err := docSvc.Delete(c.UserContext(), id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return writeError(c, fiber.StatusNotFound, "NOT_FOUND", "document not found")
			}
			return writeError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		}
		return c.SendStatus(fiber.StatusNoContent)
	})
}
