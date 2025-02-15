package network

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/Mort4lis/memdb/internal/pkg/concurrency"
	"github.com/Mort4lis/memdb/internal/pkg/netutils"
)

type TCPHandler interface {
	Handle(ctx context.Context, req string) string
}

type TCPHandlerFunc func(ctx context.Context, req string) string

func (fn TCPHandlerFunc) Handle(ctx context.Context, req string) string {
	return fn(ctx, req)
}

type TCPServerConfig struct {
	addr           string
	maxConnections int
	maxMessageSize int
	idleTimeout    time.Duration
	writeTimeout   time.Duration
}

type TCPServerOption func(c *TCPServerConfig)

func WithServerListen(addr string) TCPServerOption {
	return func(c *TCPServerConfig) {
		c.addr = addr
	}
}

func WithServerMaxConnections(n int) TCPServerOption {
	return func(c *TCPServerConfig) {
		c.maxConnections = n
	}
}

func WithServerMaxMessageSize(n int) TCPServerOption {
	return func(c *TCPServerConfig) {
		c.maxMessageSize = n
	}
}

func WithServerIdleTimeout(d time.Duration) TCPServerOption {
	return func(c *TCPServerConfig) {
		c.idleTimeout = d
	}
}

func WithServerWriteTimeout(d time.Duration) TCPServerOption {
	return func(c *TCPServerConfig) {
		c.writeTimeout = d
	}
}

const (
	defaultListenAddr     = ":7991"
	defaultMaxConnections = 100
	defaultMaxMessageSize = 4096
)

type TCPServer struct {
	lis    net.Listener
	wg     *sync.WaitGroup
	sema   *concurrency.Semaphore
	logger *slog.Logger
	cancel func()
	conf   TCPServerConfig
}

func NewTCPServer(logger *slog.Logger, opts ...TCPServerOption) (*TCPServer, error) {
	conf := TCPServerConfig{
		addr:           defaultListenAddr,
		maxConnections: defaultMaxConnections,
		maxMessageSize: defaultMaxMessageSize,
	}
	for _, opt := range opts {
		opt(&conf)
	}

	lis, err := net.Listen("tcp", conf.addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", conf.addr, err)
	}

	return &TCPServer{
		lis:    lis,
		conf:   conf,
		logger: logger,
		wg:     &sync.WaitGroup{},
		sema:   concurrency.NewSemaphore(conf.maxConnections),
	}, nil
}

func (s *TCPServer) ListenPort() int {
	return s.lis.Addr().(*net.TCPAddr).Port //nolint:errcheck // ignore
}

func (s *TCPServer) ServeHandler(h TCPHandler) {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	go func() {
		for {
			select {
			case <-ctx.Done():
			default:
			}

			conn, err := s.lis.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				s.logger.Error("failed to accept connection", slog.Any("error", err))
				continue
			}

			s.wg.Add(1)
			s.sema.Acquire()
			go func() {
				defer func() {
					s.sema.Release()
					s.wg.Done()
				}()
				s.handleConnection(ctx, conn, h)
			}()
		}
	}()

	<-ctx.Done()
}

func (s *TCPServer) handleConnection(ctx context.Context, conn net.Conn, h TCPHandler) {
	logger := s.logger.With(slog.String("client_address", conn.RemoteAddr().String()))
	logger.Info("Connected client")

	defer func() {
		if err := recover(); err != nil {
			logger.Error("caught panic", slog.Any("panic", err))
		}
		if err := conn.Close(); err != nil {
			logger.Error("failed to close connection", slog.Any("error", err))
		}
		logger.Info("Disconnected client")
	}()

	buf := make([]byte, s.conf.maxMessageSize)
	for {
		var (
			n   int
			err error
		)

		err = concurrency.WithContextCheck(ctx, func() error {
			netutils.SetReadDeadline(conn, s.conf.idleTimeout)
			n, err = conn.Read(buf)
			return err //nolint:wrapcheck // ignore
		})
		if err != nil {
			if !errors.Is(err, io.EOF) {
				logger.Error("failed to read data", slog.Any("error", err))
			}
			return
		}
		if n == len(buf) {
			logger.Warn("max message size reached")
			return
		}

		var resp string
		err = concurrency.WithContextCheck(ctx, func() error {
			resp = h.Handle(ctx, string(buf[:n]))
			return nil
		})

		err = concurrency.WithContextCheck(ctx, func() error {
			netutils.SetWriteDeadline(conn, s.conf.writeTimeout)
			_, err = conn.Write([]byte(resp))
			return err //nolint:wrapcheck // ignore
		})
		if err != nil {
			logger.Error("failed to write data", slog.Any("error", err))
			return
		}
	}
}

func (s *TCPServer) Shutdown(ctx context.Context) error {
	// Close listener to prevent accepting new connections.
	err := s.lis.Close()
	if err != nil {
		return fmt.Errorf("close listener: %w", err)
	}

	// Notify active connections about shutdown.
	if s.cancel != nil {
		s.cancel()
	}

	doneCh := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(doneCh)
	}()

	// Wait for all connections to complete or for the shutdown context to be triggered.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-doneCh:
		return nil
	}
}
