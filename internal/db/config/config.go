package config

import (
	"time"

	"github.com/Mort4lis/memdb/internal/network"
)

type Config struct {
	Engine  Engine  `yaml:"engine"`
	Network Network `yaml:"network"`
	Logging Logging `yaml:"logging"`
	WAL     *WAL    `yaml:"wal"`
}

type Engine struct {
	Type string `env-default:"in_memory" yaml:"type"`
}

type Network struct {
	Addr           string        `env-default:":7991"  yaml:"addr"`
	MaxConnections int           `env-default:"100"    yaml:"max_connections"`
	MaxMessageSize int           `env-default:"4096"   yaml:"max_message_size"`
	IdleTimeout    time.Duration `yaml:"idle_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
}

func (c Network) ServerOptions() []network.TCPServerOption {
	opts := []network.TCPServerOption{
		network.WithServerListen(c.Addr),
		network.WithServerMaxConnections(c.MaxConnections),
		network.WithServerMaxMessageSize(c.MaxMessageSize),
	}
	if c.IdleTimeout != 0 {
		opts = append(opts, network.WithServerIdleTimeout(c.IdleTimeout))
	}
	if c.WriteTimeout != 0 {
		opts = append(opts, network.WithServerWriteTimeout(c.WriteTimeout))
	}
	return opts
}

type Logging struct {
	Level  string `env-default:"info" yaml:"level"`
	Format string `env-default:"text" yaml:"format"`
}

type WAL struct {
	FlushBatchSize     int           `env-default:"100"             yaml:"flushing_batch_size"`
	FlushBatchInterval time.Duration `env-default:"10ms"            yaml:"flushing_batch_interval"`
	MaxSegmentSize     int           `env-default:"1048576"         yaml:"max_segment_size"`
	DataDir            string        `env-default:"/data/memdb/wal" yaml:"data_directory"`
}
