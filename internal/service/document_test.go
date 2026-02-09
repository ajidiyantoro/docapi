package service

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"strings"
	"testing"

	"docapi/internal/model"
	"docapi/internal/repository"
	repoMocks "docapi/internal/repository/mocks"
	"docapi/internal/storage"
	storeMocks "docapi/internal/storage/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDocumentService_Upload(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name             string
		originalFilename string
		contentType      string
		size             int64
		setupMocks       func(mStore *storeMocks.MockStorage, mRepo *repoMocks.MockDocumentRepository) io.Reader
		wantErr          error
		wantErrMsg       string
	}{
		{
			name:             "happy path",
			originalFilename: "test.txt",
			contentType:      "text/plain",
			size:             11,
			setupMocks: func(mStore *storeMocks.MockStorage, mRepo *repoMocks.MockDocumentRepository) io.Reader {
				r := strings.NewReader("hello world")
				mStore.On("Put", ctx, mock.MatchedBy(func(key string) bool {
					return strings.HasPrefix(key, "documents/") && strings.HasSuffix(key, ".txt")
				}), r, storage.PutObjectOptions{
					Size:        11,
					ContentType: "text/plain",
					Metadata:    map[string]string{"original-filename": "test.txt"},
				}).Return(storage.ObjectInfo{
					Key:         "documents/uuid.txt",
					Size:        11,
					ContentType: "text/plain",
				}, nil)

				mRepo.On("Create", ctx, mock.MatchedBy(func(doc *model.Document) bool {
					return doc.Filename != "" && doc.StoragePath == "documents/uuid.txt"
				})).Return(&model.Document{ID: "gen-id"}, nil)

				return r
			},
			wantErr: nil,
		},
		{
			name:             "validation error - nil reader",
			originalFilename: "test.txt",
			setupMocks: func(mStore *storeMocks.MockStorage, mRepo *repoMocks.MockDocumentRepository) io.Reader {
				return nil
			},
			wantErr: ErrReaderNil,
		},
		{
			name:             "storage error",
			originalFilename: "test.txt",
			size:             5,
			setupMocks: func(mStore *storeMocks.MockStorage, mRepo *repoMocks.MockDocumentRepository) io.Reader {
				r := strings.NewReader("hello")
				mStore.On("Put", ctx, mock.Anything, r, mock.Anything).
					Return(storage.ObjectInfo{}, errors.New("storage fail"))
				return r
			},
			wantErrMsg: "upload to storage: storage fail",
		},
		{
			name:             "repository error with successful rollback",
			originalFilename: "test.txt",
			size:             5,
			setupMocks: func(mStore *storeMocks.MockStorage, mRepo *repoMocks.MockDocumentRepository) io.Reader {
				r := strings.NewReader("hello")
				mStore.On("Put", ctx, mock.Anything, r, mock.Anything).
					Return(func(ctx context.Context, key string, r io.Reader, opt storage.PutObjectOptions) storage.ObjectInfo {
						return storage.ObjectInfo{Key: key}
					}, nil)
				mRepo.On("Create", ctx, mock.Anything).
					Return(nil, errors.New("db fail"))
				mStore.On("Delete", ctx, mock.Anything).Return(nil)
				return r
			},
			wantErrMsg: "db save failed: db fail",
		},
		{
			name:             "repository error with failed rollback",
			originalFilename: "test.txt",
			size:             5,
			setupMocks: func(mStore *storeMocks.MockStorage, mRepo *repoMocks.MockDocumentRepository) io.Reader {
				r := strings.NewReader("hello")
				mStore.On("Put", ctx, mock.Anything, r, mock.Anything).
					Return(func(ctx context.Context, key string, r io.Reader, opt storage.PutObjectOptions) storage.ObjectInfo {
						return storage.ObjectInfo{Key: key}
					}, nil)
				mRepo.On("Create", ctx, mock.Anything).
					Return(nil, errors.New("db fail"))
				mStore.On("Delete", ctx, mock.Anything).Return(errors.New("delete fail"))
				return r
			},
			wantErrMsg: "rollback delete failed: delete fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mStore := new(storeMocks.MockStorage)
			mRepo := new(repoMocks.MockDocumentRepository)
			svc := NewDocumentService(mStore, mRepo)

			r := tt.setupMocks(mStore, mRepo)

			doc, err := svc.Upload(ctx, r, tt.originalFilename, tt.contentType, tt.size)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else if tt.wantErrMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, doc)
			}

			mStore.AssertExpectations(t)
			mRepo.AssertExpectations(t)
		})
	}
}

func TestDocumentService_List(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		limit      int
		offset     int
		setupMocks func(mRepo *repoMocks.MockDocumentRepository)
		wantErr    error
		checkRes   func(t *testing.T, res *DocumentListResult)
	}{
		{
			name:   "happy path",
			limit:  10,
			offset: 0,
			setupMocks: func(mRepo *repoMocks.MockDocumentRepository) {
				mRepo.On("List", ctx, repository.PageQuery{Limit: 10, Offset: 0}).
					Return(&repository.PageResult[model.Document]{
						Items: []model.Document{{ID: "1"}, {ID: "2"}},
						Total: 2,
					}, nil)
			},
			checkRes: func(t *testing.T, res *DocumentListResult) {
				assert.Equal(t, 2, len(res.Items))
				assert.Equal(t, 2, res.Total)
			},
		},
		{
			name:   "pagination boundary - zero limit uses default",
			limit:  0,
			offset: -1,
			setupMocks: func(mRepo *repoMocks.MockDocumentRepository) {
				mRepo.On("List", ctx, repository.PageQuery{Limit: 10, Offset: 0}).
					Return(&repository.PageResult[model.Document]{Items: []model.Document{}, Total: 0}, nil)
			},
		},
		{
			name:  "repository error",
			limit: 10,
			setupMocks: func(mRepo *repoMocks.MockDocumentRepository) {
				mRepo.On("List", ctx, mock.Anything).Return(nil, errors.New("db fail"))
			},
			wantErr: errors.New("db fail"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mRepo := new(repoMocks.MockDocumentRepository)
			svc := NewDocumentService(nil, mRepo)

			tt.setupMocks(mRepo)

			res, err := svc.List(ctx, tt.limit, tt.offset)

			if tt.wantErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkRes != nil {
					tt.checkRes(t, res)
				}
			}
			mRepo.AssertExpectations(t)
		})
	}
}

