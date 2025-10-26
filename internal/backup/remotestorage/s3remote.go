package remotestorage

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tderick/backup-companion-go/internal/models"
)

type S3Client struct {
	client     *s3.Client
	bucketName string
}

func NewS3Client(ctx context.Context, cfg models.DestinationConfig) (*S3Client, error) {
	// Load the base AWS config with region and credentials.
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load base AWS config: %w", err)
	}

	// Create the S3 service client using the base AWS config.
	// We apply S3-specific options directly to the client here.
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		// If a custom EndpointURL is provided, set it as the BaseEndpoint.
		// This is the modern and preferred way to specify a custom endpoint.
		if cfg.EndpointURL != "" {
			o.BaseEndpoint = aws.String(cfg.EndpointURL)
			// For S3-compatible storage, path-style addressing is often required.
			o.UsePathStyle = true
		}
	})

	s3Client := &S3Client{
		client:     client,
		bucketName: cfg.BucketName,
	}
	
	return s3Client, nil
}

func (c *S3Client) UploadFile(ctx context.Context, filePath, objectKey string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", filePath, err)
	}
	defer file.Close()

	_, err = c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(objectKey),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file %q to bucket %q with key %q: %w", filePath, c.bucketName, objectKey, err)
	}

	return nil
}

func (c *S3Client) ValidateConnection(ctx context.Context) error {
	// HeadBucket is a lightweight, non-destructive way to check for bucket existence
	// and access permissions. It's an ideal choice for validating the connection.
	_, err := c.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.bucketName),
	})

	if err != nil {
		// Provide a more descriptive error message that includes the original error.
		return fmt.Errorf("failed to validate S3 connection for bucket %q: %w", c.bucketName, err)
	}

	// If no error is returned, the connection and access to the bucket are considered valid.
	return nil
}
