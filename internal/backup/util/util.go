package util

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/tderick/backup-companion-go/internal/models"
)

func CreateBackupDir(output models.OutputConfig) (string, error) {
	if _, err := os.Stat(output.Dir); os.IsNotExist(err) {
		if err := os.MkdirAll(output.Dir, 0755); err != nil {
			return "", fmt.Errorf("error creating output directory: %v", err)
		}
	}

	timestamp := time.Now().Format("2006-01-02-15-04-05")
	backupDir := filepath.Join(output.Dir, output.Name+"-"+timestamp)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("error creating backup directory: %v", err)
	}
	return backupDir, nil
}

func CreateTarGz(sourceDir, targetFile string) error {
	slog.Info("Creating archive", "sourceDir", sourceDir, "targetFile", targetFile)
	file, err := os.Create(targetFile)
	if err != nil {
		return fmt.Errorf("failed to create archive file %q: %v", targetFile, err)
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return fmt.Errorf("failed to create tar header for %q: %v", path, err)
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %q from %q: %v", path, sourceDir, err)
		}
		header.Name = relPath // Store relative path in archive

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %q: %v", path, err)
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %q: %v", path, err)
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return fmt.Errorf("failed to copy file contents from %q to archive: %v", path, err)
			}
		}
		return nil
	})
}
