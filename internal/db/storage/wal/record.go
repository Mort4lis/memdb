package wal

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/Mort4lis/memdb/internal/db/compute"
	"github.com/Mort4lis/memdb/internal/pkg/concurrency"
)

type Record struct {
	LSN       int64             `json:"lsn"`
	CommandID compute.CommandID `json:"cmd"`
	Args      []string          `json:"args"`
	promise   concurrency.PromiseError
}

func (r *Record) Encode(buf *bytes.Buffer) error {
	if err := json.NewEncoder(buf).Encode(&r); err != nil {
		return fmt.Errorf("json encode: %w", err)
	}
	return nil
}

func (r *Record) Decode(buf *bytes.Buffer) error {
	if err := json.NewDecoder(buf).Decode(&r); err != nil {
		return fmt.Errorf("json decode: %w", err)
	}
	return nil
}
