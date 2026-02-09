package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"docapi/internal/model"
	"docapi/internal/repository"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestDocumentPostgres_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewDocumentPostgres(db)
	ctx := context.Background()

	now := time.Now().UTC()
	doc := &model.Document{
		ID:          "test-uuid",
		Filename:    "test.txt",
		StoragePath: "documents/test.txt",
		Size:        123,
		ContentType: "text/plain",
		CreatedAt:   now,
	}

	rows := sqlmock.NewRows([]string{"id", "filename", "storage_path", "size", "content_type", "created_at"}).
		AddRow(doc.ID, doc.Filename, doc.StoragePath, doc.Size, doc.ContentType, doc.CreatedAt)

	mock.ExpectQuery("INSERT INTO documents").
		WithArgs(doc.ID, doc.Filename, doc.StoragePath, doc.Size, doc.ContentType, doc.CreatedAt).
		WillReturnRows(rows)

	result, err := repo.Create(ctx, doc)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, doc.ID, result.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentPostgres_FindByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewDocumentPostgres(db)
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "filename", "storage_path", "size", "content_type", "created_at"}).
			AddRow("test-id", "file.txt", "path/file.txt", 100, "text/plain", time.Now())

		mock.ExpectQuery("SELECT (.+) FROM documents WHERE id = ?").
			WithArgs("test-id").
			WillReturnRows(rows)

		doc, err := repo.FindByID(ctx, "test-id")

		assert.NoError(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "test-id", doc.ID)
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM documents WHERE id = ?").
			WithArgs("missing").
			WillReturnError(sql.ErrNoRows)

		doc, err := repo.FindByID(ctx, "missing")

		assert.Error(t, err)
		assert.True(t, IsNoRowsError(err))
		assert.Nil(t, doc)
	})
}

func TestDocumentPostgres_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewDocumentPostgres(db)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM documents").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		rows := sqlmock.NewRows([]string{"id", "filename", "storage_path", "size", "content_type", "created_at"}).
			AddRow("test-id", "file.txt", "path/file.txt", 100, "text/plain", time.Now())

		mock.ExpectQuery("SELECT (.+) FROM documents ORDER BY").
			WithArgs(10, 0).
			WillReturnRows(rows)

		res, err := repo.List(ctx, repository.PageQuery{Limit: 10, Offset: 0})

		assert.NoError(t, err)
		assert.Equal(t, 1, res.Total)
		assert.Len(t, res.Items, 1)
	})
}

func TestDocumentPostgres_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewDocumentPostgres(db)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM documents WHERE id = ?").
		WithArgs("test-id").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Delete(ctx, "test-id")

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func IsNoRowsError(err error) bool {
	return err == sql.ErrNoRows
}
