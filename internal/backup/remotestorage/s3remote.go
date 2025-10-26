package remotestorage

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

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

func UploadArchiveToDestinations(ctx context.Context, cfg *models.Config, job models.JobConfig, archivePath string) error {
	objectKey := filepath.Base(archivePath) // The name of the file in the S3 bucket

	var uploadErrors []error
	for _, destName := range job.Destinations {
		if destConfig, ok := cfg.Destinations[destName]; ok {
			slog.Info("Attempting to upload archive to destination",
				"archive_key", objectKey,
				"destination", destName,
				"provider", destConfig.Provider,
				"job_name", job.Output.Name,
			)

			s3Client, err := NewS3Client(ctx, destConfig)
			if err != nil {
				err := fmt.Errorf("failed to create S3 client for destination %q: %w", destName, err)
				slog.Error("Failed to create S3 client for upload, skipping destination",
					"destination", destName,
					"error", err,
					"job_name", job.Output.Name,
				)
				uploadErrors = append(uploadErrors, err)
				continue
			}

			if err := s3Client.UploadFile(ctx, archivePath, objectKey); err != nil {
				err := fmt.Errorf("failed to upload archive %q to destination %q: %w", objectKey, destName, err)
				slog.Error("Failed to upload archive to destination",
					"archive_key", objectKey,
					"destination", destName,
					"error", err,
					"job_name", job.Output.Name,
				)
				uploadErrors = append(uploadErrors, err)
			} else {
				slog.Info("Successfully uploaded archive to destination",
					"archive_key", objectKey,
					"destination", destName,
					"job_name", job.Output.Name,
				)
			}
		} else {
			err := fmt.Errorf("destination %q referenced by job %q not found in config during upload", destName, job.Output.Name)
			slog.Error("Destination not found in config during upload (should have been caught by earlier validation)",
				"destination", destName,
				"job_name", job.Output.Name,
				"error", err,
			)
			uploadErrors = append(uploadErrors, err)
		}
	}

	if len(uploadErrors) > 0 {
		return fmt.Errorf("encountered errors during archive upload: %v", uploadErrors)
	}
	return nil
}
