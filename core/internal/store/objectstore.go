package store

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ObjectStore writes raw snapshots to S3-compatible storage (MinIO locally).
type ObjectStore struct {
	client *minio.Client
	bucket string
}

// ObjectStoreConfig configures the S3 client (sourced from env in callers).
type ObjectStoreConfig struct {
	Endpoint  string // host:port, no scheme
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	Region    string
}

// NewObjectStore connects to object storage and ensures the bucket exists.
func NewObjectStore(ctx context.Context, cfg ObjectStoreConfig) (*ObjectStore, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("bucket exists: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, fmt.Errorf("make bucket: %w", err)
		}
	}
	return &ObjectStore{client: client, bucket: cfg.Bucket}, nil
}

// RawKey is the object key for a raw snapshot's bytes (ARCHITECTURE.md Section 3).
func RawKey(jurisdiction, dataset, fiscalYear, sha256 string) string {
	return fmt.Sprintf("raw/%s/%s/%s/%s.bin", jurisdiction, dataset, fiscalYear, sha256)
}

// RawMetaKey is the object key for a raw snapshot's sidecar meta file.
func RawMetaKey(jurisdiction, dataset, fiscalYear, sha256 string) string {
	return fmt.Sprintf("raw/%s/%s/%s/%s.meta.json", jurisdiction, dataset, fiscalYear, sha256)
}

// Put writes bytes to a key (idempotent — content-addressed keys).
func (o *ObjectStore) Put(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := o.client.PutObject(ctx, o.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}
	return nil
}

// Get reads an object's bytes.
func (o *ObjectStore) Get(ctx context.Context, key string) ([]byte, error) {
	obj, err := o.client.GetObject(ctx, o.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", key, err)
	}
	defer obj.Close()
	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", key, err)
	}
	return data, nil
}

// ObjectStoreConfigFromEnv builds the S3 config from S3_* env vars (shared by the cmds).
func ObjectStoreConfigFromEnv() ObjectStoreConfig {
	endpoint := os.Getenv("S3_ENDPOINT")
	useSSL := strings.HasPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}
	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		bucket = "fiscal-raw"
	}
	region := os.Getenv("S3_REGION")
	if region == "" {
		region = "us-east-1"
	}
	return ObjectStoreConfig{
		Endpoint:  endpoint,
		AccessKey: os.Getenv("S3_ACCESS_KEY"),
		SecretKey: os.Getenv("S3_SECRET_KEY"),
		Bucket:    bucket,
		Region:    region,
		UseSSL:    useSSL,
	}
}

// Exists reports whether an object key is present.
func (o *ObjectStore) Exists(ctx context.Context, key string) (bool, error) {
	_, err := o.client.StatObject(ctx, o.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" || errResp.StatusCode == 404 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
