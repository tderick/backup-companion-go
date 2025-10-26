package filesystem

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/tderick/backup-companion-go/internal/models"
)

func BackupFilesOnly(ctx context.Context, cfg *models.Config, job models.JobConfig, backupDir string) {
	for _, dirName := range job.Directories {
		if dirConfig, ok := cfg.Sources.Directories[dirName]; ok {
			BackupDirectory(ctx, dirConfig, backupDir) // Pass context and dirConfig
		} else {
			// This case should ideally be caught by validateReferences
			fmt.Printf("Error: Directory %q referenced by job %q not found in sources\n", dirName, job.Output)
		}
	}
}

// BackupDirectory recursively copies the contents of a source directory to the backup directory.
func BackupDirectory(ctx context.Context, dir models.DirectoryConfig, backupDir string) {
	fmt.Printf("Backing up directory: %q to path: %q\n", dir.Path, backupDir)

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
		fmt.Printf("Error backing up directory %q: %v\n", dir.Path, err)
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
