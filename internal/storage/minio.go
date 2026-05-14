package storage

import (
	"context"
	"fmt"
	"io"

	"stockmind/internal/common"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ObjectStore abstracts file storage operations.
type ObjectStore interface {
	Put(ctx context.Context, key string, reader io.Reader, size int64) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// MinIOStore implements ObjectStore using MinIO.
type MinIOStore struct {
	client *minio.Client
	bucket string
}

// NewMinIOStore creates a MinIO client and ensures the bucket exists.
func NewMinIOStore(ctx context.Context, cfg common.MinIO) (*MinIOStore, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.SSLMode,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: minio client init: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("storage: check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("storage: create bucket: %w", err)
		}
	}

	return &MinIOStore{client: client, bucket: cfg.Bucket}, nil
}

func (s *MinIOStore) Put(ctx context.Context, key string, reader io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("storage: put %s: %w", key, err)
	}
	return nil
}

func (s *MinIOStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("storage: get %s: %w", key, err)
	}
	return obj, nil
}

func (s *MinIOStore) Delete(ctx context.Context, key string) error {
	objectsCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    key,
		Recursive: true,
	})
	for obj := range objectsCh {
		if obj.Err != nil {
			return fmt.Errorf("storage: list prefix %s: %w", key, obj.Err)
		}
		if err := s.client.RemoveObject(ctx, s.bucket, obj.Key, minio.RemoveObjectOptions{}); err != nil {
			return fmt.Errorf("storage: delete %s: %w", obj.Key, err)
		}
	}
	return nil
}
