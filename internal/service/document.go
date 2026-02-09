package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"docapi/internal/model"
	"docapi/internal/repository"
	"docapi/internal/storage"
)

var (
	ErrIDRequired = errors.New("id is required")
	ErrNotFound   = errors.New("document not found")
	ErrReaderNil  = errors.New("reader is nil")
)

// DocumentListResult is the service-level DTO for paginated documents.
type DocumentListResult struct {
	Items []model.Document `json:"data"`
	Total int              `json:"total"`
}

// DocumentService defines the use cases for handling documents.
type DocumentService interface {
	// Upload uploads the content to object storage, saves metadata to DB, and rolls back storage if DB save fails.
	// - originalFilename is used only to extract extension; stored filename will be UUID + original extension.
	Upload(ctx context.Context, r io.Reader, originalFilename string, contentType string, size int64) (*model.Document, error)

	// List returns documents using limit/offset and a total count.
	List(ctx context.Context, limit, offset int) (*DocumentListResult, error)

	// Get returns a single document by its ID.
	Get(ctx context.Context, id string) (*model.Document, error)

	// Delete removes a document by ID from both storage and repository.
	Delete(ctx context.Context, id string) error
}

// documentService is a concrete implementation of DocumentService.
type documentService struct {
	store storage.Storage
	repo  repository.DocumentRepository
}

// NewDocumentService constructs a new DocumentService.
func NewDocumentService(store storage.Storage, repo repository.DocumentRepository) DocumentService {
	return &documentService{store: store, repo: repo}
}

func (s *documentService) Upload(ctx context.Context, r io.Reader, originalFilename string, contentType string, size int64) (*model.Document, error) {
	if r == nil {
		return nil, ErrReaderNil
	}
	// Generate filename using UUID + extension
	ext := filepath.Ext(originalFilename)
	genName := uuid.New().String() + ext
	key := filepath.ToSlash(filepath.Join("documents", genName))

	// Upload to object storage
	objInfo, err := s.store.Put(ctx, key, r, storage.PutObjectOptions{
		Size:        size,
		ContentType: contentType,
		Metadata: map[string]string{
			"original-filename": originalFilename,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	// Save metadata to database
	doc := &model.Document{
		ID:          uuid.New().String(),
		Filename:    genName,
		StoragePath: objInfo.Key,
		Size:        objInfo.Size,
		ContentType: objInfo.ContentType,
		CreatedAt:   time.Now().UTC(),
	}
	stored, err := s.repo.Create(ctx, doc)
	if err != nil {
		// Rollback: delete the object from storage
		if delErr := s.store.Delete(ctx, key); delErr != nil {
			return nil, fmt.Errorf("db save failed: %v; rollback delete failed: %v", err, delErr)
		}
		return nil, fmt.Errorf("db save failed: %w", err)
	}
	return stored, nil
}

// List returns paginated documents without exposing repository types.
func (s *documentService) List(ctx context.Context, limit, offset int) (*DocumentListResult, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	res, err := s.repo.List(ctx, repository.PageQuery{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	return &DocumentListResult{Items: res.Items, Total: res.Total}, nil
}

// Get returns a document by ID.
func (s *documentService) Get(ctx context.Context, id string) (*model.Document, error) {
	if id == "" {
		return nil, ErrIDRequired
	}
	doc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return doc, nil
}

// Delete removes a document from storage, then deletes its record.
func (s *documentService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return ErrIDRequired
	}
	// Find the document to get its storage path
	doc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	// Delete from storage first; if this fails, keep DB row to avoid orphaned storage reference loss
	if err := s.store.Delete(ctx, doc.StoragePath); err != nil {
		return fmt.Errorf("delete storage: %w", err)
	}
	// Delete DB row (repository ignores missing row errors as per contract)
	return s.repo.Delete(ctx, id)
}
