package models

type Config struct {
	Sources      SourcesConfig                `mapstructure:"sources"  validate:"required"`
	Destinations map[string]DestinationConfig `mapstructure:"destinations"  validate:"required"`
	Jobs         map[string]JobConfig         `mapstructure:"jobs"  validate:"required"`
}

type SourcesConfig struct {
	Databases   map[string]DatabaseConfig  `mapstructure:"databases"   validate:"required_without=Directories"`
	Directories map[string]DirectoryConfig `mapstructure:"directories" validate:"required_without=Databases"`
}

type DatabaseConfig struct {
	Driver   string `mapstructure:"driver" validate:"required,oneof=postgres mysql"`
	Host     string `mapstructure:"host"  validate:"required"`
	Port     int    `mapstructure:"port"  validate:"required"`
	User     string `mapstructure:"user"  validate:"required"`
	Password string `mapstructure:"password"  validate:"required"`
	Name     string `mapstructure:"name"  validate:"required"`
}

type DirectoryConfig struct {
	Path string `mapstructure:"path"  validate:"required,dir"`
}

type DestinationConfig struct {
	Provider        string `mapstructure:"provider"  validate:"required,oneof=s3 minio"`
	BucketName      string `mapstructure:"bucketName"  validate:"required"`
	AccessKeyID     string `mapstructure:"accessKeyId"  validate:"required"`
	SecretAccessKey string `mapstructure:"secretAccessKey"  validate:"required"`
	Region          string `mapstructure:"region" validate:"required_if=Provider s3"`
	EndpointURL     string `mapstructure:"endpointUrl" validate:"required_if=Provider minio,url"`
}

type OutputConfig struct {
	Dir  string `mapstructure:"dir"  validate:"required,dir"`
	Name string `mapstructure:"name"  validate:"required"`
}

type JobConfig struct {
	Output       OutputConfig `mapstructure:"output"  validate:"required"`
	Databases    []string     `mapstructure:"databases" validate:"required_without=Directories"`
	Directories  []string     `mapstructure:"directories" validate:"required_without=Databases"`
	Destinations []string     `mapstructure:"destinations" validate:"required,min=1"`
}
