package database

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
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

// BackupDatabase dispatches the backup operation to the appropriate driver-specific function.
func BackupDatabase(ctx context.Context, db models.DatabaseConfig, backupDir string) {
	slog.Info("Performing backup for database", "db_name", db.Name, "driver", db.Driver, "backup_dir", backupDir)

	// Determine file extension based on driver
	var fileExtension string
	switch db.Driver {
	case "postgres":
		fileExtension = ".pgdump" // Custom binary format
	case "mysql":
		fileExtension = ".sql.gz" // mysqldump produces SQL, then gzip compresses it
	default:
		// This case should ideally be caught by validation, but as a fallback
		fileExtension = ".unknown.dump"
	}

	outputPath := filepath.Join(backupDir, fmt.Sprintf("%s_%s%s", db.Name, time.Now().Format("20060102150405"), fileExtension))

	var err error
	switch db.Driver {
	case "postgres":
		err = backupPostgres(ctx, db, outputPath)
	case "mysql":
		err = backupMysql(ctx, db, outputPath)
	default:
		err = fmt.Errorf("unsupported database driver for backup: %q", db.Driver)
	}

	if err != nil {
		slog.Error("Database backup failed", "db_name", db.Name, "driver", db.Driver, "error", err)
	} else {
		slog.Info("Database backup completed successfully", "db_name", db.Name, "driver", db.Driver, "path", outputPath)
	}
}

// backupPostgres performs a backup of a PostgreSQL database using pg_dump.
func backupPostgres(ctx context.Context, db models.DatabaseConfig, outputPath string) error {
	args := []string{
		"-h", db.Host,
		"-p", fmt.Sprintf("%d", db.Port),
		"-U", db.User,
		"-F", "c", // Custom format (compressed)
		"-b", // Include large objects
		"-v", // Verbose mode
		"-f", outputPath,
		db.Name,
	}

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", db.Password)) // Pass password securely via env

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error executing pg_dump for %q: %w\nOutput: %s", db.Name, err, string(output))
	}
	return nil
}

// backupMysql performs a backup of a MySQL database using mysqldump and pipes to gzip.
func backupMysql(ctx context.Context, db models.DatabaseConfig, outputPath string) error {
	args := []string{
		"-h", db.Host,
		fmt.Sprintf("-P%d", db.Port),
		fmt.Sprintf("-u%s", db.User),
		// --password or -p is usually omitted from args and handled by MYSQL_PWD env var for security
		"--single-transaction", // Essential for InnoDB consistency
		"--quick",              // Essential for large tables
		db.Name,
	}

	cmd := exec.CommandContext(ctx, "mysqldump", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("MYSQL_PWD=%s", db.Password)) // Pass password securely via env

	// Create the output file (e.g., .sql.gz)
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating output file %q for MySQL backup: %w", outputPath, err)
	}
	defer outFile.Close() // Ensure file is closed

	// Create a gzip writer to compress the mysqldump output
	gzipWriter := gzip.NewWriter(outFile)
	defer gzipWriter.Close() // Ensure gzip writer is closed and flushed

	// Pipe mysqldump's stdout directly to the gzip writer
	cmd.Stdout = gzipWriter

	// Capture stderr for logging potential mysqldump errors
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	slog.Debug("Executing mysqldump command", "command", cmd.String())

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error executing mysqldump for %q: %w\nStderr: %s", db.Name, err, stderr.String())
	}

	return nil
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
