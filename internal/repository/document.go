package repository

import (
	"context"

	"docapi/internal/model"
)

// DocumentRepository defines data access for documents using SQL queries only.
// No business logic here — strictly persistence operations.
type DocumentRepository interface {
	// Create inserts a new document record.
	// The caller should provide required fields (e.g., ID, CreatedAt) according to the database schema defaults.
	// Returns the stored document (may include values set by the DB).
	Create(ctx context.Context, doc *model.Document) (*model.Document, error)

	// FindByID returns a document by its ID.
	FindByID(ctx context.Context, id string) (*model.Document, error)

	// List returns a paginated list of documents and total rows count for the given filter.
	List(ctx context.Context, pq PageQuery) (*PageResult[model.Document], error)

	// Delete removes a document by ID. It returns nil if the row was deleted or did not exist.
	Delete(ctx context.Context, id string) error
}

// PageQuery holds limit/offset pagination parameters.
type PageQuery struct {
	Limit  int
	Offset int
}

// PageResult is a generic pagination result wrapper.
// T is typically a model type.
type PageResult[T any] struct {
	Items []T
	Total int
}
