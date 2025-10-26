package config

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
	"github.com/tderick/backup-companion-go/internal/models"
)

func LoadConfig(configFile string) (*models.Config, error) {
	v := viper.New()

	if configFile != "" {
		v.SetConfigFile(configFile) // use the explicit path provided by --config
	} else {
		v.SetConfigFile("config.yaml") // fallback to CWD/config.yaml
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg models.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Validate cross-references in jobs
	if err := validateReferences(&cfg); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		// This part is for printing user-friendly error messages.
		validationErrors := err.(validator.ValidationErrors)

		slog.Error("Configuration validation failed with the following errors:")

		for _, fieldErr := range validationErrors {
			slog.Error("Config field invalid", // Use slog.Error with structured keys
				"field_namespace", fieldErr.Namespace(),
				"validation_rule", fieldErr.Tag(),
				"error", fieldErr.Error(), // Include original validation error
			)
		}
		// Exit the program because the config is invalid.
		return nil, fmt.Errorf("please correct the errors in your configuration file and try again")
	}
	slog.Info("Configuration file loaded and validated successfully.")

	return &cfg, nil
}

// validateReferences ensures that each job only references existing databases,
// directories, and destinations defined in the config.
func validateReferences(cfg *models.Config) error {
	var b strings.Builder

	for jobName, job := range cfg.Jobs {
		// Validate output is provided
		if job.Output.Dir == "" || job.Output.Name == "" {
			fmt.Fprintf(&b, "job %q requires an output dir/name\n", jobName)
		}

		// Validate at least one database or directory is specified
		if len(job.Databases) == 0 && len(job.Directories) == 0 {
			fmt.Fprintf(&b, "job %q requires at least one database or directory\n", jobName)
		}

		// Validate at least one destination is specified
		if len(job.Destinations) == 0 {
			fmt.Fprintf(&b, "job %q requires at least one destination\n", jobName)
		}

		// Databases
		for _, db := range job.Databases {
			if _, ok := cfg.Sources.Databases[db]; !ok {
				fmt.Fprintf(&b, "job %q references unknown database %q\n", jobName, db)
			}
		}
		// Directories
		for _, dir := range job.Directories {
			if _, ok := cfg.Sources.Directories[dir]; !ok {
				fmt.Fprintf(&b, "job %q references unknown directory %q\n", jobName, dir)
			}
		}
		// Destinations
		for _, dst := range job.Destinations {
			if _, ok := cfg.Destinations[dst]; !ok {
				fmt.Fprintf(&b, "job %q references unknown destination %q\n", jobName, dst)
			}
		}
	}

	if b.Len() > 0 {
		return fmt.Errorf("invalid configuration:\n%s", b.String())
	}
	return nil
}
