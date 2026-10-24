package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type R2Storage struct {
	client  *s3.Client
	bucket  string
	baseURL string
}

func NewR2Storage(endpoint, accessKeyID, secretAccessKey, bucket, baseURL string) (*R2Storage, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               endpoint,
			SigningRegion:     "auto",
			HostnameImmutable: true,
		}, nil
	})

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithEndpointResolverWithOptions(customResolver),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		awsconfig.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load R2 AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &R2Storage{
		client:  client,
		bucket:  bucket,
		baseURL: strings.TrimRight(baseURL, "/"),
	}, nil
}

func (s *R2Storage) Save(filename string, r io.Reader) (string, int64, string, error) {
	ext := filepath.Ext(filename)
	safeExt := strings.ToLower(strings.TrimPrefix(ext, "."))
	uniqueID := uuid.New().String()
	key := fmt.Sprintf("%s.%s", uniqueID, safeExt)

	// Read first 512 bytes for content-type detection
	buf := make([]byte, 512)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return "", 0, "", fmt.Errorf("failed to read file header: %w", err)
	}

	mimeType := http.DetectContentType(buf[:n])

	// Read the rest of the stream into memory or temporary buffer
	var fullBody bytes.Buffer
	if n > 0 {
		fullBody.Write(buf[:n])
	}
	copied, err := io.Copy(&fullBody, r)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to read full payload: %w", err)
	}
	totalSize := int64(n) + copied

	_, putErr := s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(fullBody.Bytes()),
		ContentType:   aws.String(mimeType),
		ContentLength: aws.Int64(totalSize),
	})
	if putErr != nil {
		return "", 0, "", fmt.Errorf("failed to upload object to R2: %w", putErr)
	}

	return key, totalSize, mimeType, nil
}

func (s *R2Storage) Open(storedPath string) (io.ReadCloser, error) {
	key := filepath.Base(storedPath)
	output, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from R2: %w", err)
	}
	return output.Body, nil
}

func (s *R2Storage) Delete(storedPath string) error {
	key := filepath.Base(storedPath)
	_, err := s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (s *R2Storage) GetURL(storedPath string) string {
	cleanName := filepath.Base(storedPath)
	return fmt.Sprintf("%s/api/v1/documents/files/%s", s.baseURL, cleanName)
}
