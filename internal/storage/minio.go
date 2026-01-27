package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"docapi/internal/config"
)

// minioStorage implements the Storage interface using an S3-compatible backend (MinIO, AWS S3, etc.).
// It is safe for concurrent use by multiple goroutines.
 type minioStorage struct {
	client *minio.Client
	bucket string
}

// NewMinIO creates a new S3-compatible storage client backed by MinIO.
// It validates connectivity and ensures the bucket exists (creates it if missing).
func NewMinIO(cfg config.MinIOConfig) (Storage, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("minio endpoint is required")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("minio credentials are required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("minio bucket is required")
	}

	cli, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	ms := &minioStorage{client: cli, bucket: cfg.Bucket}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ensure bucket exists.
	exists, err := cli.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket existence: %w", err)
	}
	if !exists {
		if err := cli.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
	}

	return ms, nil
}

// Put uploads an object using streaming I/O only (no local disk).
func (m *minioStorage) Put(ctx context.Context, key string, r io.Reader, opt PutObjectOptions) (ObjectInfo, error) {
	putOpts := minio.PutObjectOptions{
		ContentType: opt.ContentType,
		UserMetadata: opt.Metadata,
	}
	info, err := m.client.PutObject(ctx, m.bucket, key, r, opt.Size, putOpts)
	if err != nil {
		return ObjectInfo{}, err
	}
	return ObjectInfo{
		Key:          key,
		Size:         info.Size,
		ETag:         info.ETag,
		ContentType:  opt.ContentType,
		LastModified: time.Now(), // MinIO PutObjectInfo doesn't return LastModified
		Metadata:     opt.Metadata,
	}, nil
}

// Get downloads an object content as a ReadCloser along with basic info.
func (m *minioStorage) Get(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	obj, err := m.client.GetObject(ctx, m.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, ObjectInfo{}, err
	}
	// Fetch stat to populate info; avoid reading content into memory.
	st, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, ObjectInfo{}, err
	}
	info := ObjectInfo{
		Key:          key,
		Size:         st.Size,
		ETag:         st.ETag,
		ContentType:  st.ContentType,
		LastModified: st.LastModified,
		Metadata:     st.UserMetadata,
	}
	return obj, info, nil
}

// Delete removes an object by key.
func (m *minioStorage) Delete(ctx context.Context, key string) error {
	return m.client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
}

// PresignGet generates a pre-signed URL for GET with the specified expiry.
func (m *minioStorage) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	u, err := m.client.PresignedGetObject(ctx, m.bucket, key, expiry, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
