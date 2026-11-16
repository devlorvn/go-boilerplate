package configurations

import (
	"time"

	"go-boilerplate/internal/golibs/configs"
)

// Config of the file service: configs/common.yaml + configs/file.yaml.
type Config struct {
	configs.Base `mapstructure:",squash"`
	Storage      Storage `mapstructure:"storage"`
	Cleanup      Cleanup `mapstructure:"cleanup"`
}

type Storage struct {
	Region          string `mapstructure:"region"`
	Bucket          string `mapstructure:"bucket"`
	Endpoint        string `mapstructure:"endpoint"` // set for MinIO/LocalStack, empty for real S3
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
}

type Cleanup struct {
	Interval time.Duration `mapstructure:"interval"`
}
