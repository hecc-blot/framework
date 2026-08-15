package server

import (
	envType "github.com/hecc-blot/hecc-blot-core/enum/env"
)

type Config struct {
	Env           envType.Value `mapstructure:"env"`
	Port          string        `mapstructure:"port"`
	EnableTrace   bool          `mapstructure:"enable_trace"`
	ReadTimeout   int           `mapstructure:"read_timeout"`
	WriteTimeout  int           `mapstructure:"write_timeout"`
	IdleTimeout   int           `mapstructure:"idle_timeout"`
	BodySizeLimit int64         `mapstructure:"body_size_limit"`
}
