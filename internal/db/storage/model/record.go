package model

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"google.golang.org/protobuf/encoding/protodelim"
)

//go:generate protoc --go_out=. --go_opt=paths=source_relative record.proto

// EncodeRecords encodes a list of Record instances into a buffer using the protocol buffer delimited format.
func EncodeRecords(rs []*Record, buf *bytes.Buffer) error {
	for i, r := range rs {
		if _, err := protodelim.MarshalTo(buf, r); err != nil {
			return fmt.Errorf("encode record[%d]: %w", i, err)
		}
	}
	return nil
}

// DecodeRecords reads and decodes a series of Record messages from the provided buffer using protobuf delimited encoding.
// It returns a slice of pointers to Record objects or an error if decoding fails.
func DecodeRecords(buf *bytes.Buffer) ([]*Record, error) {
	var rs []*Record

	var i int
	for {
		var r Record
		err := protodelim.UnmarshalFrom(buf, &r)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode record[%d]: %w", i, err)
		}

		rs = append(rs, &r)
		i++
	}
	return rs, nil
}
