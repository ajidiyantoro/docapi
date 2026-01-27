package postgres

import (
	"context"
	"database/sql"
	"errors"

	"docapi/internal/model"
	"docapi/internal/repository"
)

// DocumentPostgres is a PostgreSQL implementation of repository.DocumentRepository.
// It uses database/sql with parameterized queries and contains no business logic.
type DocumentPostgres struct {
	db *sql.DB
}

// NewDocumentPostgres creates a new DocumentPostgres repository.
func NewDocumentPostgres(db *sql.DB) *DocumentPostgres {
	return &DocumentPostgres{db: db}
}

var _ repository.DocumentRepository = (*DocumentPostgres)(nil)

// Create inserts a new document row and returns the stored record.
func (r *DocumentPostgres) Create(ctx context.Context, doc *model.Document) (*model.Document, error) {
	const q = `
		INSERT INTO documents (id, filename, storage_path, size, content_type, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, filename, storage_path, size, content_type, created_at
	`
	row := r.db.QueryRowContext(ctx, q,
		doc.ID,
		doc.Filename,
		doc.StoragePath,
		doc.Size,
		doc.ContentType,
		doc.CreatedAt,
	)
	var out model.Document
	if err := row.Scan(
		&out.ID,
		&out.Filename,
		&out.StoragePath,
		&out.Size,
		&out.ContentType,
		&out.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &out, nil
}

// FindByID fetches a single document by its ID.
func (r *DocumentPostgres) FindByID(ctx context.Context, id string) (*model.Document, error) {
	const q = `
		SELECT id, filename, storage_path, size, content_type, created_at
		FROM documents
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, q, id)
	var d model.Document
	if err := row.Scan(
		&d.ID,
		&d.Filename,
		&d.StoragePath,
		&d.Size,
		&d.ContentType,
		&d.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	return &d, nil
}

// List returns documents using LIMIT/OFFSET pagination and a total count.
func (r *DocumentPostgres) List(ctx context.Context, pq repository.PageQuery) (*repository.PageResult[model.Document], error) {
	// Count total rows
	const qCount = `SELECT COUNT(*) FROM documents`
	var total int
	if err := r.db.QueryRowContext(ctx, qCount).Scan(&total); err != nil {
		return nil, err
	}

	// Fetch page
	const qList = `
		SELECT id, filename, storage_path, size, content_type, created_at
		FROM documents
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, qList, pq.Limit, pq.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.Document, 0)
	for rows.Next() {
		var d model.Document
		if err := rows.Scan(
			&d.ID,
			&d.Filename,
			&d.StoragePath,
			&d.Size,
			&d.ContentType,
			&d.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &repository.PageResult[model.Document]{
		Items: items,
		Total: total,
	}, nil
}

// Delete removes a document by ID. It does not return an error if the row does not exist.
func (r *DocumentPostgres) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM documents WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	// Ignore rows affected to keep behavior simple per requirement (no business logic).
	_, _ = res.RowsAffected()
	return nil
}
