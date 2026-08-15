package cache

import (
	"context"
	"time"
)

type IBaseCache interface {
	// Set 将value写入缓存
	Set(ctx context.Context, key string, val interface{}, expire time.Duration) error
	// Get 根据key值获取value
	Get(ctx context.Context, key string) (interface{}, error)
	// Del 删除key值
	Del(ctx context.Context, key string) error
	// Exists 判断key是否存在
	Exists(ctx context.Context, key string) (bool, error)
}
