package compute

import (
	"context"
	"errors"
	"log/slog"

	dberrors "github.com/Mort4lis/memdb/internal/db/errors"
)

//go:generate mockery --inpackage --testonly --case underscore --name Storage
type Storage interface {
	Set(ctx context.Context, key, value string) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
}

type QueryHandler struct {
	logger *slog.Logger
	store  Storage
}

func NewQueryHandler(logger *slog.Logger, store Storage) *QueryHandler {
	return &QueryHandler{
		store:  store,
		logger: logger.With(slog.String("layer", "compute")),
	}
}

func (h *QueryHandler) Handle(ctx context.Context, req string) string {
	query, err := ParseQuery(req)
	if err != nil {
		h.logger.Warn("failed to parse query", slog.Any("error", err))
		return ParseQueryErrorResponse.WithErr(err).String()
	}

	switch query.cmdID {
	case SetCommandID:
		return h.handleSet(ctx, query)
	case GetCommandID:
		return h.handleGet(ctx, query)
	case DelCommandID:
		return h.handleDel(ctx, query)
	default:
		h.logger.Error(
			"handler is not configured for serving query",
			slog.String("command", query.cmdID.String()),
		)
		return InternalErrorResponse.WithErr(dberrors.ErrInternal).String()
	}
}

func (h *QueryHandler) handleSet(ctx context.Context, query Query) string {
	args := query.Args()
	if err := h.store.Set(ctx, args[0], args[1]); err != nil {
		h.logger.Error("failed to handle SET query", slog.Any("error", err))
		return InternalErrorResponse.WithErr(err).String()
	}
	return OKResponse.String()
}

func (h *QueryHandler) handleGet(ctx context.Context, query Query) string {
	args := query.Args()
	res, err := h.store.Get(ctx, args[0])
	if errors.Is(err, dberrors.ErrNotFound) {
		h.logger.Warn(
			"key is not found",
			slog.String("key", args[0]),
		)
		return NotFoundResponse.WithErr(err).String()
	}
	if err != nil {
		h.logger.Error("failed to handle GET query", slog.Any("error", err))
		return InternalErrorResponse.WithErr(err).String()
	}
	return OKResponse.WithValue(res).String()
}

func (h *QueryHandler) handleDel(ctx context.Context, query Query) string {
	args := query.Args()
	if err := h.store.Del(ctx, args[0]); err != nil {
		h.logger.Error("failed to handle DEL query", slog.Any("error", err))
		return InternalErrorResponse.WithErr(err).String()
	}
	return OKResponse.String()
}
