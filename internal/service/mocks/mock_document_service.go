package mocks

import (
	"context"
	"io"

	"docapi/internal/model"
	"docapi/internal/service"
	"github.com/stretchr/testify/mock"
)

type MockDocumentService struct {
	mock.Mock
}

func (m *MockDocumentService) Upload(ctx context.Context, r io.Reader, originalFilename string, contentType string, size int64) (*model.Document, error) {
	args := m.Called(ctx, r, originalFilename, contentType, size)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Document), args.Error(1)
}

func (m *MockDocumentService) List(ctx context.Context, limit, offset int) (*service.DocumentListResult, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.DocumentListResult), args.Error(1)
}

func (m *MockDocumentService) Get(ctx context.Context, id string) (*model.Document, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Document), args.Error(1)
}

func (m *MockDocumentService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
