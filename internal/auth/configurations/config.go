package configurations

import (
	"time"

	"go-boilerplate/internal/golibs/configs"
)

// Config of the auth service: configs/common.yaml + configs/auth.yaml.
type Config struct {
	configs.Base `mapstructure:",squash"`
	Tokens       Tokens `mapstructure:"tokens"`
}

type Tokens struct {
	AccessTTL  time.Duration `mapstructure:"access_ttl"`
	RefreshTTL time.Duration `mapstructure:"refresh_ttl"`
}
