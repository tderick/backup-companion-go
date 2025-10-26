package backup

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

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

	objectKey := filepath.Base(archivePath)
	for _, destName := range job.Destinations {
		if destConfig, ok := cfg.Destinations[destName]; ok {
			slog.Info("Uploading archive to destination", "objectKey", objectKey, "destination", destName, "provider", destConfig.Provider)

			s3Client, err := remotestorage.NewS3Client(ctx, destConfig)
			if err != nil {
				slog.Error("Failed to create S3 client", "destination", destName, "error", err)
				continue // Try next destination
			}

			if err := s3Client.UploadFile(ctx, archivePath, objectKey); err != nil {
				slog.Error("Failed to upload archive to destination", "objectKey", objectKey, "destination", destName, "error", err)
			} else {
				slog.Info("Successfully uploaded archive to destination", "objectKey", objectKey, "destination", destName)
			}
		} else {
			slog.Error("Destination referenced by job not found in config", "destination", destName, "jobName", jobName)
		}
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
