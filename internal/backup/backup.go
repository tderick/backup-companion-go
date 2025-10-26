package backup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/tderick/backup-companion-go/internal/backup/database"
	"github.com/tderick/backup-companion-go/internal/backup/filesystem"
	"github.com/tderick/backup-companion-go/internal/backup/remotestorage"
	"github.com/tderick/backup-companion-go/internal/backup/util"
	"github.com/tderick/backup-companion-go/internal/models"
)

func Execute(ctx context.Context, cfg *models.Config) {
	for jobName, job := range cfg.Jobs {
		backupJob(ctx, cfg, jobName, job)
	}
}

func backupJob(ctx context.Context, cfg *models.Config, jobName string, job models.JobConfig) {
	slog.Info("Starting backup job", "jobName", jobName, "job", job)

	// Validate database sources for this job
	if err := validateJobDatabases(ctx, cfg, jobName, job); err != nil {
		slog.Error("Skipping backup job due to database source validation failures",
			"job_name", jobName,
			"error", err,
		)
		return
	}
	slog.Info("All database sources for job validated successfully", "job_name", jobName)

	// Validate remote destinations for this job
	if err := validateJobDestinations(ctx, cfg, jobName, job); err != nil {
		slog.Error("Skipping backup job due to destination validation failures",
			"job_name", jobName,
			"error", err,
		)
		return
	}
	slog.Info("All remote destinations for job validated successfully", "job_name", jobName)

	// Create a temporary directory for this job's backup artifacts
	backupDir, err := util.CreateBackupDir(job.Output)
	if err != nil {
		slog.Error("Failed to create a backup directory", "jobName", jobName, "error", err)
		return
	}

	archivePath := backupDir + ".tar.gz"

	defer func() {
		if err := os.RemoveAll(backupDir); err != nil {
			slog.Error("Failed to cleanup temporary backup directory", "backupDir", backupDir, "jobName", jobName, "error", err)
		} else {
			slog.Info("Cleaned up temporary backup directory", "backupDir", backupDir, "jobName", jobName)
		}
		if err := os.Remove(archivePath); err != nil {
			slog.Error("Failed to cleanup archive file", "archivePath", archivePath, "jobName", jobName, "error", err)
		} else {
			slog.Info("Cleaned up archive file", "archivePath", archivePath, "jobName", jobName)
		}
	}()

	// Determine job type and call appropriate handlers
	switch getJobType(job) {
	case "files-only":
		filesystem.BackupFilesOnly(ctx, cfg, job, backupDir)
	case "databases-only":
		database.BackupDatabasesOnly(ctx, cfg, job, backupDir)
	case "both":
		filesystem.BackupFilesOnly(ctx, cfg, job, backupDir)
		database.BackupDatabasesOnly(ctx, cfg, job, backupDir)
	}

	if err := util.CreateTarGz(backupDir, archivePath); err != nil {
		slog.Error("Failed to create archive", "jobName", jobName, "error", err)
		return
	}
	slog.Info("Successfully created archive", "jobName", jobName, "archivePath", archivePath)

	// Call the new function to upload the archive to destinations
	if err := remotestorage.UploadArchiveToDestinations(ctx, cfg, job, archivePath); err != nil {
		slog.Error("Failed to upload archive to one or more destinations",
			"job_name", jobName,
			"archive_path", archivePath,
			"error", err,
		)
	} else {
		slog.Info("Archive successfully uploaded to all destinations for job",
			"job_name", jobName,
			"archive_path", archivePath,
		)
	}

}

func getJobType(job models.JobConfig) string {
	hasFiles := len(job.Directories) > 0
	hasDatabases := len(job.Databases) > 0

	if hasFiles && !hasDatabases {
		return "files-only"
	}
	if !hasFiles && hasDatabases {
		return "databases-only"
	}
	return "both" // If both are empty, earlier validation should have caught it.
}

// validateJobDatabases validates all database sources referenced by a job.
func validateJobDatabases(ctx context.Context, cfg *models.Config, jobName string, job models.JobConfig) error {
	var validationErrors []string
	for _, dbName := range job.Databases {
		if dbConfig, ok := cfg.Sources.Databases[dbName]; ok {
			slog.Debug("Attempting to validate database connection", "database", dbName)
			if err := database.ValidateConnection(ctx, dbConfig); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("database %q failed connection validation: %v", dbName, err))
			} else {
				slog.Info("Database source validated successfully", "database", dbName)
			}
		} else {
			validationErrors = append(validationErrors, fmt.Sprintf("database %q referenced by job %q not found in sources", dbName, jobName))
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf(strings.Join(validationErrors, "; "))
	}
	return nil
}

// validateJobDestinations validates all remote S3 destinations referenced by a job.
func validateJobDestinations(ctx context.Context, cfg *models.Config, jobName string, job models.JobConfig) error {
	var validationErrors []string
	for _, destName := range job.Destinations {
		if destConfig, ok := cfg.Destinations[destName]; ok {
			slog.Debug("Attempting to create S3 client for destination", "destination", destName)
			s3Client, err := remotestorage.NewS3Client(ctx, destConfig)
			if err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("failed to create S3 client for destination %q: %v", destName, err))
				continue
			}
			slog.Debug("Attempting to validate connection for destination", "destination", destName)
			if err := s3Client.ValidateConnection(ctx); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("destination %q failed connection validation: %v", destName, err))
			} else {
				slog.Info("Destination validated successfully", "destination", destName)
			}
		} else {
			validationErrors = append(validationErrors, fmt.Sprintf("destination %q referenced by job %q not found in config", destName, jobName))
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf(strings.Join(validationErrors, "; "))
	}
	return nil
}
