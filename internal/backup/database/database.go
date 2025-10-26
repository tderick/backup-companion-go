package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/tderick/backup-companion-go/internal/models"
)

func BackupDatabasesOnly(ctx context.Context, cfg *models.Config, job models.JobConfig, backupDir string) {
	for _, dbName := range job.Databases {
		if dbConfig, ok := cfg.Sources.Databases[dbName]; ok {
			BackupDatabase(ctx, dbConfig, backupDir)
		} else {
			slog.Error("Database referenced by job not found in sources", "database_name", dbName, "job_name", job.Output.Name)
		}
	}
}

func BackupDatabase(ctx context.Context, db models.DatabaseConfig, backupDir string) {
	slog.Info("Performing backup for database", "db_name", db.Name, "driver", db.Driver, "backup_dir", backupDir)
	// TODO: Implement actual database backup logic here (e.g., calling pg_dump or mysqldump).
	// This might involve creating further sub-packages like `dbdump` or `postgres` / `mysql`.

	// Example: Create a dummy file for demonstration
	outputPath := filepath.Join(backupDir, fmt.Sprintf("%s_%s.sql", db.Name, time.Now().Format("20060102150405")))
	dummyContent := fmt.Sprintf("-- Database backup for %s from %s:%d\n", db.Name, db.Host, db.Port)
	if err := os.WriteFile(outputPath, []byte(dummyContent), 0644); err != nil {
		slog.Error("Error creating dummy database backup file", "db_name", db.Name, "error", err)
	} else {
		slog.Info("Dummy database backup created", "db_name", db.Name, "path", outputPath)
	}
}

// ValidateConnection checks if a database connection can be established.
func ValidateConnection(ctx context.Context, db models.DatabaseConfig) error {
	slog.Debug("Attempting to validate database connection", "driver", db.Driver, "host", db.Host, "port", db.Port, "db_name", db.Name)

	var dsn string
	switch db.Driver {
	case "postgres":
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			db.Host, db.Port, db.User, db.Password, db.Name)
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			db.User, db.Password, db.Host, db.Port, db.Name)
	default:
		return fmt.Errorf("unsupported database driver: %q", db.Driver)
	}

	dbConn, err := sql.Open(db.Driver, dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	defer dbConn.Close()

	// Set a short connection timeout for validation
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := dbConn.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database %q (%s:%d): %w", db.Name, db.Host, db.Port, err)
	}

	slog.Info("Successfully validated database connection", "db_name", db.Name, "driver", db.Driver)
	return nil
}
