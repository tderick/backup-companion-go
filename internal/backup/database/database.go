package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tderick/backup-companion-go/internal/models"
)

func BackupDatabasesOnly(ctx context.Context, cfg *models.Config, job models.JobConfig, backupDir string) {
	for _, dbName := range job.Databases {
		if dbConfig, ok := cfg.Sources.Databases[dbName]; ok {
			BackupDatabase(ctx, dbConfig, backupDir) // Pass context and dbConfig
		} else {
			// This case should ideally be caught by validateReferences
			fmt.Printf("Error: Database %q referenced by job %q not found in sources\n", dbName, job.Output)
		}
	}
}

func BackupDatabase(ctx context.Context, db models.DatabaseConfig, backupDir string) {
	fmt.Printf("Performing backup for database: %s (Driver: %s) into %s\n", db.Name, db.Driver, backupDir)
	// TODO: Implement actual database backup logic here (e.g., calling pg_dump or mysqldump).
	// This might involve creating further sub-packages like `dbdump` or `postgres` / `mysql`.

	// Example: Create a dummy file for demonstration
	outputPath := filepath.Join(backupDir, fmt.Sprintf("%s_%s.sql", db.Name, time.Now().Format("20060102150405")))
	dummyContent := fmt.Sprintf("-- Database backup for %s from %s:%d\n", db.Name, db.Host, db.Port)
	if err := os.WriteFile(outputPath, []byte(dummyContent), 0644); err != nil {
		fmt.Printf("Error creating dummy database backup file for %s: %v\n", db.Name, err)
	} else {
		fmt.Printf("Dummy database backup for %s created at %s\n", db.Name, outputPath)
	}
}
