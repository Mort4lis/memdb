package network

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const timeout = 10 * time.Millisecond

type tcpServerTestFunc func(conn1, conn2 net.Conn)

var defaultHandlerFunc = TCPHandlerFunc(func(_ context.Context, req string) string {
	return req + "-response"
})

func TestTCPServer_ServeHandler_success(t *testing.T) {
	runTCPServerTest(
		t,
		defaultHandlerFunc,
		[]TCPServerOption{
			WithServerIdleTimeout(timeout),
			WithServerWriteTimeout(timeout),
		},
		func(conn1, conn2 net.Conn) {
			reqs := []string{"hello", "world"}

			respCh := make(chan string)
			errCh := make(chan error)
			go func() {
				for _, req := range reqs {
					resp, err := doRequest(conn2, "client_2-"+req)
					if err != nil {
						errCh <- err
						continue
					}
					respCh <- resp
				}
			}()

			for _, req := range reqs {
				resp, err := doRequest(conn1, "client_1-"+req)
				require.NoError(t, err)
				assert.Equal(t, "client_1-"+req+"-response", resp)

				select {
				case resp = <-respCh:
					assert.Equal(t, "client_2-"+req+"-response", resp)
				case err = <-errCh:
					require.NoError(t, err)
				}
			}
		},
	)
}

func TestTCPServer_ServeHandler_maxMessageSizeReached(t *testing.T) {
	const maxMessageSize = 512

	buf := make([]byte, maxMessageSize+1)
	_, err := rand.Read(buf)
	require.NoError(t, err)

	runTCPServerTest(
		t,
		defaultHandlerFunc,
		[]TCPServerOption{WithServerMaxMessageSize(maxMessageSize)},
		func(_, conn2 net.Conn) {
			setDeadline(t, conn2)
			_, err = conn2.Write(buf)
			require.NoError(t, err)

			_, err = conn2.Read(buf)
			require.Error(t, err)
		},
	)
}

func TestTCPServer_ServeHandler_maxConnectionsReached(t *testing.T) {
	const maxConnections = 1

	runTCPServerTest(
		t,
		defaultHandlerFunc,
		[]TCPServerOption{WithServerMaxConnections(maxConnections)},
		func(_, conn2 net.Conn) {
			setDeadline(t, conn2)
			_, err := conn2.Write([]byte("hello"))
			require.NoError(t, err)

			buf := make([]byte, 512)
			_, err = conn2.Read(buf)
			require.Error(t, err)
		},
	)
}

func runTCPServerTest(t *testing.T, h TCPHandler, opts []TCPServerOption, fn tcpServerTestFunc) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	opts = append(opts, WithServerListen(":0"))

	srv, err := NewTCPServer(logger, opts...)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	}()

	go srv.ServeHandler(h)
	time.Sleep(100 * time.Millisecond)

	conn1, err := net.Dial("tcp", fmt.Sprintf(":%d", srv.ListenPort()))
	require.NoError(t, err)
	defer conn1.Close()

	conn2, err := net.Dial("tcp", fmt.Sprintf(":%d", srv.ListenPort()))
	require.NoError(t, err)
	defer conn2.Close()

	fn(conn1, conn2)
}

func setDeadline(t *testing.T, conn net.Conn) {
	t.Helper()
	err := conn.SetDeadline(time.Now().Add(timeout))
	require.NoError(t, err)
}

func doRequest(conn net.Conn, req string) (string, error) {
	err := conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return "", err
	}
	_, err = conn.Write([]byte(req))
	if err != nil {
		return "", err
	}

	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}
