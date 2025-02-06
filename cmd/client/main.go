package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/Mort4lis/memdb/internal/network"
)

const (
	defaultDialTimeout    = 10 * time.Second
	defaultReadTimeout    = 10 * time.Second
	defaultWriteTimeout   = 10 * time.Second
	defaultReadBufferSize = 4096
)

func main() {
	app := &cli.App{
		Name:      "memdb-cli",
		HelpName:  "memdb-cli",
		Usage:     "is a command line interface client for memdb",
		UsageText: "memdb-cli [OPTIONS]",
		Authors: []*cli.Author{
			{Name: "Pavel Korchagin", Email: "mortalis94@gmail.com"},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "address",
				Value: "127.0.0.1:7991",
				Usage: "Server address to connect",
			},
			&cli.DurationFlag{
				Name:  "dial-timeout",
				Value: defaultDialTimeout,
				Usage: "Timeout to establish connection with server",
			},
			&cli.DurationFlag{
				Name:  "read-timeout",
				Value: defaultReadTimeout,
				Usage: "Timeout for waiting response",
			},
			&cli.DurationFlag{
				Name:  "write-timeout",
				Value: defaultWriteTimeout,
				Usage: "Timeout for sending request",
			},
			&cli.IntFlag{
				Name:  "read-buffer-size",
				Value: defaultReadBufferSize,
				Usage: "Max buffer size for reading response",
			},
		},
		Action:               action,
		EnableBashCompletion: true,
	}

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func action(c *cli.Context) error {
	addr := c.String("address")
	opts := []network.TCPClientOption{
		network.WithClientDialTimeout(c.Duration("dial-timeout")),
		network.WithClientReadTimeout(c.Duration("read-timeout")),
		network.WithClientWriteTimeout(c.Duration("write-timeout")),
		network.WithClientReadBufferSize(c.Int("read-buffer-size")),
	}

	client, err := network.NewTCPClient(addr, opts...)
	if err != nil {
		return fmt.Errorf("init tcp client: %w", err)
	}
	defer client.Close()

	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		req := strings.TrimSpace(sc.Text())
		if req == "" {
			continue
		}

		var resp string
		resp, err = client.Send(req)
		if err != nil {
			return fmt.Errorf("send request: %w", err)
		}
		_, _ = fmt.Fprintln(os.Stdout, resp)
	}
	if sc.Err() != nil {
		return fmt.Errorf("scan error: %w", err)
	}
	return nil
}