func TestDocumentService_Get(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		id         string
		setupMocks func(mRepo *repoMocks.MockDocumentRepository)
		wantErr    error
	}{
		{
			name: "happy path",
			id:   "valid-id",
			setupMocks: func(mRepo *repoMocks.MockDocumentRepository) {
				mRepo.On("FindByID", ctx, "valid-id").Return(&model.Document{ID: "valid-id"}, nil)
			},
		},
		{
			name:       "validation - empty id",
			id:         "",
			setupMocks: func(mRepo *repoMocks.MockDocumentRepository) {},
			wantErr:    ErrIDRequired,
		},
		{
			name: "not found - mapping sql.ErrNoRows",
			id:   "missing-id",
			setupMocks: func(mRepo *repoMocks.MockDocumentRepository) {
				mRepo.On("FindByID", ctx, "missing-id").Return(nil, sql.ErrNoRows)
			},
			wantErr: ErrNotFound,
		},
		{
			name: "generic repository error",
			id:   "error-id",
			setupMocks: func(mRepo *repoMocks.MockDocumentRepository) {
				mRepo.On("FindByID", ctx, "error-id").Return(nil, errors.New("db fail"))
			},
			wantErr: errors.New("db fail"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mRepo := new(repoMocks.MockDocumentRepository)
			svc := NewDocumentService(nil, mRepo)

			tt.setupMocks(mRepo)

			doc, err := svc.Get(ctx, tt.id)

			if tt.wantErr != nil {
				if errors.Is(tt.wantErr, ErrIDRequired) || errors.Is(tt.wantErr, ErrNotFound) {
					assert.ErrorIs(t, err, tt.wantErr)
				} else {
					assert.Error(t, err)
				}
				assert.Nil(t, doc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, doc)
				assert.Equal(t, tt.id, doc.ID)
			}
			mRepo.AssertExpectations(t)
		})
	}
}

func TestDocumentService_Delete(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		id         string
		setupMocks func(mStore *storeMocks.MockStorage, mRepo *repoMocks.MockDocumentRepository)
		wantErr    error
	}{
		{
			name: "happy path",
			id:   "valid-id",
			setupMocks: func(mStore *storeMocks.MockStorage, mRepo *repoMocks.MockDocumentRepository) {
				mRepo.On("FindByID", ctx, "valid-id").Return(&model.Document{ID: "valid-id", StoragePath: "path/to/obj"}, nil)
				mStore.On("Delete", ctx, "path/to/obj").Return(nil)
				mRepo.On("Delete", ctx, "valid-id").Return(nil)
			},
		},
		{
			name:       "validation - empty id",
			id:         "",
			setupMocks: func(mStore *storeMocks.MockStorage, mRepo *repoMocks.MockDocumentRepository) {},
			wantErr:    ErrIDRequired,
		},
		{
			name: "not found",
			id:   "missing-id",
			setupMocks: func(mStore *storeMocks.MockStorage, mRepo *repoMocks.MockDocumentRepository) {
				mRepo.On("FindByID", ctx, "missing-id").Return(nil, sql.ErrNoRows)
			},
			wantErr: ErrNotFound,
		},
		{
			name: "storage delete error",
			id:   "storage-fail-id",
			setupMocks: func(mStore *storeMocks.MockStorage, mRepo *repoMocks.MockDocumentRepository) {
				mRepo.On("FindByID", ctx, "storage-fail-id").Return(&model.Document{ID: "id", StoragePath: "path"}, nil)
				mStore.On("Delete", ctx, "path").Return(errors.New("storage fail"))
			},
			wantErr: errors.New("delete storage: storage fail"),
		},
		{
			name: "repository delete error",
			id:   "repo-fail-id",
			setupMocks: func(mStore *storeMocks.MockStorage, mRepo *repoMocks.MockDocumentRepository) {
				mRepo.On("FindByID", ctx, "repo-fail-id").Return(&model.Document{ID: "id", StoragePath: "path"}, nil)
				mStore.On("Delete", ctx, "path").Return(nil)
				mRepo.On("Delete", ctx, "repo-fail-id").Return(errors.New("db fail"))
			},
			wantErr: errors.New("db fail"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mStore := new(storeMocks.MockStorage)
			mRepo := new(repoMocks.MockDocumentRepository)
			svc := NewDocumentService(mStore, mRepo)

			tt.setupMocks(mStore, mRepo)

			err := svc.Delete(ctx, tt.id)

			if tt.wantErr != nil {
				if errors.Is(tt.wantErr, ErrIDRequired) || errors.Is(tt.wantErr, ErrNotFound) {
					assert.ErrorIs(t, err, tt.wantErr)
				} else {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.wantErr.Error())
				}
			} else {
				assert.NoError(t, err)
			}
			mStore.AssertExpectations(t)
			mRepo.AssertExpectations(t)
		})
	}
}
