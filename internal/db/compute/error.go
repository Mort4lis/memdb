package compute

import (
	"fmt"
)

var (
	parseQueryErrorPrefix = "parse_query_error"
	notFoundErrorPrefix   = "not_found"
	internalErrorPrefix   = "internal_error"
)

func formatError(prefix string, err error) string {
	return fmt.Sprintf("%s: %v", prefix, err.Error())
}
