package ratelimit

import "context"

// Algorithm 限流算法类型。
type Algorithm string

const (
	// SlidingWindow 滑动窗口：窗口内计数，边界平滑，无突发。
	SlidingWindow Algorithm = "sliding_window"
	// TokenBucket 令牌桶：恒定速率放行，允许短时突发。
	TokenBucket Algorithm = "token_bucket"
)

// 默认限流参数。
const (
	DefaultLimit  = 100 // 默认窗口内最大请求数 / 令牌桶容量
	DefaultWindow = 60  // 默认窗口时长（秒）
)

// Config 限流参数，内存后端与 Redis 后端共用。
type Config struct {
	Algorithm Algorithm `mapstructure:"algorithm"` // 限流算法，默认 SlidingWindow
	Limit     int       `mapstructure:"limit"`     // 窗口内最大请求数 / 令牌桶容量
	Window    int       `mapstructure:"window"`    // 窗口时长（秒）
}

// Normalize 补全 Config 默认值，供各后端构造限流器时调用。
func Normalize(cfg Config) Config {
	if cfg.Algorithm == "" {
		cfg.Algorithm = SlidingWindow
	}
	if cfg.Limit <= 0 {
		cfg.Limit = DefaultLimit
	}
	if cfg.Window <= 0 {
		cfg.Window = DefaultWindow
	}
	return cfg
}

// RateLimiter 限流器：判断 key 本次请求是否放行。
//
// 内存实现按 key（如客户端 IP）本地计数；Redis 实现跨实例统一计数，
// 供集群场景使用。实现方应保证 Allow 并发安全。
type RateLimiter interface {
	Allow(ctx context.Context, key string) bool
}
