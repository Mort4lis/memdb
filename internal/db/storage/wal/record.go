package wal

import (
	"github.com/Mort4lis/memdb/internal/db/compute"
	"github.com/Mort4lis/memdb/internal/pkg/concurrency"
)

type Record struct {
	LSN       int64             `json:"lsn"`
	CommandID compute.CommandID `json:"cmd"`
	Args      []string          `json:"args"`
	promise   concurrency.PromiseError
}
