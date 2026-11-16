package configurations

import "go-boilerplate/internal/golibs/configs"

// Config of the auth service: configs/common.yaml + configs/auth.yaml.
type Config struct {
	configs.Base `mapstructure:",squash"`
}
