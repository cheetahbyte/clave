package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Config configures the S3 (or S3-compatible) backend. Endpoint and
// UsePathStyle allow targeting MinIO, Cloudflare R2, Backblaze B2, etc.
type S3Config struct {
	Bucket          string `json:"bucket"`
	Region          string `json:"region"`
	Endpoint        string `json:"endpoint"`          // optional custom endpoint
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Prefix          string `json:"prefix"`            // optional key prefix
	UsePathStyle    bool   `json:"use_path_style"`    // required by MinIO and some others
}

type s3Backend struct {
	client   *s3.Client
	uploader *manager.Uploader
	bucket   string
	prefix   string
}

// NewS3 builds an S3-backed Backend from cfg.
func NewS3(cfg S3Config) (Backend, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3 storage: bucket is required")
	}
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, fmt.Errorf("s3 storage: access_key_id and secret_access_key are required")
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	opts := s3.Options{
		Region:       cfg.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		UsePathStyle: cfg.UsePathStyle,
	}
	if cfg.Endpoint != "" {
		opts.BaseEndpoint = aws.String(cfg.Endpoint)
	}

	client := s3.New(opts)
	return &s3Backend{
		client:   client,
		uploader: manager.NewUploader(client),
		bucket:   cfg.Bucket,
		prefix:   strings.Trim(cfg.Prefix, "/"),
	}, nil
}

func newS3FromConfig(raw json.RawMessage) (Backend, error) {
	cfg := S3Config{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse s3 storage config: %w", err)
	}
	return NewS3(cfg)
}

func (s *s3Backend) objectKey(key string) string {
	key = strings.TrimLeft(key, "/")
	if s.prefix == "" {
		return key
	}
	return s.prefix + "/" + key
}

// countingReader tracks bytes read so Put can report the stored size even
// though the multipart uploader does not return it.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

func (s *s3Backend) Put(ctx context.Context, key string, r io.Reader) (int64, error) {
	cr := &countingReader{r: r}
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.objectKey(key)),
		Body:   cr,
	})
	if err != nil {
		return 0, fmt.Errorf("s3 upload: %w", err)
	}
	return cr.n, nil
}

func (s *s3Backend) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.objectKey(key)),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 get: %w", err)
	}
	return out.Body, nil
}

func (s *s3Backend) Verify(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return fmt.Errorf("s3 connection failed: %w", err)
	}
	return nil
}

func (s *s3Backend) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.objectKey(key)),
	})
	if err != nil {
		return fmt.Errorf("s3 delete: %w", err)
	}
	return nil
}
