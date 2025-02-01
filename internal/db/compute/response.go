package compute

import (
	"fmt"
)

type Response struct {
	kind  string
	value string
	err   error
}

func (r Response) WithValue(v string) Response {
	return Response{kind: r.kind, value: v}
}

func (r Response) WithErr(err error) Response {
	return Response{kind: r.kind, err: err}
}

func (r Response) String() string {
	if r.err != nil {
		return fmt.Sprintf("[%s] %v", r.kind, r.err)
	}
	if r.value != "" {
		return fmt.Sprintf("[%s] %s", r.kind, r.value)
	}
	return fmt.Sprintf("[%s]", r.kind)
}

var (
	OKResponse = Response{kind: "ok"}

	NotFoundResponse        = Response{kind: "not_found"}
	ParseQueryErrorResponse = Response{kind: "parse_query_error"}
	InternalErrorResponse   = Response{kind: "internal_error"}
)
