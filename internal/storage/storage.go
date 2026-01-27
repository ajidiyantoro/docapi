package storage

import (
	"context"
	"io"
	"time"
)

// Package storage contains file/object storage abstractions and utilities for object stores (S3-compatible).
// Implementations must avoid using local disk and rely on streaming I/O only.

// PutObjectOptions define optional parameters for uploading objects.
// Size should be the exact number of bytes if known; if unknown, set to -1 and the implementation
// will buffer/chunk as supported by the backend.
// ContentType and Metadata are optional.
 type PutObjectOptions struct {
	Size        int64
	ContentType string
	Metadata    map[string]string
}

// ObjectInfo contains basic information about an object in storage.
 type ObjectInfo struct {
	Key          string
	Size         int64
	ETag         string
	ContentType  string
	LastModified time.Time
	Metadata     map[string]string
}

// Storage is a reusable, S3-compatible object storage client interface.
// Methods use context and streaming readers/writers; no local disk is used.
 type Storage interface {
	// Put uploads an object under the given key using the provided reader and options.
	Put(ctx context.Context, key string, r io.Reader, opt PutObjectOptions) (ObjectInfo, error)
	// Get retrieves an object's content as a streaming reader alongside its info.
	Get(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error)
	// Delete removes an object by key.
	Delete(ctx context.Context, key string) error
	// PresignGet returns a time-limited URL that can be used to download the object without credentials.
	PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error)
}
