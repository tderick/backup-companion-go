package filesystem

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/tderick/backup-companion-go/internal/models"
)

func BackupFilesOnly(ctx context.Context, cfg *models.Config, job models.JobConfig, backupDir string) {
	for _, dirName := range job.Directories {
		if dirConfig, ok := cfg.Sources.Directories[dirName]; ok {
			BackupDirectory(ctx, dirConfig, backupDir)
		} else {
			// This case should ideally be caught by validateReferences
			slog.Error("Directory referenced by job not found in sources", "dirName", dirName, "job", job.Output)
		}
	}
}

// BackupDirectory recursively copies the contents of a source directory to the backup directory.
func BackupDirectory(ctx context.Context, dir models.DirectoryConfig, backupDir string) {
	slog.Info("Backing up directory", "dir", dir.Path, "path", backupDir)

	err := filepath.Walk(dir.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path to maintain directory structure
		relPath, err := filepath.Rel(dir.Path, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %q: %v", path, err)
		}

		targetPath := filepath.Join(backupDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		// Copy the file
		return efficientCopy(path, targetPath)
	})

	if err != nil {
		slog.Error("Error backing up directory", "dir", dir.Path, "error", err)
	}
}

// efficientCopy copies a file from src to dst using a buffer.
func efficientCopy(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %q: %v", src, err)
	}
	defer sourceFile.Close()

	// Create parent directories if they don't exist
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directories for %q: %v", dst, err)
	}

	destinationFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %q: %v", dst, err)
	}
	defer destinationFile.Close()

	buf := make([]byte, 1024*1024) // 1MB buffer for efficient copying
	_, err = io.CopyBuffer(destinationFile, sourceFile, buf)
	if err != nil {
		return fmt.Errorf("failed to copy file from %q to %q: %v", src, dst, err)
	}

	return nil
}
