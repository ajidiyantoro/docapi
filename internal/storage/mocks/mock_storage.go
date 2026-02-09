package mocks

import (
	"context"
	"io"
	"time"

	"docapi/internal/storage"

	"github.com/stretchr/testify/mock"
)

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Put(ctx context.Context, key string, r io.Reader, opt storage.PutObjectOptions) (storage.ObjectInfo, error) {
	args := m.Called(ctx, key, r, opt)
	if f, ok := args.Get(0).(func(context.Context, string, io.Reader, storage.PutObjectOptions) storage.ObjectInfo); ok {
		return f(ctx, key, r, opt), args.Error(1)
	}
	return args.Get(0).(storage.ObjectInfo), args.Error(1)
}

func (m *MockStorage) Get(ctx context.Context, key string) (io.ReadCloser, storage.ObjectInfo, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(io.ReadCloser), args.Get(1).(storage.ObjectInfo), args.Error(2)
}

func (m *MockStorage) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockStorage) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	// Note: time is not imported yet, I will fix it in next step if needed
	args := m.Called(ctx, key, expiry)
	return args.String(0), args.Error(1)
}
