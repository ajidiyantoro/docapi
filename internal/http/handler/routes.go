package handler

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	_ "docapi/docs"
	_ "docapi/internal/model"
	"docapi/internal/service"
)

// HealthCheck handles the health check request.
// @Summary Health check
// @Description Check database connectivity
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} errorPayload
// @Router /health [get]
func HealthCheck(db *sql.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			return writeError(c, fiber.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "dependency unavailable")
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "healthy"})
	}
}

// LivenessProbe handles the liveness probe request.
// @Summary Liveness probe
// @Description Simple liveness probe
// @Tags health
// @Success 200 {string} string "OK"
// @Router /healthz [get]
func LivenessProbe() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	}
}

// ListDocuments handles listing documents.
// @Summary List documents
// @Description Get a list of documents with pagination
// @Tags documents
// @Produce json
// @Param limit query int false "Limit" default(10)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} model.Document
// @Failure 400 {object} errorPayload
// @Failure 500 {object} errorPayload
// @Router /documents [get]
func ListDocuments(docSvc service.DocumentService) fiber.Handler {
	return func(c *fiber.Ctx) error {
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
	}
}

// UploadDocument handles document upload.
// @Summary Upload document
// @Description Upload a new document
// @Tags documents
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Document file"
// @Success 201 {object} model.Document
// @Failure 400 {object} errorPayload
// @Failure 500 {object} errorPayload
// @Router /documents [post]
func UploadDocument(docSvc service.DocumentService) fiber.Handler {
	return func(c *fiber.Ctx) error {
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
	}
}

// GetDocument handles getting a document by ID.
// @Summary Get document
// @Description Get a document by ID
// @Tags documents
// @Produce json
// @Param id path string true "Document ID"
// @Success 200 {object} model.Document
// @Failure 400 {object} errorPayload
// @Failure 404 {object} errorPayload
// @Failure 500 {object} errorPayload
// @Router /documents/{id} [get]
func GetDocument(docSvc service.DocumentService) fiber.Handler {
	return func(c *fiber.Ctx) error {
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
	}
}

// DeleteDocument handles deleting a document by ID.
// @Summary Delete document
// @Description Delete a document by ID
// @Tags documents
// @Param id path string true "Document ID"
// @Success 204 "No Content"
// @Failure 400 {object} errorPayload
// @Failure 404 {object} errorPayload
// @Failure 500 {object} errorPayload
// @Router /documents/{id} [delete]
func DeleteDocument(docSvc service.DocumentService) fiber.Handler {
	return func(c *fiber.Ctx) error {
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
	}
}

// RegisterRoutes attaches HTTP routes to the provided Fiber app.
// Keep handlers minimal and free of business logic in this skeleton.
func RegisterRoutes(app *fiber.App, db *sql.DB, docSvc service.DocumentService) {
	// Health check endpoint: checks DB connectivity only
	app.Get("/health", HealthCheck(db))

	// Backward-compatible simple liveness probe
	app.Get("/healthz", LivenessProbe())

	// List documents endpoint with limit & offset
	app.Get("/documents", ListDocuments(docSvc))

	// Upload document endpoint (multipart/form-data, field name: file)
	app.Post("/documents", UploadDocument(docSvc))

	// Get document by ID
	app.Get("/documents/:id", GetDocument(docSvc))

	// Delete document by ID
	app.Delete("/documents/:id", DeleteDocument(docSvc))
}
