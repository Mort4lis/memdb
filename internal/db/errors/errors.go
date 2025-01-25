package errors

import (
	"errors"
)

var (
	ErrNotFound = errors.New("key is not found")
	ErrInternal = errors.New("internal server error")
)
