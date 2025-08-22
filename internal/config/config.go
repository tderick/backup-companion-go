package config

import (
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
	return &cfg, nil
}
