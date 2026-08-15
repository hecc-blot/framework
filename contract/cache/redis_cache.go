package cache

import (
	"context"
)

type IRedisCache interface {
	IBaseCache

	HSet(ctx context.Context, key string, values ...interface{}) error
	HGet(ctx context.Context, key, field string) (string, error)
	HDel(ctx context.Context, key string, fields ...string) error

	Close() error
}
