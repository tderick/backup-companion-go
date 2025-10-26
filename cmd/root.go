package cmd

import (
	"context"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "backup-companion",
	Short: "A brief description of your application",
	Long:  `Backup Companion is a robust, production-ready Docker container that automates the backup of your databases (PostgreSQL, MySQL, MariaDB) and specified directories to any S3-compatible object storage provider.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { // Use PersistentPreRunE
		// Initialize logging before any command runs
		return initLogger(cmd.Context())
	},
}

// path to config file from --config
var cfgPath string

var logLevel string

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.backup-companion.yaml)")
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "", "path to config file (e.g. /etc/backup-companion/config.yaml)")
	// Add a persistent flag for log level
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Set the logging level (debug, info, warn, error)")
	// Bind the log-level flag to Viper
	viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initLogger sets up the global slog logger based on the configured log level.
func initLogger(ctx context.Context) error {
	level := new(slog.Level)
	switch strings.ToLower(viper.GetString("log.level")) {
	case "debug":
		*level = slog.LevelDebug
	case "info":
		*level = slog.LevelInfo
	case "warn":
		*level = slog.LevelWarn
	case "error":
		*level = slog.LevelError
	default:
		log.Printf("Invalid log level %q, defaulting to info.", viper.GetString("log.level"))
		*level = slog.LevelInfo
	}

	// For CLI applications, a TextHandler to Stderr is often suitable.
	// For production environments, a JSONHandler might be preferred for log aggregation.
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
		// AddSource: true, // Uncomment to include file/line number in logs
	})

	slog.SetDefault(slog.New(handler))

	slog.Debug("Logger initialized", "level", level.String())
	return nil
}
