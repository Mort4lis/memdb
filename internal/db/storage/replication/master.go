package replication

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/protobuf/proto"

	"github.com/Mort4lis/memdb/internal/db/storage/replication/contract"
	"github.com/Mort4lis/memdb/internal/network"
	"github.com/Mort4lis/memdb/internal/pkg/concurrency"
)

//go:generate protoc --go_out=. --go_opt=paths=source_relative contract/replication.proto

type SegmentDirectory interface {
	ContentByName(name string) ([]byte, error)
	NextRotatedSegmentName(from string) (string, error)
	LastSegmentName() (string, error)
	Save(name string, data []byte) error
}

type Master struct {
	logger *slog.Logger
	srv    *network.TCPServer
	sd     SegmentDirectory
}

func NewMaster(logger *slog.Logger, sd SegmentDirectory, opts ...network.TCPServerOption) (*Master, error) {
	logger = logger.With(slog.String("component", "replication.master"))

	srv, err := network.NewTCPServer(logger, opts...)
	if err != nil {
		return nil, fmt.Errorf("create tcp server: %w", err)
	}

	m := &Master{logger: logger, srv: srv, sd: sd}
	go func() {
		logger.Info("Start to listen replication master tcp server", slog.String("addr", srv.ListenAddr()))
		srv.ServeHandler(network.TCPHandlerFunc(m.handler))
	}()

	return m, nil
}

func (m *Master) IsSlave() bool {
	return false
}

func (m *Master) handler(ctx context.Context, b []byte) []byte {
	var req contract.NextSegmentRequest
	if err := proto.Unmarshal(b, &req); err != nil {
		m.logger.Error("Failed to unmarshal request", slog.Any("error", err))
		return contract.RespondError(contract.ErrInternal)
	}

	m.logger.Debug("Received replication request to get next rotated segment",
		slog.String("last_segment_name", req.LastSegmentName),
	)

	var (
		nextRotatedName string
		err             error
	)
	err = concurrency.WithContextCheck(ctx, func() error {
		nextRotatedName, err = m.sd.NextRotatedSegmentName(req.LastSegmentName)
		return err //nolint:wrapcheck // ignore
	})
	if err != nil {
		m.logger.Error("Failed to get next rotated segment name",
			slog.Any("error", err),
			slog.String("last_segment_name", req.LastSegmentName),
		)
		return contract.RespondError(contract.ErrInternal)
	}

	if nextRotatedName == "" {
		m.logger.Debug("New rotated segment is not found",
			slog.String("last_segment_name", req.LastSegmentName),
		)
		return contract.RespondError(contract.ErrNotFound)
	}

	var data []byte
	err = concurrency.WithContextCheck(ctx, func() error {
		data, err = m.sd.ContentByName(nextRotatedName)
		return err //nolint:wrapcheck // ignore
	})
	if err != nil {
		m.logger.Error("Failed to next rotated segment",
			slog.Any("error", err),
			slog.String("segment_name", nextRotatedName),
		)
		return contract.RespondError(contract.ErrInternal)
	}

	respBytes, err := proto.Marshal(&contract.NextSegmentResponse{
		SegmentName: nextRotatedName,
		Data:        data,
	})
	if err != nil {
		m.logger.Error("Failed to marshal response", slog.Any("error", err))
		return contract.RespondError(contract.ErrInternal)
	}
	return respBytes
}

func (m *Master) Shutdown(ctx context.Context) error {
	if m.srv == nil {
		return nil
	}
	if err := m.srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown tcp server: %w", err)
	}
	return nil
}
