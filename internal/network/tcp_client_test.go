package network

import (
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTCPClient(t *testing.T) {
	const serverResponse = "hello, client"

	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer lis.Close()

	addr := lis.(*net.TCPListener).Addr().String() //nolint:errcheck // ignore

	errCh := make(chan error, 1)
	go func() {
		for {
			conn, err := lis.Accept()
			if err != nil {
				return
			}

			_, err = conn.Read(make([]byte, 512))
			if err != nil {
				errCh <- err
				return
			}
			_, err = conn.Write([]byte(serverResponse))
			if err != nil {
				errCh <- err
				return
			}
		}
	}()

	testCases := []struct {
		name       string
		clientFunc func(t *testing.T) *TCPClient
		wantErr    error
	}{
		{
			name: "Client with I/O timeout",
			clientFunc: func(t *testing.T) *TCPClient {
				cli, err := NewTCPClient(
					addr,
					WithClientReadTimeout(timeout),
					WithClientWriteTimeout(timeout),
				)
				require.NoError(t, err)
				return cli
			},
			wantErr: nil,
		},
		{
			name: "Client with dial timeout",
			clientFunc: func(t *testing.T) *TCPClient {
				cli, err := NewTCPClient(
					addr,
					WithClientDialTimeout(timeout),
				)
				require.NoError(t, err)
				return cli
			},
			wantErr: nil,
		},
		{
			name: "Client with small buffer size",
			clientFunc: func(t *testing.T) *TCPClient {
				cli, err := NewTCPClient(
					addr,
					WithClientReadBufferSize(5),
				)
				require.NoError(t, err)
				return cli
			},
			wantErr: errors.New("buffer is full"),
		},
		{
			name: "Client with connection error",
			clientFunc: func(t *testing.T) *TCPClient {
				cli, err := NewTCPClient(
					"172.31.255.255", // non-routable IP address for testing
					WithClientDialTimeout(timeout),
				)
				require.Error(t, err)
				return cli
			},
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cli := tc.clientFunc(t)
			if cli == nil {
				return
			}

			resp, err := cli.Send("hello, server")
			if tc.wantErr == nil {
				require.NoError(t, err)
				assert.Equal(t, serverResponse, resp)
			} else {
				require.Error(t, err)
				assert.Equal(t, tc.wantErr, err)
			}
			assert.NoError(t, cli.Close())
		})
	}
}
