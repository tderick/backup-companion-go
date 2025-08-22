package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Sources      SourcesConfig                `mapstructure:"sources"`
	Destinations map[string]DestinationConfig `mapstructure:"destinations"`
	Jobs         map[string]JobConfig         `mapstructure:"jobs"`
}

type SourcesConfig struct {
	Databases   map[string]DatabaseConfig  `mapstructure:"databases"`
	Directories map[string]DirectoryConfig `mapstructure:"directories"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
}

type DirectoryConfig struct {
	Path string `mapstructure:"path"`
}

type DestinationConfig struct {
	Provider        string `mapstructure:"provider"`
	BucketName      string `mapstructure:"bucketName"`
	AccessKeyID     string `mapstructure:"accessKeyId"`
	SecretAccessKey string `mapstructure:"secretAccessKey"`
	Region          string `mapstructure:"region"`
	EndpointURL     string `mapstructure:"endpointUrl"`
}

type JobConfig struct {
	Output       string   `mapstructure:"output"`
	Databases    []string `mapstructure:"databases"`
	Directories  []string `mapstructure:"directories"`
	Destinations []string `mapstructure:"destinations"`
}

func LoadConfig(configFile string) (*Config, error) {
	v := viper.New()

	if configFile != "" {
		v.SetConfigFile(configFile) // use the explicit path provided by --config
	} else {
		v.SetConfigFile("config.yaml") // fallback to CWD/config.yaml
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Validate cross-references in jobs
	if err := validateReferences(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validateReferences ensures that each job only references existing databases,
// directories, and destinations defined in the config.
func validateReferences(cfg *Config) error {
	var b strings.Builder

	for jobName, job := range cfg.Jobs {
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
