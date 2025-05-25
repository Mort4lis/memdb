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

	var storeBuilder storage.Builder
	store, err := storeBuilder.Build(logger, &conf)
	if err != nil {
		return fmt.Errorf("build storage: %w", err)
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
