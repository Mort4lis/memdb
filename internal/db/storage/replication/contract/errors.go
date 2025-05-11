package contract

import (
	"fmt"
	"log/slog"

	"google.golang.org/protobuf/proto"
)

var (
	ErrInternal = &Error{Code: ErrorCode_INTERNAL, Text: "Internal server error"}
	ErrNotFound = &Error{Code: ErrorCode_NOT_FOUND, Text: "Not found"}
)

func (e *Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code.String(), e.Text)
}

func RespondError(e *Error) []byte {
	resp := &NextSegmentResponse{Error: e}

	b, err := proto.Marshal(resp)
	if err != nil {
		slog.Default().Error("Failed to marshal response", slog.Any("error", err))
		return nil
	}
	return b
}
