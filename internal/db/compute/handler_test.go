package compute

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	dberrors "github.com/Mort4lis/memdb/internal/db/errors"
)

var errUnexpected = errors.New("unexpected")

func TestQueryHandler_Handle(t *testing.T) {
	testCases := []struct {
		name       string
		request    string
		mockSetup  func(store *MockStorage)
		wantResult string
	}{
		{
			name:    "get: ok",
			request: "GET key",
			mockSetup: func(store *MockStorage) {
				store.On("Get", mock.Anything, "key").Return("val", nil)
			},
			wantResult: "[ok] val",
		},
		{
			name:    "get: not found",
			request: "GET key",
			mockSetup: func(store *MockStorage) {
				store.On("Get", mock.Anything, "key").Return("", dberrors.ErrNotFound)
			},
			wantResult: "[not_found] key is not found",
		},
		{
			name:    "get: internal server error",
			request: "GET key",
			mockSetup: func(store *MockStorage) {
				store.On("Get", mock.Anything, "key").Return("", errUnexpected)
			},
			wantResult: "[internal_error] unexpected",
		},
		{
			name:    "set: ok",
			request: "SET key val",
			mockSetup: func(store *MockStorage) {
				store.On("Set", mock.Anything, "key", "val").Return(nil)
			},
			wantResult: "[ok]",
		},
		{
			name:    "set: internal server error",
			request: "SET key val",
			mockSetup: func(store *MockStorage) {
				store.On("Set", mock.Anything, "key", "val").Return(errUnexpected)
			},
			wantResult: "[internal_error] unexpected",
		},
		{
			name:    "del: ok",
			request: "DEL key",
			mockSetup: func(store *MockStorage) {
				store.On("Del", mock.Anything, "key").Return(nil)
			},
			wantResult: "[ok]",
		},
		{
			name:    "del: internal server error",
			request: "DEL key",
			mockSetup: func(store *MockStorage) {
				store.On("Del", mock.Anything, "key").Return(errUnexpected)
			},
			wantResult: "[internal_error] unexpected",
		},
		{
			name:       "parse error",
			request:    "UNKNOWN t1 t2",
			wantResult: "[parse_query_error] unsupport command UNKNOWN",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewMockStorage(t)
			if tc.mockSetup != nil {
				tc.mockSetup(store)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			gotResult := NewQueryHandler(slog.Default(), store).Handle(ctx, tc.request)
			assert.Equal(t, tc.wantResult, gotResult)
		})
	}
}
