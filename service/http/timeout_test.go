package service

import (
	"testing"

	serverConfig "github.com/hecc-blot/framework/config/http"
	"github.com/stretchr/testify/assert"
)

// TestResolveTimeouts："默认值 / 显式关闭"的语义边界 —— 这是配置层最容易出错的地方。
func TestResolveTimeouts(t *testing.T) {
	cases := []struct {
		name                          string
		cfg                           serverConfig.Config
		wantRead, wantWrite, wantIdle int
	}{
		{
			name:      "未配置：全部用默认",
			cfg:       serverConfig.Config{},
			wantRead:  30,
			wantWrite: 30,
			wantIdle:  60,
		},
		{
			name:      "显式配置：原样生效",
			cfg:       serverConfig.Config{ReadTimeout: 10, WriteTimeout: 15, IdleTimeout: 20},
			wantRead:  10,
			wantWrite: 15,
			wantIdle:  20,
		},
		{
			name:      "write_timeout = -1：不设写超时（read/idle 不受影响）",
			cfg:       serverConfig.Config{WriteTimeout: -1},
			wantRead:  30,
			wantWrite: 0,
			wantIdle:  60,
		},
		{
			name:      "write_timeout = 0：用默认 30，而不是关闭",
			cfg:       serverConfig.Config{WriteTimeout: 0},
			wantRead:  30,
			wantWrite: 30,
			wantIdle:  60,
		},
		{
			name:      "read/idle 的负值回落到默认值",
			cfg:       serverConfig.Config{ReadTimeout: -5, IdleTimeout: -1},
			wantRead:  30,
			wantWrite: 30,
			wantIdle:  60,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := c.cfg
			read, write, idle := resolveTimeouts(&cfg)
			assert.Equal(t, c.wantRead, read, "read")
			assert.Equal(t, c.wantWrite, write, "write")
			assert.Equal(t, c.wantIdle, idle, "idle")
		})
	}
}
