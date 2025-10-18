package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type StorageService struct {
	client     *s3.Client
	bucketName string
	endpoint   string
}

// NewStorageService creates a new storage service using Supabase's S3-compatible endpoint
func NewStorageService(endpoint, region, accessKey, secretKey, bucketName string) (*StorageService, error) {
	// Create custom resolver for Supabase endpoint
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				URL:               endpoint,
				HostnameImmutable: true,
				SigningRegion:     region,
			}, nil
		}
		return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
	})

	// Load AWS configuration with custom endpoint
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // Use path-style addressing for Supabase
	})

	return &StorageService{
		client:     client,
		bucketName: bucketName,
		endpoint:   endpoint,
	}, nil
}

// UploadFile uploads a file to Supabase storage
func (s *StorageService) UploadFile(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Generate public URL
	publicURL := fmt.Sprintf("%s/object/public/%s/%s", s.endpoint, s.bucketName, key)
	return publicURL, nil
}

// DownloadFile downloads a file from Supabase storage
func (s *StorageService) DownloadFile(ctx context.Context, key string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	result, err := s.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	return data, nil
}

// DeleteFile deletes a file from Supabase storage
func (s *StorageService) DeleteFile(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetPublicURL returns the public URL for a file
func (s *StorageService) GetPublicURL(key string) string {
	return fmt.Sprintf("%s/object/public/%s/%s", s.endpoint, s.bucketName, key)
}

// GenerateAudioKey generates a storage key for an audio file
func GenerateAudioKey(articleID int) string {
	return filepath.Join("audio", fmt.Sprintf("article_%d.mp3", articleID))
}