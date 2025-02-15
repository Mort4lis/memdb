package concurrency

import (
	"context"
)

func WithContextCheck(ctx context.Context, action func() error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if action != nil {
		return action()
	}
	return nil
}
