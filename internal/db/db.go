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
	_ = handler

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	logger.Info("Caught signal. Shutting down...", slog.String("signal", sig.String()))

	return nil
}
