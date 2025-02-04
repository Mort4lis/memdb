package config

import (
	"time"
)

type Config struct {
	Engine  Engine  `yaml:"engine"`
	Network Network `yaml:"network"`
	Logging Logging `yaml:"logging"`
}

type Engine struct {
	Type string `env-default:"in_memory" yaml:"type"`
}

type Network struct {
	Addr           string        `env-default:":7991" yaml:"addr"`
	MaxConnections int           `env-default:"100"   yaml:"max_connections"`
	MaxMessageSize int           `env-default:"4096"  yaml:"max_message_size"`
	IdleTimeout    time.Duration `env-default:"2m"    yaml:"idle_timeout"`
	WriteTimeout   time.Duration `env-default:"15s"   yaml:"write_timeout"`
}

type Logging struct {
	Level  string `env-default:"info" yaml:"level"`
	Format string `env-default:"text" yaml:"format"`
}
