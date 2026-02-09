package mocks

import (
	"context"

	"docapi/internal/model"
	"docapi/internal/repository"
	"github.com/stretchr/testify/mock"
)

type MockDocumentRepository struct {
	mock.Mock
}

func (m *MockDocumentRepository) Create(ctx context.Context, doc *model.Document) (*model.Document, error) {
	args := m.Called(ctx, doc)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Document), args.Error(1)
}

func (m *MockDocumentRepository) FindByID(ctx context.Context, id string) (*model.Document, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Document), args.Error(1)
}

func (m *MockDocumentRepository) List(ctx context.Context, pq repository.PageQuery) (*repository.PageResult[model.Document], error) {
	args := m.Called(ctx, pq)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.PageResult[model.Document]), args.Error(1)
}

func (m *MockDocumentRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
