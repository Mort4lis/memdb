package netutils

import (
	"log/slog"
	"net"
	"time"
)

func SetReadDeadline(conn net.Conn, timeout time.Duration) {
	if timeout == 0 {
		return
	}
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		slog.Error("failed to set read deadline", slog.Any("error", err)) //nolint:sloglint // ignore global logger
	}
}

func SetWriteDeadline(conn net.Conn, timeout time.Duration) {
	if timeout == 0 {
		return
	}
	if err := conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		slog.Error("failed to set write deadline", slog.Any("error", err)) //nolint:sloglint // ignore global logger
	}
}
