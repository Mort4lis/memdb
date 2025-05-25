package engine

import (
	"context"
	"hash/fnv"

	dberrors "github.com/Mort4lis/memdb/internal/db/errors"
)

type Engine struct {
	shards []*shard
}

func NewEngine(partitionsNumber uint) *Engine {
	if partitionsNumber == 0 {
		partitionsNumber = 1
	}

	shards := make([]*shard, partitionsNumber)
	for i := range shards {
		shards[i] = newShard()
	}
	return &Engine{shards: shards}
}

func (e *Engine) Set(_ context.Context, key, value string) error {
	sh := e.getShardByKey(key)
	sh.Set(key, value)
	return nil
}

func (e *Engine) Get(_ context.Context, key string) (string, error) {
	sh := e.getShardByKey(key)
	value, ok := sh.Get(key)
	if !ok {
		return "", dberrors.ErrNotFound
	}
	return value, nil
}

func (e *Engine) Del(_ context.Context, key string) error {
	sh := e.getShardByKey(key)
	sh.Del(key)
	return nil
}

func (e *Engine) getShardByKey(key string) *shard {
	hasher := fnv.New32a()
	hasher.Write([]byte(key))
	return e.shards[int(hasher.Sum32())%len(e.shards)]
}
