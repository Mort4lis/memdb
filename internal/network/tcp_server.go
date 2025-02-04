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
)

type TCPHandler interface {
	Handle(ctx context.Context, req string) string
}

type TCPHandlerFunc func(ctx context.Context, req string) string

func (fn TCPHandlerFunc) Handle(ctx context.Context, req string) string {
	return fn(ctx, req)
}

type TCPServerConfig struct {
	Addr           string
	MaxConnections int
	MaxMessageSize int
	IdleTimeout    time.Duration
	WriteTimeout   time.Duration
}

type TCPServer struct {
	lis    net.Listener
	wg     *sync.WaitGroup
	sema   *concurrency.Semaphore
	logger *slog.Logger
	cancel func()
	conf   TCPServerConfig
}

func NewTCPServer(logger *slog.Logger, conf TCPServerConfig) (*TCPServer, error) {
	lis, err := net.Listen("tcp", conf.Addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", conf.Addr, err)
	}

	return &TCPServer{
		lis:    lis,
		conf:   conf,
		logger: logger,
		wg:     &sync.WaitGroup{},
		sema:   concurrency.NewSemaphore(conf.MaxConnections),
	}, nil
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
				s.logger.Error("failed to accept connection", slog.Any("error", err))
				continue
			}

			s.sema.Acquire()
			s.wg.Add(1)
			go func() {
				defer func() {
					s.wg.Done()
					s.sema.Release()
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

	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if err := recover(); err != nil {
			logger.Error("caught panic", slog.Any("panic", err))
		}
		if err := conn.Close(); err != nil {
			logger.Error("failed to close connection", slog.Any("error", err))
		}

		cancel()
		logger.Info("Disconnected client")
	}()

	buf := make([]byte, s.conf.MaxMessageSize)
	for {
		s.setReadDeadline(logger, conn)
		n, err := conn.Read(buf)
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

		resp := h.Handle(ctx, string(buf[:n]))

		s.setWriteDeadline(logger, conn)
		if _, err = conn.Write([]byte(resp)); err != nil {
			logger.Error("failed to write data", slog.Any("error", err))
			return
		}
	}
}

func (s *TCPServer) setReadDeadline(logger *slog.Logger, conn net.Conn) {
	if s.conf.IdleTimeout == 0 {
		return
	}

	if err := conn.SetReadDeadline(time.Now().Add(s.conf.IdleTimeout)); err != nil {
		logger.Error("failed to set read deadline", slog.Any("error", err))
	}
}

func (s *TCPServer) setWriteDeadline(logger *slog.Logger, conn net.Conn) {
	if s.conf.WriteTimeout == 0 {
		return
	}

	if err := conn.SetWriteDeadline(time.Now().Add(s.conf.WriteTimeout)); err != nil {
		logger.Error("failed to set write deadline", slog.Any("error", err))
	}
}

func (s *TCPServer) Shutdown() error {
	err := s.lis.Close()
	s.cancel()
	s.wg.Wait()

	if err != nil {
		return fmt.Errorf("close listener: %w", err)
	}
	return nil
}
