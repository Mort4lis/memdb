package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ilyakaznacheev/cleanenv"

	"github.com/Mort4lis/memdb/internal/db/compute"
	"github.com/Mort4lis/memdb/internal/db/config"
	"github.com/Mort4lis/memdb/internal/db/logging"
	"github.com/Mort4lis/memdb/internal/db/storage"
	"github.com/Mort4lis/memdb/internal/db/storage/wal"
	"github.com/Mort4lis/memdb/internal/db/storage/wal/filesystem"
	"github.com/Mort4lis/memdb/internal/network"
)

const shutdownTimeout = 30 * time.Second

func Run(confPath string) error {
	var conf config.Config
	if err := cleanenv.ReadConfig(confPath, &conf); err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	logger, err := logging.NewLoggerFromConfig(conf.Logging)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	store, err := initStorage(&conf)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}

	handler := compute.NewQueryHandler(logger, store)
	server, err := network.NewTCPServer(logger, conf.Network.ServerOptions()...)
	if err != nil {
		return fmt.Errorf("create tcp server: %w", err)
	}

	go func() {
		logger.Info("Start to listen tcp server", slog.String("addr", conf.Network.Addr))
		server.ServeHandler(handler)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	logger.Info("Caught signal. Shutting down...", slog.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err = store.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown storage", slog.Any("error", err))
	}

	if err = server.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown tcp server", slog.Any("error", err))
	}
	return nil
}

func initStorage(conf *config.Config) (*storage.Storage, error) {
	var opts []storage.Option
	if conf.WAL != nil {
		w, err := initWAL(conf.WAL)
		if err != nil {
			return nil, fmt.Errorf("init WAL: %w", err)
		}
		opts = append(opts, storage.WithWAL(w))
	}

	store, err := storage.NewStorage(conf.Engine, opts...)
	if err != nil {
		return nil, fmt.Errorf("create storage: %w", err)
	}
	return store, nil
}

func initWAL(conf *config.WAL) (*wal.WAL, error) {
	segmentDir, err := filesystem.NewSegmentDirectory(conf.DataDir)
	if err != nil {
		return nil, fmt.Errorf("new segment directory: %w", err)
	}

	segment, err := filesystem.NewSegment(conf.DataDir, conf.MaxSegmentSize)
	if err != nil {
		return nil, fmt.Errorf("new segment: %w", err)
	}

	res, err := wal.NewWAL(segmentDir, segment, conf.FlushBatchSize, conf.FlushBatchInterval)
	if err != nil {
		return nil, fmt.Errorf("new WAL: %w", err)
	}
	return res, nil
}
