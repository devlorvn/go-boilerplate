package configurations

import (
	"go-boilerplate/internal/gateway/app"
	"go-boilerplate/internal/golibs/configs"
)

// Config of the gateway: configs/common.yaml + configs/gateway.yaml.
type Config struct {
	configs.Base `mapstructure:",squash"`
	Upstreams    Upstreams     `mapstructure:"upstreams"`
	RateLimit    app.RateLimit `mapstructure:"rate_limit"`
}

type Upstreams struct {
	AuthAddr string `mapstructure:"auth_addr"`
	FileAddr string `mapstructure:"file_addr"`
}
