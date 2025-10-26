package backup

import (
	"context"
	"fmt"
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
	//fmt.Printf("Starting backup job %q: %+v\n", jobName, job)

	// Create a temporary directory for this job's backup artifacts
	backupDir, err := util.CreateBackupDir(job.Output)
	if err != nil {
		fmt.Printf("Failed to create a backup directory for job %q: %v\n", jobName, err)
		return
	}

	defer func() {
		if err := os.RemoveAll(backupDir); err != nil {
			fmt.Printf("Failed to cleanup temporary backup directory %q for job %q: %v\n", backupDir, jobName, err)
		} else {
			fmt.Printf("Cleaned up temporary backup directory %q for job %q.\n", backupDir, jobName)
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

	archivePath := backupDir + ".tar.gz"

	if err := util.CreateTarGz(backupDir, archivePath); err != nil {
		fmt.Printf("Failed to create archive for job %q: %v\n", jobName, err)
		return
	}
	fmt.Printf("Successfully created archive for job %q at %q\n", jobName, archivePath)

	objectKey := filepath.Base(archivePath)

	for _, destName := range job.Destinations {
		if destConfig, ok := cfg.Destinations[destName]; ok {
			fmt.Printf("Uploading archive %q to destination %q (%s)...\n", objectKey, destName, destConfig.Provider)

			s3Client, err := remotestorage.NewS3Client(ctx, destConfig)
			if err != nil {
				fmt.Printf("Failed to create S3 client for destination %q: %v\n", destName, err)
				continue // Try next destination
			}

			if err := s3Client.UploadFile(ctx, archivePath, objectKey); err != nil {
				fmt.Printf("Failed to upload archive %q to destination %q: %v\n", objectKey, destName, err)
			} else {
				fmt.Printf("Successfully uploaded archive %q to destination %q.\n", objectKey, destName)
			}
		} else {
			fmt.Printf("Error: Destination %q referenced by job %q not found in config.\n", destName, jobName)
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
