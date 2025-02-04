package db

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ilyakaznacheev/cleanenv"

	"github.com/Mort4lis/memdb/internal/db/compute"
	"github.com/Mort4lis/memdb/internal/db/config"
	"github.com/Mort4lis/memdb/internal/db/logging"
	"github.com/Mort4lis/memdb/internal/db/storage"
	"github.com/Mort4lis/memdb/internal/network"
)

func Run(confPath string) error {
	var conf config.Config
	if err := cleanenv.ReadConfig(confPath, &conf); err != nil {
		return fmt.Errorf("read config: %v", err)
	}

	logger, err := logging.NewLoggerFromConfig(conf.Logging)
	if err != nil {
		return fmt.Errorf("create logger: %v", err)
	}

	engine := storage.NewEngine()
	handler := compute.NewQueryHandler(logger, engine)
	server, err := network.NewTCPServer(logger, network.TCPServerConfig{
		Addr:           conf.Network.Addr,
		MaxConnections: conf.Network.MaxConnections,
		MaxMessageSize: conf.Network.MaxMessageSize,
		IdleTimeout:    conf.Network.IdleTimeout,
		WriteTimeout:   conf.Network.WriteTimeout,
	})
	if err != nil {
		return fmt.Errorf("create tcp server: %v", err)
	}

	go func() {
		logger.Info("Start to listen tcp server", slog.String("addr", conf.Network.Addr))
		server.ServeHandler(handler)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	logger.Info("Caught signal. Shutting down...", slog.String("signal", sig.String()))

	if err = server.Shutdown(); err != nil {
		logger.Error("Failed to shutdown tcp server", slog.Any("error", err))
	}
	return nil
}
