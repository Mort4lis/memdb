package replication

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"github.com/Mort4lis/memdb/internal/db/compute"
	"github.com/Mort4lis/memdb/internal/db/storage/replication/contract"
	"github.com/Mort4lis/memdb/internal/db/storage/wal"
	"github.com/Mort4lis/memdb/internal/network"
	"github.com/Mort4lis/memdb/internal/pkg/concurrency"
)

type Slave struct {
	logger       *slog.Logger
	cli          *network.TCPClient
	sd           SegmentDirectory
	syncInterval time.Duration

	lastSegmentName string
	doneCh          chan struct{}
	cancel          func()
}

func NewSlave(
	logger *slog.Logger,
	sd SegmentDirectory,
	masterAddr string,
	syncInterval time.Duration,
	opts ...network.TCPClientOption,
) (*Slave, error) {
	cli, err := network.NewTCPClient(masterAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("create tcp client: %w", err)
	}

	lastSegmentName, err := sd.LastSegmentName()
	if err != nil {
		return nil, fmt.Errorf("get last segment name: %w", err)
	}

	return &Slave{
		cli:             cli,
		sd:              sd,
		syncInterval:    syncInterval,
		lastSegmentName: lastSegmentName,
		doneCh:          make(chan struct{}),
		logger:          logger.With(slog.String("component", "replication.slave")),
	}, nil
}

func (s *Slave) IsSlave() bool {
	return true
}

func (s *Slave) StartHandle(fn func(cid compute.CommandID, args []string) error) {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	go func() {
		ticker := time.NewTicker(s.syncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				close(s.doneCh)
				return
			case <-ticker.C:
				records, err := s.getNextRecords(ctx)
				if err != nil {
					s.logger.Error("Failed to get next records", slog.Any("error", err))
					continue
				}
				for _, r := range records {
					if err = fn(compute.CommandID(r.CommandId), r.Args); err != nil {
						s.logger.Error("Failed to handle record", slog.Any("error", err))
					}
				}
			}
		}
	}()
}

func (s *Slave) getNextRecords(ctx context.Context) ([]*wal.Record, error) {
	req := contract.NextSegmentRequest{
		LastSegmentName: s.lastSegmentName,
		RequestId:       uuid.New().String(),
	}
	reqBytes, err := proto.Marshal(&req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	var respBytes []byte
	err = concurrency.WithContextCheck(ctx, func() error {
		respBytes, err = s.cli.Send(reqBytes)
		return err //nolint:wrapcheck // ignore
	})
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	var resp contract.NextSegmentResponse
	if err = proto.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if resp.Error != nil {
		if resp.Error.Code == contract.ErrorCode_NOT_FOUND {
			s.logger.Debug("New rotated segment is not found",
				slog.String("last_segment_name", s.lastSegmentName),
			)
			return nil, nil
		}
		return nil, fmt.Errorf("get next segment: %w", resp.Error)
	}

	s.logger.Debug("Received next rotated segment from master, applying records...",
		slog.String("last_segment_name", s.lastSegmentName),
	)

	rs, decErr := wal.DecodeRecords(resp.Data)
	if decErr != nil {
		return nil, fmt.Errorf("decode records: %w", decErr)
	}
	if err = s.sd.Save(resp.SegmentName, resp.Data); err != nil {
		return nil, fmt.Errorf("save segment: %w", err)
	}

	s.lastSegmentName = resp.SegmentName
	return rs, nil
}

func (s *Slave) Shutdown(ctx context.Context) (err error) {
	if s.cancel == nil {
		return nil
	}
	defer func() {
		closeErr := s.cli.Close()
		if closeErr != nil && err == nil {
			err = fmt.Errorf("close tcp client: %w", closeErr)
		}
	}()

	s.cancel()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.doneCh:
		return nil
	}
}
