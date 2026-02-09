package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"docapi/internal/model"
	"docapi/internal/service"
	serviceMocks "docapi/internal/service/mocks"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck(t *testing.T) {
	db, dbMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	app := fiber.New()
	app.Get("/health", HealthCheck(db))

	t.Run("healthy", func(t *testing.T) {
		dbMock.ExpectPing().WillReturnError(nil)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		assert.Equal(t, "healthy", body["status"])
	})

	t.Run("unhealthy", func(t *testing.T) {
		dbMock.ExpectPing().WillReturnError(errors.New("db error"))

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

		var body errorPayload
		json.NewDecoder(resp.Body).Decode(&body)
		assert.Equal(t, "SERVICE_UNAVAILABLE", body.Error.Code)
	})
}

func TestLivenessProbe(t *testing.T) {
	app := fiber.New()
	app.Get("/healthz", LivenessProbe())

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp, _ := app.Test(req)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestListDocuments(t *testing.T) {
	mockSvc := new(serviceMocks.MockDocumentService)
	app := fiber.New()
	app.Get("/documents", ListDocuments(mockSvc))

	t.Run("success", func(t *testing.T) {
		expectedRes := &service.DocumentListResult{
			Items: []model.Document{{ID: uuid.New().String(), Filename: "test.pdf"}},
			Total: 1,
		}
		mockSvc.On("List", mock.Anything, 10, 0).Return(expectedRes, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/documents?limit=10&offset=0", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result service.DocumentListResult
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Len(t, result.Items, 1)
		assert.Equal(t, 1, result.Total)
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/documents?limit=abc", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		var body errorPayload
		json.NewDecoder(resp.Body).Decode(&body)
		assert.Equal(t, "INVALID_LIMIT", body.Error.Code)
	})

	t.Run("service error", func(t *testing.T) {
		mockSvc.On("List", mock.Anything, 10, 0).Return(nil, errors.New("service error")).Once()

		req := httptest.NewRequest(http.MethodGet, "/documents", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockSvc.AssertExpectations(t)
	})
}

func TestUploadDocument(t *testing.T) {
	mockSvc := new(serviceMocks.MockDocumentService)
	app := fiber.New()
	app.Post("/documents", UploadDocument(mockSvc))

	t.Run("success", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.txt")
		part.Write([]byte("hello world"))
		writer.Close()

		expectedDoc := &model.Document{ID: uuid.New().String(), Filename: "test.txt"}
		mockSvc.On("Upload", mock.Anything, mock.Anything, "test.txt", mock.Anything, mock.Anything).Return(expectedDoc, nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/documents", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result model.Document
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, expectedDoc.ID, result.ID)
		mockSvc.AssertExpectations(t)
	})

	t.Run("no file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/documents", nil)
		// Missing content-type and body
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		var res errorPayload
		json.NewDecoder(resp.Body).Decode(&res)
		assert.Equal(t, "FILE_REQUIRED", res.Error.Code)
	})

	t.Run("service error", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("file", "test.txt")
		part.Write([]byte("hello"))
		writer.Close()

		mockSvc.On("Upload", mock.Anything, mock.Anything, "test.txt", mock.Anything, mock.Anything).Return(nil, errors.New("upload failed")).Once()

		req := httptest.NewRequest(http.MethodPost, "/documents", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockSvc.AssertExpectations(t)
	})
}

func TestGetDocument(t *testing.T) {
	mockSvc := new(serviceMocks.MockDocumentService)
	app := fiber.New()
	app.Get("/documents/:id", GetDocument(mockSvc))

	t.Run("success", func(t *testing.T) {
		id := uuid.New().String()
		expectedDoc := &model.Document{ID: id, Filename: "test.txt"}
		mockSvc.On("Get", mock.Anything, id).Return(expectedDoc, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/documents/"+id, nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result model.Document
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, id, result.ID)
		mockSvc.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		id := uuid.New().String()
		mockSvc.On("Get", mock.Anything, id).Return(nil, sql.ErrNoRows).Once()

		req := httptest.NewRequest(http.MethodGet, "/documents/"+id, nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		var res errorPayload
		json.NewDecoder(resp.Body).Decode(&res)
		assert.Equal(t, "NOT_FOUND", res.Error.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/documents/invalid-uuid", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		var res errorPayload
		json.NewDecoder(resp.Body).Decode(&res)
		assert.Equal(t, "INVALID_ID", res.Error.Code)
	})

	t.Run("service error", func(t *testing.T) {
		id := uuid.New().String()
		mockSvc.On("Get", mock.Anything, id).Return(nil, errors.New("db error")).Once()

		req := httptest.NewRequest(http.MethodGet, "/documents/"+id, nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockSvc.AssertExpectations(t)
	})
}

func TestDeleteDocument(t *testing.T) {
	mockSvc := new(serviceMocks.MockDocumentService)
	app := fiber.New()
	app.Delete("/documents/:id", DeleteDocument(mockSvc))

	t.Run("success", func(t *testing.T) {
		id := uuid.New().String()
		mockSvc.On("Delete", mock.Anything, id).Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/documents/"+id, nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		mockSvc.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		id := uuid.New().String()
		mockSvc.On("Delete", mock.Anything, id).Return(sql.ErrNoRows).Once()

		req := httptest.NewRequest(http.MethodDelete, "/documents/"+id, nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		var res errorPayload
		json.NewDecoder(resp.Body).Decode(&res)
		assert.Equal(t, "NOT_FOUND", res.Error.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		id := uuid.New().String()
		mockSvc.On("Delete", mock.Anything, id).Return(errors.New("delete error")).Once()

		req := httptest.NewRequest(http.MethodDelete, "/documents/"+id, nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockSvc.AssertExpectations(t)
	})
}

func TestRouting(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler(),
	})

	mockSvc := new(serviceMocks.MockDocumentService)
	// Register all routes
	RegisterRoutes(app, nil, mockSvc)

	t.Run("not found route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/non-existent", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		var res errorPayload
		json.NewDecoder(resp.Body).Decode(&res)
		assert.Equal(t, "NOT_FOUND", res.Error.Code)
	})

	t.Run("method not allowed", func(t *testing.T) {
		// Health endpoint only allows GET
		req := httptest.NewRequest(http.MethodPost, "/health", nil)
		resp, _ := app.Test(req)

		// Fiber returns 405 by default if route exists but method doesn't match
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
		var res errorPayload
		json.NewDecoder(resp.Body).Decode(&res)
		assert.Equal(t, "METHOD_NOT_ALLOWED", res.Error.Code)
	})
}
