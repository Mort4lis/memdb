package network

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/Mort4lis/memdb/internal/pkg/netutils"
)

type TCPClientConfig struct {
	dialTimeout    time.Duration
	readTimeout    time.Duration
	writeTimeout   time.Duration
	readBufferSize int
}

type TCPClientOption func(o *TCPClientConfig)

func WithClientDialTimeout(d time.Duration) TCPClientOption {
	return func(c *TCPClientConfig) {
		c.dialTimeout = d
	}
}

func WithClientReadTimeout(d time.Duration) TCPClientOption {
	return func(c *TCPClientConfig) {
		c.readTimeout = d
	}
}

func WithClientWriteTimeout(d time.Duration) TCPClientOption {
	return func(c *TCPClientConfig) {
		c.writeTimeout = d
	}
}

func WithClientReadBufferSize(n int) TCPClientOption {
	return func(c *TCPClientConfig) {
		c.readBufferSize = n
	}
}

const (
	defaultReadBufferSize = 4096
)

type TCPClient struct {
	conn net.Conn
	conf TCPClientConfig
}

func NewTCPClient(addr string, opts ...TCPClientOption) (*TCPClient, error) {
	conf := TCPClientConfig{readBufferSize: defaultReadBufferSize}
	for _, opt := range opts {
		opt(&conf)
	}

	var (
		conn net.Conn
		err  error
	)

	if conf.dialTimeout != 0 {
		conn, err = net.DialTimeout("tcp", addr, conf.dialTimeout)
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return nil, fmt.Errorf("dial server: %w", err)
	}
	return &TCPClient{
		conn: conn,
		conf: conf,
	}, nil
}

func (c *TCPClient) Send(req string) (string, error) {
	netutils.SetWriteDeadline(c.conn, c.conf.writeTimeout)
	if _, err := c.conn.Write([]byte(req)); err != nil {
		return "", fmt.Errorf("write tcp socket: %w", err)
	}

	buf := make([]byte, c.conf.readBufferSize)

	netutils.SetReadDeadline(c.conn, c.conf.readTimeout)
	n, err := c.conn.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read tcp socket: %w", err)
	}
	if n == c.conf.readBufferSize {
		return "", errors.New("buffer is full")
	}
	return string(buf[:n]), nil
}

func (c *TCPClient) Close() error {
	if c.conn == nil {
		return nil
	}
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("close connection: %w", err)
	}
	return nil
}
