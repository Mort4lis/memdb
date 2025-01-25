package logging

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/Mort4lis/memdb/internal/db/config"
)

const (
	JSONFormat = "json"
	TextFormat = "text"
)

const (
	DebugLevel = "debug"
	InfoLevel  = "info"
	WarnLevel  = "warn"
	ErrorLevel = "error"
)

func NewLoggerFromConfig(conf config.Logging) (*slog.Logger, error) {
	var (
		level   slog.Leveler
		handler slog.Handler
	)

	switch strings.ToLower(conf.Level) {
	case DebugLevel:
		level = slog.LevelDebug
	case InfoLevel:
		level = slog.LevelInfo
	case WarnLevel:
		level = slog.LevelWarn
	case ErrorLevel:
		level = slog.LevelError
	default:
		return nil, fmt.Errorf("unsupported logging level: %s", conf.Level)
	}

	opts := &slog.HandlerOptions{Level: level}

	switch strings.ToLower(conf.Format) {
	case JSONFormat:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case TextFormat:
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		return nil, fmt.Errorf("unsupported logging format: %s", conf.Format)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger, nil
}
